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
	Alpha:     1.0, // Fully opaque by default
}

//go:embed resources/default.png
var defaultTextureFS embed.FS

// MaterialGroup represents a submesh with a single material
type MaterialGroup struct {
	Material   *Material // Material for this group
	IndexStart int32     // Starting index in the index buffer
	IndexCount int32     // Number of indices for this group
}

type Model struct {
	// HOT DATA - Accessed every frame in render loop (keep in first cache lines)
	ModelMatrix             mgl32.Mat4     // Transformation matrix - used every frame
	Position                mgl32.Vec3     // Position in world space
	Scale                   mgl32.Vec3     // Scale factors
	Rotation                mgl32.Quat     // Rotation quaternion
	Material                *Material      // Material properties pointer
	VAO                     uint32         // Vertex Array Object
	VBO                     uint32         // Vertex Buffer Object
	EBO                     uint32         // Element Buffer Object
	InstanceVBO             uint32         // Instance Vertex Buffer Object (for instanced rendering)
	InstanceVBOCapacity     int            // GPU buffer capacity in bytes for buffer reuse optimization
	InstanceCount           int            // Number of instances
	IsDirty                 bool           // Needs recalculation flag
	IsInstanced             bool           // Instanced rendering flag
	InstanceMatricesUpdated bool           // Flag to track if matrices need GPU upload
	
	// MEDIUM DATA - Conditional/periodic access
	BoundingSphereCenter    mgl32.Vec3     // For frustum culling
	BoundingSphereRadius    float32        // For frustum culling
	IsBatched               bool           // Batching flag
	Shader                  Shader         // Custom shader for this model
	CustomUniforms          map[string]interface{} // Custom uniforms for this model
	Metadata                map[string]interface{} // General metadata for editor/game logic
	InstanceModelMatrices   []mgl32.Mat4   // Instance model matrices (bulk data)
	
	// COLD DATA - Initialization only or rarely accessed
	Id                      int            // Model identifier
	Name                    string         // Model name
	SourcePath              string         // Original file path (for scene serialization)
	Vertices                []float32      // Vertex position data
	Indices                 []uint32       // Index data (Vulkan)
	Normals                 []float32      // Normal vectors
	Faces                   []int32        // Face indices (OpenGL)
	TextureCoords           []float32      // Texture coordinates
	InterleavedData         []float32      // Combined vertex data
	MaterialGroups          []MaterialGroup // For multi-material models
	vertexBuffer            vk.Buffer      // Vulkan vertex buffer
	vertexMemory            vk.DeviceMemory // Vulkan vertex memory
	indexBuffer             vk.Buffer      // Vulkan index buffer
	indexMemory             vk.DeviceMemory // Vulkan index memory
	// InstanceBoundingBox     [2]mgl32.Vec3 // Cached bounding box for instances [min, max]
}

type Material struct {
	// HOT DATA - Accessed every render call for shading calculations
	DiffuseColor  [3]float32 // Base color for lighting
	SpecularColor [3]float32 // Specular highlight color
	Shininess     float32    // Specular exponent
	Metallic      float32    // 0.0 = dielectric, 1.0 = metallic
	Roughness     float32    // 0.0 = mirror, 1.0 = completely rough
	Exposure      float32    // HDR exposure control
	Alpha         float32    // Transparency (0.0 = transparent, 1.0 = opaque)
	TextureID     uint32     // OpenGL texture ID
	
	// COLD DATA - Rarely accessed (identification only)
	Name          string     // Material name for debugging
	TexturePath   string     // Path to texture file (loaded lazily when OpenGL is ready)
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

	// Declare variables for both paths
	var center mgl32.Vec3
	var maxDistanceSq float32

	// For instanced models, create a simple but effective bounding sphere
	if m.IsInstanced && len(m.InstanceModelMatrices) > 0 {
		// Simple approach: use first and last instance to estimate bounds
		// This is generic and works for any instanced model type

		firstPos := m.InstanceModelMatrices[0].Col(3).Vec3()
		lastPos := m.InstanceModelMatrices[len(m.InstanceModelMatrices)-1].Col(3).Vec3()

		// Calculate center between first and last instance
		center = firstPos.Add(lastPos).Mul(0.5)

		// Calculate radius to cover both instances plus some margin
		distance := firstPos.Sub(lastPos).Len()
		maxDistanceSq = (distance*0.5 + 100.0) * (distance*0.5 + 100.0) // Add margin

		m.BoundingSphereCenter = center
		m.BoundingSphereRadius = float32(math.Sqrt(float64(maxDistanceSq)))
		return
	}

	// For non-instanced models, use the original calculation

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
	// Matrix multiplication order: translation * rotation * scale (correct OpenGL convention)
	// Matrices are multiplied right-to-left: T * R * S transforms vertices as: scale first, then rotate, then translate
	scaleMatrix := mgl32.Scale3D(m.Scale[0], m.Scale[1], m.Scale[2])
	rotationMatrix := m.Rotation.Mat4()
	translationMatrix := mgl32.Translate3D(m.Position[0], m.Position[1], m.Position[2])
	// Combine the transformations: ModelMatrix = translation * rotation * scale (TRS order)
	m.ModelMatrix = translationMatrix.Mul4(rotationMatrix).Mul4(scaleMatrix)
	
	// For instanced models, we do NOT update instance matrices here.
	// The shader now handles hierarchical transformation (model * instance).
	// This allows moving the entire group of instances by changing the Model's position/scale/rotation.
	
	if FrustumCullingEnabled {
		m.CalculateBoundingSphere()
	}
}

// CalculateModelMatrix calculates the transformation matrix for a model
func (m *Model) calculateModelMatrix() {
	// Correct transformation order: Translation * Rotation * Scale (TRS)
	scaleMatrix := mgl32.Scale3D(m.Scale.X(), m.Scale.Y(), m.Scale.Z())
	rotationMatrix := m.Rotation.Mat4()
	translationMatrix := mgl32.Translate3D(m.Position.X(), m.Position.Y(), m.Position.Z())
	
	// Apply transformations in correct order: TRS
	m.ModelMatrix = translationMatrix.Mul4(rotationMatrix).Mul4(scaleMatrix)
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

// ensureMaterial creates a new material instance if one doesn't exist or fixes incomplete materials
func (m *Model) ensureMaterial() {
	if m.Material == nil {
		logger.Log.Info("Creating new default material")
		// Create a new material instance instead of sharing DefaultMaterial
		m.Material = &Material{
			Name:          "default",
			DiffuseColor:  [3]float32{1.0, 1.0, 1.0},
			SpecularColor: [3]float32{1.0, 1.0, 1.0},
			Shininess:     32.0,
			TextureID:     0,
			Metallic:      0.0,
			Roughness:     0.5,
			Exposure:      1.0,
			Alpha:         1.0,
		}
	} else if m.Material == DefaultMaterial {
		// CRITICAL: Create a copy if pointing to shared DefaultMaterial
		// This prevents multiple models from sharing the same material instance
		logger.Log.Info("Creating unique material copy (was sharing DefaultMaterial)")
		m.Material = &Material{
			Name:          m.Material.Name,
			DiffuseColor:  m.Material.DiffuseColor,
			SpecularColor: m.Material.SpecularColor,
			Shininess:     m.Material.Shininess,
			TextureID:     m.Material.TextureID,
			Metallic:      m.Material.Metallic,
			Roughness:     m.Material.Roughness,
			Exposure:      m.Material.Exposure,
			Alpha:         m.Material.Alpha,
			TexturePath:   m.Material.TexturePath,
		}
	} else {
		// Fix incomplete materials (from MTL files that only set Name)
		// Only fix if ALL critical values are zero, indicating incomplete initialization
		if m.Material.Alpha == 0.0 && m.Material.Roughness == 0.0 && m.Material.Exposure == 0.0 && m.Material.Shininess == 0.0 {
			logger.Log.Info("Fixing incomplete material: " + m.Material.Name)
			// Set proper defaults for incomplete materials
			m.Material.Roughness = 0.5  // Medium roughness
			m.Material.Exposure = 1.0   // Standard exposure
			m.Material.Alpha = 1.0      // Fully opaque
			m.Material.Shininess = 32.0 // Default shininess
		}
	}
}

func (m *Model) SetDiffuseColor(r, g, b float32) {
	m.ensureMaterial()
	m.Material.DiffuseColor = [3]float32{r, g, b}
	// Material changes don't affect transformation matrix, so don't set IsDirty
}

func (m *Model) SetSpecularColor(r, g, b float32) {
	m.ensureMaterial()
	m.Material.SpecularColor = [3]float32{r, g, b}
	// Material changes don't affect transformation matrix, so don't set IsDirty
}

// Modern PBR material setters
func (m *Model) SetMaterialPBR(metallic, roughness float32) {
	m.ensureMaterial()
	m.Material.Metallic = metallic
	m.Material.Roughness = roughness
	// Material changes don't affect transformation matrix, so don't set IsDirty
}

func (m *Model) SetExposure(exposure float32) {
	m.ensureMaterial()
	m.Material.Exposure = exposure
	// Material changes don't affect transformation matrix, so don't set IsDirty
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

// Set transparency/alpha
func (m *Model) SetAlpha(alpha float32) {
	m.ensureMaterial()
	m.Material.Alpha = alpha
	// Material changes don't affect transformation matrix, so don't set IsDirty
}

// Glass material preset
func (m *Model) SetGlass(r, g, b, alpha float32) {
	m.SetDiffuseColor(r, g, b)
	m.SetMaterialPBR(0.0, 0.05) // Non-metallic, very smooth
	m.SetAlpha(alpha)
	m.SetExposure(1.2) // Slightly higher exposure for glass
}

// Transparent plastic
func (m *Model) SetTransparentPlastic(r, g, b, alpha, roughness float32) {
	m.SetPlasticMaterial(r, g, b, roughness)
	m.SetAlpha(alpha)
}

func (m *Model) SetTexture(texturePath string) {
	// Store texture path - it will be loaded when model is added to renderer
	// This avoids needing a renderer instance here
	if m.Material == nil {
		logger.Log.Info("Setting default material for texture")
		m.Material = &Material{
			Name:          "default",
			DiffuseColor:  [3]float32{1.0, 1.0, 1.0},
			SpecularColor: [3]float32{1.0, 1.0, 1.0},
			Shininess:     32.0,
			Metallic:      0.0,
			Roughness:     0.5,
			Exposure:      1.0,
			Alpha:         1.0,
		}
	}
	m.Material.TexturePath = texturePath
	logger.Log.Debug("Texture path set for model",
		zap.String("path", texturePath),
		zap.String("material", m.Material.Name))
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

		// Apply TRS transformations in correct order
		m.InstanceModelMatrices[index] = translationMatrix.Mul4(rotationMatrix).Mul4(scaleMatrix)

		// Mark instance matrices as needing GPU update
		m.InstanceMatricesUpdated = true
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
		Position:        mgl32.Vec3{0, 0, 0},       // Initialize position
		Rotation:        mgl32.Quat{},              // Initialize rotation (zero quat = identity matrix)
		Scale:           mgl32.Vec3{1.0, 1.0, 1.0}, // Initialize scale
		Vertices:        flattenVertices(vertices),
		Faces:           indices,
		InterleavedData: interleavedData,
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
