package loader

import (
	"Gopher3D/internal/logger"
	"Gopher3D/internal/renderer"
	"bufio"
	"errors"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-gl/mathgl/mgl32"
	"go.uber.org/zap"
)

func LoadObjectWithPath(Path string, recalculateNormals bool) (*renderer.Model, error) {
	model, err := LoadModel(Path, recalculateNormals)
	return model, err
}

// LoadObjectInstance loads a model with instancing enabled
func LoadObjectInstance(Path string, recalculateNormals bool, instanceCount int) (*renderer.Model, error) {
	model, err := LoadModel(Path, recalculateNormals)
	if err != nil {
		return nil, err
	}

	model.IsInstanced = true
	model.InstanceCount = instanceCount
	// Pass instance data to renderer instead of modifying the model
	model.InstanceModelMatrices = make([]mgl32.Mat4, instanceCount)
	for i := 0; i < instanceCount; i++ {
		// Initialize with identity matrices, set positions later
		model.InstanceModelMatrices[i] = mgl32.Ident4()
	}
	return model, nil
}

func LoadPlane(gridSize int, gridSpacing float32) (*renderer.Model, error) {
	if gridSize < 2 {
		return nil, errors.New("gridSize must be at least 2")
	}

	vertices := make([]mgl32.Vec3, 0, gridSize*gridSize)
	indices := make([]int32, 0, (gridSize-1)*(gridSize-1)*6)

	// Generate vertices
	for x := 0; x < gridSize; x++ {
		for z := 0; z < gridSize; z++ {
			vertices = append(vertices, mgl32.Vec3{
				float32(x) * gridSpacing,
				0, // Initial height is 0
				float32(z) * gridSpacing,
			})
		}
	}

	// Generate indices for triangles
	for x := 0; x < gridSize-1; x++ {
		for z := 0; z < gridSize-1; z++ {
			topLeft := int32(x*gridSize + z)
			topRight := topLeft + 1
			bottomLeft := int32((x+1)*gridSize + z)
			bottomRight := bottomLeft + 1

			indices = append(indices, topLeft, bottomLeft, bottomRight, topLeft, bottomRight, topRight)
		}
	}

	// Create a model from the vertices and indices
	model := renderer.CreateModel(vertices, indices)
	return model, nil
}

// LoadWaterSurface creates an optimized water surface with configurable resolution
// for better performance and visual quality control
func LoadWaterSurface(size float32, centerX, centerZ float32, resolution int) (*renderer.Model, error) {
	// Validate resolution parameter
	if resolution < 16 {
		resolution = 16 // Minimum resolution for basic functionality
	}
	if resolution > 8192 {
		resolution = 8192 // Much higher maximum resolution for detailed water
	}

	baseResolution := resolution

	vertices := make([]mgl32.Vec3, 0, baseResolution*baseResolution)
	indices := make([]int32, 0, (baseResolution-1)*(baseResolution-1)*6)

	stepSize := size / float32(baseResolution-1)
	startX := centerX - size*0.5
	startZ := centerZ - size*0.5

	// Generate vertices in a simple grid pattern
	for x := 0; x < baseResolution; x++ {
		for z := 0; z < baseResolution; z++ {
			posX := startX + float32(x)*stepSize
			posZ := startZ + float32(z)*stepSize

			vertices = append(vertices, mgl32.Vec3{posX, 0, posZ})
		}
	}

	// Generate triangle indices in the standard way
	for x := 0; x < baseResolution-1; x++ {
		for z := 0; z < baseResolution-1; z++ {
			// Calculate vertex indices for this quad
			topLeft := int32(x*baseResolution + z)
			topRight := topLeft + 1
			bottomLeft := int32((x+1)*baseResolution + z)
			bottomRight := bottomLeft + 1

			// Create two triangles for each quad
			// Triangle 1: topLeft -> bottomLeft -> bottomRight
			indices = append(indices, topLeft, bottomLeft, bottomRight)
			// Triangle 2: topLeft -> bottomRight -> topRight
			indices = append(indices, topLeft, bottomRight, topRight)
		}
	}

	// Create the model
	model := renderer.CreateModel(vertices, indices)

	logger.Log.Info("Water surface created",
		zap.Int("vertices", len(vertices)),
		zap.Int("triangles", len(indices)/3),
		zap.Float32("size", size),
		zap.Int("resolution", baseResolution))

	return model, nil
}

func LoadObject() *renderer.Model {
	files, err := os.ReadDir("../obj")
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".obj") {
			model, err := LoadModel("../obj/"+file.Name(), true)
			if err != nil {
				logger.Log.Error("Could not load the obj file", zap.String("file:", file.Name()), zap.Error(err))
			}
			return model
		}
	}
	return nil
}

func LoadModel(filename string, recalculateNormals bool) (*renderer.Model, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var modelMaterials map[string]*renderer.Material
	var model *renderer.Model
	var vertices []float32
	var textureCoords []float32
	var normals []float32
	var faces []int32
	var currentMaterialName string
	// TODO: I may want to review this later
	model = &renderer.Model{}
	model.Material = renderer.DefaultMaterial
	uniqueMaterial := *model.Material
	model.Material = &uniqueMaterial
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		switch parts[0] {
		case "v":
			vertex, err := parseVertex(parts[1:])
			if err != nil {
				logger.Log.Error("Error parsing vertex: ", zap.Error(err))
				return nil, err
			}
			vertices = append(vertices, vertex...)
		case "vn":
			normal, err := parseVertex(parts[1:])
			if err != nil {
				logger.Log.Error("Error parsing normal: ", zap.Error(err))
				return nil, err
			}
			normals = append(normals, normal...)
		case "vt":
			texCoord, err := parseTextureCoordinate(parts[1:])
			if err != nil {
				logger.Log.Error("Error parsing texture coordinate: ", zap.Error(err))
				return nil, err
			}
			textureCoords = append(textureCoords, texCoord[0], texCoord[1])
		case "f":
			face, err := parseFace(parts[1:])
			if err != nil {
				logger.Log.Error("Error parsing face: ", zap.Error(err))
				return nil, err
			}
			faces = append(faces, face...)
		// MATERIALS PLACEHOLDER
		case "mtllib":
			mtlPath := filepath.Join(filepath.Dir(filename), parts[1])
			modelMaterials = LoadMaterials(mtlPath)
			// LoadMaterials always returns at least a default material, no error check needed

			// TODO: SUPPORT MULTIPLE MATERIALS
			for _, mat := range modelMaterials {
				model.Material = mat
				break // Just take the first material found
			}
		// TODO: For models with multiple parts, each possibly using a different material
		case "usemtl":
			if len(parts) >= 2 {
				currentMaterialName = parts[1]
				if material, ok := modelMaterials[currentMaterialName]; ok {
					model.Material = material
				} else {
					// TODO: Change log to Warn or Info
					logger.Log.Debug("Material not found", zap.String("Material:", currentMaterialName))
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	vertexCount := len(vertices) / 3

	for len(textureCoords)/2 < vertexCount {
		textureCoords = append(textureCoords, 0, 0)
	}
	for len(normals)/3 < vertexCount {
		normals = append(normals, 0, 0, 0)

	}

	// Some models have broken normals, so we recalculate them ourselves
	if recalculateNormals {
		normals = RecalculateNormals(vertices, faces)
	}

	interleavedData := make([]float32, 0, vertexCount*8)
	for i := 0; i < vertexCount; i++ {
		interleavedData = append(interleavedData, vertices[i*3:i*3+3]...)
		interleavedData = append(interleavedData, textureCoords[i*2:i*2+2]...)
		interleavedData = append(interleavedData, normals[i*3:i*3+3]...)
	}

	model.InterleavedData = interleavedData
	model.Vertices = vertices
	model.Faces = faces

	model.Position = [3]float32{0, 0, 0}
	model.Rotation = mgl32.Quat{}
	model.Scale = [3]float32{1, 1, 1}
	// TODO: MAYBE NOT NECCESARY
	model.CalculateBoundingSphere()
	return model, nil
}

// LoadMaterials loads material properties from a .mtl file.
func LoadMaterials(filename string) map[string]*renderer.Material {
	defaultMaterial := renderer.DefaultMaterial
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		logger.Log.Error("Error opening material file: ", zap.Error(err))
		return map[string]*renderer.Material{"default": defaultMaterial}
	}

	file, err := os.Open(filename)
	if err != nil {
		logger.Log.Error("Error opening material file: ", zap.Error(err))
		return map[string]*renderer.Material{"default": defaultMaterial}
	}
	defer file.Close()
	var currentMaterial *renderer.Material
	materials := make(map[string]*renderer.Material)
	scanner := bufio.NewScanner(file)
	defer file.Close()

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "newmtl":
			if len(fields) < 2 {
				logger.Log.Error("Malformed material line: ", zap.String("Line:", line))
				continue
			}
			currentMaterial = &renderer.Material{Name: fields[1]}
			materials[fields[1]] = currentMaterial
		case "Kd": // Diffuse color
			if len(fields) == 4 {
				currentMaterial.DiffuseColor = parseColor(fields[1:])
			}
		case "Ks": // Specular color
			if len(fields) == 4 {
				currentMaterial.SpecularColor = parseColor(fields[1:])
			}
		case "Ns": // Shininess
			if len(fields) == 2 {
				currentMaterial.Shininess = parseFloat(fields[1])
			}
			/*
				case "map_Kd": // Diffuse texture map
					if len(fields) == 2 {
						textureID := renderer.SetTexture(fields[1]) // TODO: Implement this in renderer
						currentMaterial.TextureID = textureID
					}

			*/
		}
	}

	if err := scanner.Err(); err != nil {
		// Handle error
		panic(err)
	}

	return materials
}

// parseColor parses RGB color components from a list of strings to an array of float32.
func parseColor(fields []string) [3]float32 {
	var color [3]float32
	for i, field := range fields {
		if val, err := strconv.ParseFloat(field, 32); err == nil {
			color[i] = float32(val)
		} else {
			logger.Log.Error("Error parsing color component: ", zap.Error(err))
			color[i] = 0.0 // Defaulting to 0 in case of error
		}
	}
	return color
}

// parseFloat parses a single string to a float32.
func parseFloat(s string) float32 {
	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		logger.Log.Error("Error parsing Shininess: ", zap.Error(err))
		return 0
	}
	return float32(f)
}

func parseVertex(parts []string) ([]float32, error) {
	var vertex []float32
	for _, part := range parts {
		val, err := strconv.ParseFloat(part, 32)
		if err != nil {
			return nil, fmt.Errorf("Invalid vertex value %v: %v", part, err)
		}
		vertex = append(vertex, float32(val))
	}
	return vertex, nil
}

func parseFace(parts []string) ([]int32, error) {
	var face []int32
	startIndex := 0

	// Skip the first part if it's "f", adjusting for OBJ face definitions
	if parts[0] == "f" {
		startIndex = 1
	}

	for _, part := range parts[startIndex:] {
		vals := strings.Split(part, "/")
		idx, err := strconv.ParseInt(vals[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("Invalid face index %v: %v", vals[0], err)
		}
		face = append(face, int32(idx-1)) // .obj indices start at 1, not 0
	}

	// Convert quads to triangles
	if len(face) == 4 {
		return []int32{face[0], face[1], face[2], face[0], face[2], face[3]}, nil
	} else {
		return face, nil
	}
}

// for 2D textures
func parseTextureCoordinate(parts []string) ([]float32, error) {
	var texCoord []float32
	for _, part := range parts {
		val, err := strconv.ParseFloat(part, 32)
		if err != nil {
			return nil, fmt.Errorf("Invalid texture coordinate value %v: %v", part, err)
		}
		texCoord = append(texCoord, float32(val))
	}
	return texCoord, nil
}

func RecalculateNormals(vertices []float32, faces []int32) []float32 {
	if len(vertices) == 0 || len(faces) == 0 {
		log.Println("Empty vertices or faces slice")
		return nil // Return an empty slice or handle this case as appropriate
	}

	var normals = make([]float32, len(vertices))

	// Calculate normals for each face
	for i := 0; i+2 < len(faces); i += 3 {
		idx0 := faces[i] * 3
		idx1 := faces[i+1] * 3
		idx2 := faces[i+2] * 3

		// Ensure indices are within the bounds of the vertices array
		if idx0+2 >= int32(len(vertices)) || idx1+2 >= int32(len(vertices)) || idx2+2 >= int32(len(vertices)) {
			log.Printf("Index out of bounds: idx0=%d, idx1=%d, idx2=%d, len(vertices)=%d", idx0, idx1, idx2, len(vertices))
			continue // Skip this iteration to avoid panic
		}

		v0 := mgl32.Vec3{vertices[idx0], vertices[idx0+1], vertices[idx0+2]}
		v1 := mgl32.Vec3{vertices[idx1], vertices[idx1+1], vertices[idx1+2]}
		v2 := mgl32.Vec3{vertices[idx2], vertices[idx2+1], vertices[idx2+2]}

		edge1 := v1.Sub(v0)
		edge2 := v2.Sub(v0)
		normal := edge1.Cross(edge2).Normalize()

		// Safely add this normal to each vertex's normals
		for j := 0; j < 3; j++ {
			if idx0+int32(j) < int32(len(normals)) {
				normals[idx0+int32(j)] += normal[j]
			}
			if idx1+int32(j) < int32(len(normals)) {
				normals[idx1+int32(j)] += normal[j]
			}
			if idx2+int32(j) < int32(len(normals)) {
				normals[idx2+int32(j)] += normal[j]
			}
		}
	}

	// Normalize the normals
	for i := 0; i < len(normals); i += 3 {
		if i+2 < len(normals) { // Ensure i+2 is within bounds
			normal := mgl32.Vec3{normals[i], normals[i+1], normals[i+2]}.Normalize()
			normals[i], normals[i+1], normals[i+2] = normal[0], normal[1], normal[2]
		}
	}

	return normals
}
