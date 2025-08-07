package renderer

import (
	"Gopher3D/internal/logger"
	"bytes"
	"embed"
	"image"
	"math"

	"github.com/go-gl/mathgl/mgl32"
	vk "github.com/vulkan-go/vulkan"
	"go.uber.org/zap"
)

// DefaultMaterial provides a basic material to fall back on
var DefaultMaterial = &Material{
	Name:          "default",
	DiffuseColor:  [3]float32{1.0, 1.0, 1.0}, // White color
	SpecularColor: [3]float32{1.0, 1.0, 1.0},
	Shininess:     32.0,
	TextureID:     0,
	// Modern PBR defaults
	Metallic:  0.0, // Non-metallic by default
	Roughness: 0.5, // Medium roughness
	Exposure:  1.0, // Standard exposure
}

//go:embed resources/default.png
var defaultTextureFS embed.FS

type Model struct {
	Id                    int
	Name                  string
	Position              mgl32.Vec3
	Scale                 mgl32.Vec3
	Rotation              mgl32.Quat
	Vertices              []float32
	Indices               []uint32
	vertexBuffer          vk.Buffer
	vertexMemory          vk.DeviceMemory
	indexBuffer           vk.Buffer
	indexMemory           vk.DeviceMemory
	Normals               []float32
	Faces                 []int32
	TextureCoords         []float32
	InterleavedData       []float32
	Material              *Material
	VAO                   uint32 // Vertex Array Object
	VBO                   uint32 // Vertex Buffer Object
	EBO                   uint32 // Element Buffer Object
	ModelMatrix           mgl32.Mat4
	BoundingSphereCenter  mgl32.Vec3
	BoundingSphereRadius  float32
	IsDirty               bool
	IsBatched             bool
	IsInstanced           bool
	InstanceCount         int
	InstanceModelMatrices []mgl32.Mat4           // Instance model matrices
	Shader                Shader                 // Custom shader for this model
	CustomUniforms        map[string]interface{} // Custom uniforms for this model
}

type Material struct {
	Name          string
	DiffuseColor  [3]float32
	SpecularColor [3]float32
	Shininess     float32
	TextureID     uint32 // OpenGL texture ID
	// Modern PBR properties
	Metallic  float32 // 0.0 = dielectric, 1.0 = metallic
	Roughness float32 // 0.0 = mirror, 1.0 = completely rough
	Exposure  float32 // HDR exposure control
}

func (m *Model) X() float32 {
	return m.Position[0]
}

func (m *Model) Y() float32 {
	return m.Position[1]
}

func (m *Model) Z() float32 {
	return m.Position[2]
}

func (m *Model) Rotate(angleX, angleY, angleZ float32) {
	if m.Rotation == (mgl32.Quat{}) {
		m.Rotation = mgl32.QuatIdent()
	}
	rotationX := mgl32.QuatRotate(mgl32.DegToRad(angleX), mgl32.Vec3{1, 0, 0})
	rotationY := mgl32.QuatRotate(mgl32.DegToRad(angleY), mgl32.Vec3{0, 1, 0})
	rotationZ := mgl32.QuatRotate(mgl32.DegToRad(angleZ), mgl32.Vec3{0, 0, 1})
	m.Rotation = m.Rotation.Mul(rotationX).Mul(rotationY).Mul(rotationZ)
	m.updateModelMatrix()
	m.IsDirty = true
}

// SetPosition sets the position of the model
func (m *Model) SetPosition(x, y, z float32) {
	m.Position = mgl32.Vec3{x, y, z}
	m.updateModelMatrix()
	m.IsDirty = true
}

func (m *Model) SetScale(x, y, z float32) {
	m.Scale = mgl32.Vec3{x, y, z}
	m.updateModelMatrix()
	m.IsDirty = true
}

func (m *Model) CalculateBoundingSphere() {
	if !FrustumCullingEnabled {
		return
	}
	var center mgl32.Vec3
	var maxDistanceSq float32

	numVertices := len(m.Vertices) / 3 // Assuming 3 float32s per vertex
	for i := 0; i < numVertices; i++ {
		// Extracting vertex from the flat array
		vertex := mgl32.Vec3{m.Vertices[i*3], m.Vertices[i*3+1], m.Vertices[i*3+2]}
		transformedVertex := ApplyModelTransformation(vertex, m.Position, m.Scale, m.Rotation)
		center = center.Add(transformedVertex)
	}
	center = center.Mul(1.0 / float32(numVertices))

	for i := 0; i < numVertices; i++ {
		vertex := mgl32.Vec3{m.Vertices[i*3], m.Vertices[i*3+1], m.Vertices[i*3+2]}
		transformedVertex := ApplyModelTransformation(vertex, m.Position, m.Scale, m.Rotation)
		distanceSq := transformedVertex.Sub(center).LenSqr()
		if distanceSq > maxDistanceSq {
			maxDistanceSq = distanceSq
		}
	}

	m.BoundingSphereCenter = center
	m.BoundingSphereRadius = float32(math.Sqrt(float64(maxDistanceSq)))
}

func (m *Model) updateModelMatrix() {
	// Matrix multiplication order: scale -> rotate -> translate
	scaleMatrix := mgl32.Scale3D(m.Scale[0], m.Scale[1], m.Scale[2])
	rotationMatrix := m.Rotation.Mat4()
	translationMatrix := mgl32.Translate3D(m.Position[0], m.Position[1], m.Position[2])
	// Combine the transformations: ModelMatrix = translation * rotation * scale
	m.ModelMatrix = translationMatrix.Mul4(rotationMatrix).Mul4(scaleMatrix)
	// If the model is instanced, update the instance matrices automatically
	if m.IsInstanced && len(m.InstanceModelMatrices) > 0 {
		for i := 0; i < m.InstanceCount; i++ {
			instancePosition := m.InstanceModelMatrices[i].Col(3).Vec3() // Retrieve the instance position
			instanceMatrix := mgl32.Translate3D(instancePosition.X(), instancePosition.Y(), instancePosition.Z()).
				Mul4(rotationMatrix).Mul4(scaleMatrix)
			m.InstanceModelMatrices[i] = instanceMatrix
		}
	}
	if FrustumCullingEnabled {
		m.CalculateBoundingSphere()
	}
}

// CalculateModelMatrix calculates the transformation matrix for a model
func (m *Model) calculateModelMatrix() {
	// Start with an identity matrix
	m.ModelMatrix = mgl32.Ident4()

	// Apply scaling, rotation, and translation in sequence without extra matrix allocations
	m.ModelMatrix = m.ModelMatrix.Mul4(mgl32.Scale3D(m.Scale.X(), m.Scale.Y(), m.Scale.Z()))
	m.ModelMatrix = m.ModelMatrix.Mul4(m.Rotation.Mat4())
	m.ModelMatrix = m.ModelMatrix.Mul4(mgl32.Translate3D(m.Position.X(), m.Position.Y(), m.Position.Z()))
}

// Aux functions, maybe I need to move them to another package
func ApplyModelTransformation(vertex, position, scale mgl32.Vec3, rotation mgl32.Quat) mgl32.Vec3 {
	// Apply scaling
	scaledVertex := mgl32.Vec3{vertex[0] * scale[0], vertex[1] * scale[1], vertex[2] * scale[2]}

	// Apply rotation
	// Note: mgl32.Quat doesn't directly multiply with Vec3, so we convert it to a Mat4 first
	rotatedVertex := rotation.Mat4().Mul4x1(scaledVertex.Vec4(1)).Vec3()

	// Apply translation
	transformedVertex := rotatedVertex.Add(position)

	return transformedVertex
}

func (m *Model) SetDiffuseColor(r, g, b float32) {
	if m.Material == nil {
		logger.Log.Info("Setting default material")
		m.Material = DefaultMaterial
	}
	m.Material.DiffuseColor = [3]float32{r, g, b}
	m.IsDirty = true // Mark the model as dirty for re-rendering
}

func (m *Model) SetSpecularColor(r, g, b float32) {
	if m.Material == nil {
		logger.Log.Info("Setting default material")
		m.Material = DefaultMaterial
	}
	m.Material.SpecularColor = [3]float32{r, g, b}
	m.IsDirty = true // Mark the model as dirty for re-rendering
}

// Modern PBR material setters
func (m *Model) SetMaterialPBR(metallic, roughness float32) {
	if m.Material == nil {
		logger.Log.Info("Setting default material")
		m.Material = DefaultMaterial
	}
	m.Material.Metallic = metallic
	m.Material.Roughness = roughness
	m.IsDirty = true
}

func (m *Model) SetExposure(exposure float32) {
	if m.Material == nil {
		logger.Log.Info("Setting default material")
		m.Material = DefaultMaterial
	}
	m.Material.Exposure = exposure
	m.IsDirty = true
}

// Preset materials for easy use
func (m *Model) SetMetallicMaterial(r, g, b, roughness float32) {
	m.SetDiffuseColor(r, g, b)
	m.SetMaterialPBR(1.0, roughness) // Fully metallic
}

func (m *Model) SetPlasticMaterial(r, g, b, roughness float32) {
	m.SetDiffuseColor(r, g, b)
	m.SetMaterialPBR(0.0, roughness) // Non-metallic
}

func (m *Model) SetRoughMetal(r, g, b float32) {
	m.SetMetallicMaterial(r, g, b, 0.8) // Rough metal
}

func (m *Model) SetPolishedMetal(r, g, b float32) {
	m.SetMetallicMaterial(r, g, b, 0.1) // Polished metal
}

func (m *Model) SetMatte(r, g, b float32) {
	m.SetPlasticMaterial(r, g, b, 0.9) // Matte surface
}

func (m *Model) SetGlossy(r, g, b float32) {
	m.SetPlasticMaterial(r, g, b, 0.2) // Glossy surface
}

func (m *Model) SetTexture(texturePath string) {
	// TODO: Use THE CONFIG to know which renderer to use
	textureID, err := (&OpenGLRenderer{}).LoadTexture(texturePath)
	if err != nil {
		logger.Log.Error("Failed to load texture", zap.String("path", texturePath), zap.Error(err))
		return
	}

	if m.Material == nil {
		logger.Log.Info("Setting default material")
		m.Material = DefaultMaterial

	}
	m.Material.TextureID = textureID
}

func SetDefaultTexture(RendererAPI Render) {
	// Read the embedded texture
	textureBytes, err := defaultTextureFS.ReadFile("resources/default.png")
	if err != nil {
		logger.Log.Error("Failed to read embedded default texture", zap.Error(err))
		return
	}

	// Create an image from the texture bytes
	img, _, err := image.Decode(bytes.NewReader(textureBytes))
	if err != nil {
		logger.Log.Error("Failed to decode embedded default texture", zap.Error(err))
		return
	}

	// Convert the image to a texture and set it as the default texture
	// TODO: It should use the renderer API to create the texture and not an OpenGL-specific function
	textureID, err := RendererAPI.CreateTextureFromImage(img)
	if err != nil {
		logger.Log.Error("Failed to create texture from embedded default image", zap.Error(err))
		return
	}

	DefaultMaterial.TextureID = textureID
}

func (m *Model) SetInstanceCount(count int) {
	m.InstanceCount = count
	m.InstanceModelMatrices = make([]mgl32.Mat4, count)
}

func (m *Model) SetInstancePosition(index int, position mgl32.Vec3) {
	if index >= 0 && index < len(m.InstanceModelMatrices) {
		// Combine translation, rotation (if needed), and scaling for each instance
		scaleMatrix := mgl32.Scale3D(m.Scale[0], m.Scale[1], m.Scale[2])
		rotationMatrix := m.Rotation.Mat4()
		translationMatrix := mgl32.Translate3D(position.X(), position.Y(), position.Z())

		// Apply translation * rotation * scale transformations
		m.InstanceModelMatrices[index] = translationMatrix.Mul4(rotationMatrix).Mul4(scaleMatrix)
	}
}

func CreateModel(vertices []mgl32.Vec3, indices []int32) *Model {
	interleavedData := make([]float32, 0, len(vertices)*8) // Assuming 3 for position, 2 for texture, 3 for normals

	for _, v := range vertices {
		// Add vertex position (X, Y, Z)
		interleavedData = append(interleavedData, v.X(), v.Y(), v.Z())

		// Add placeholder texture coordinates (U, V)
		interleavedData = append(interleavedData, 0.0, 0.0)

		// Add placeholder normals (X, Y, Z)
		interleavedData = append(interleavedData, 0.0, 1.0, 0.0)
	}

	return &Model{
		Vertices:        flattenVertices(vertices),
		Faces:           indices,
		InterleavedData: interleavedData,
		Scale:           mgl32.Vec3{1.0, 1.0, 1.0},
	}
}

// Helper to flatten Vec3 array
func flattenVertices(vertices []mgl32.Vec3) []float32 {
	flat := make([]float32, 0, len(vertices)*3)
	for _, v := range vertices {
		flat = append(flat, v.X(), v.Y(), v.Z())
	}
	return flat
}
