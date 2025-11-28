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
	// Build unified index buffer first (preserve exact OBJ order)
	var unifiedFaces []FaceVertex
	var faceMaterialMap []string // Maps each face to its material

	// Helper function to ensure material has proper defaults
	ensureMaterial := func(material *renderer.Material) {
		if material == nil {
			return
		}
		if material.Alpha == 0 && material.Roughness == 0 && material.Metallic == 0 && material.Exposure == 0 {
			material.Alpha = 1.0
			material.Roughness = 0.5
			material.Metallic = 0.0
			material.Exposure = 1.0
		}
	}

	// TODO: I may want to review this later
	model = &renderer.Model{}
	model.SourcePath = filename // Store original file path for scene serialization
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
			faceVertices, err := parseFace(parts[1:])
			if err != nil {
				logger.Log.Error("Error parsing face: ", zap.Error(err))
				return nil, err
			}

			// Add face vertices to unified index buffer (preserve exact OBJ order)
			unifiedFaces = append(unifiedFaces, faceVertices...)

			// Track which material this face uses
			matName := currentMaterialName
			if matName == "" {
				matName = "default"
			}

			// Add material name for each vertex in this face
			for i := 0; i < len(faceVertices); i++ {
				faceMaterialMap = append(faceMaterialMap, matName)
			}

			// Also add to legacy faces array for compatibility (vertex indices only)
			for _, faceVertex := range faceVertices {
				faces = append(faces, faceVertex.VertexIdx)
			}
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

	// Fix 1: Only apply index unification if model uses separate indices
	hasSeparateIndices := false
	for _, fv := range unifiedFaces {
		if fv.TexCoordIdx >= 0 || fv.NormalIdx >= 0 {
			hasSeparateIndices = true
			break
		}
	}

	// Declare variables in outer scope
	var unifiedVertices []float32
	var unifiedIndices []uint32
	var unifiedMaterialMap map[int]string

	if hasSeparateIndices && len(unifiedFaces) > 0 {
		// Index unification: Convert separate indices to unified vertex buffer
		type VertexKey struct {
			v, vt, vn int32
			// Note: Material is NOT part of the key - vertices CAN be shared across materials
			// The material is applied at draw time via uniforms, not stored per-vertex
		}

		vertexMap := make(map[VertexKey]uint32)   // Maps triplet -> unified index
		unifiedVertices = []float32{}             // Interleaved [x,y,z,u,v,nx,ny,nz]
		unifiedIndices = []uint32{}               // Final index buffer
		unifiedMaterialMap = make(map[int]string) // Material per index position in index buffer

		// Process each face vertex triplet
		for i, faceVertex := range unifiedFaces {
			matName := faceMaterialMap[i]

			// Create key for this vertex combination (geometry only, no material)
			key := VertexKey{
				v:  faceVertex.VertexIdx,
				vt: faceVertex.TexCoordIdx,
				vn: faceVertex.NormalIdx,
			}

			// Check if we've seen this combination before
			if existingIdx, exists := vertexMap[key]; exists {
				// Reuse existing unified vertex
				unifiedIndices = append(unifiedIndices, existingIdx)
			} else {
				// Create new unified vertex
				newIdx := uint32(len(unifiedVertices) / 8) // 8 floats per vertex
				vertexMap[key] = newIdx

				// Fetch and append position (3 floats)
				if faceVertex.VertexIdx >= 0 && int(faceVertex.VertexIdx*3+2) < len(vertices) {
					unifiedVertices = append(unifiedVertices,
						vertices[faceVertex.VertexIdx*3],
						vertices[faceVertex.VertexIdx*3+1],
						vertices[faceVertex.VertexIdx*3+2])
				} else {
					// Vertex index out of bounds!
					logger.Log.Error("Vertex index out of bounds during unification",
						zap.Int32("vertexIdx", faceVertex.VertexIdx),
						zap.Int("verticesLen", len(vertices)/3))
					unifiedVertices = append(unifiedVertices, 0.0, 0.0, 0.0)
				}

				// Fetch and append texcoord (2 floats)
				if faceVertex.TexCoordIdx >= 0 && int(faceVertex.TexCoordIdx*2+1) < len(textureCoords) {
					unifiedVertices = append(unifiedVertices,
						textureCoords[faceVertex.TexCoordIdx*2],
						textureCoords[faceVertex.TexCoordIdx*2+1])
				} else {
					// Default UV coordinates
					unifiedVertices = append(unifiedVertices, 0.0, 0.0)
				}

				// Fetch and append normal (3 floats)
				if faceVertex.NormalIdx >= 0 && int(faceVertex.NormalIdx*3+2) < len(normals) {
					unifiedVertices = append(unifiedVertices,
						normals[faceVertex.NormalIdx*3],
						normals[faceVertex.NormalIdx*3+1],
						normals[faceVertex.NormalIdx*3+2])
				} else {
					// Normal index out of bounds - use default
					if faceVertex.NormalIdx >= 0 {
						logger.Log.Warn("Normal index out of bounds",
							zap.Int32("normalIdx", faceVertex.NormalIdx),
							zap.Int("normalsLen", len(normals)/3))
					}
					unifiedVertices = append(unifiedVertices, 0.0, 1.0, 0.0)
				}

				unifiedIndices = append(unifiedIndices, newIdx)
			}

			// Store material for THIS index position in the index buffer
			// This allows the same vertex to be used with different materials
			unifiedMaterialMap[len(unifiedIndices)-1] = matName
		}

		// Update model with unified data
		model.InterleavedData = unifiedVertices
		model.Vertices = []float32{}      // Now in unified buffer
		model.TextureCoords = []float32{} // Now in unified buffer
		model.Normals = []float32{}       // Now in unified buffer

		// Convert unified indices to int32 for compatibility
		faces = make([]int32, len(unifiedIndices))
		for i, idx := range unifiedIndices {
			faces[i] = int32(idx)
		}
		model.Faces = faces

		// CRITICAL VALIDATION: Check that all indices are within bounds
		maxVertexIndex := len(unifiedVertices) / 8
		for i, idx := range faces {
			if int(idx) >= maxVertexIndex {
				logger.Log.Error("INDEX OUT OF BOUNDS!",
					zap.Int("faceIdx", i),
					zap.Int32("index", idx),
					zap.Int("maxVertexIndex", maxVertexIndex))
			}
		}

		logger.Log.Info("Index unification applied",
			zap.Int("originalVertices", len(vertices)/3),
			zap.Int("unifiedVertices", len(unifiedVertices)/8),
			zap.Int("totalIndices", len(unifiedIndices)))
	} else {
		// Use the already-built interleavedData (old path)
		// DON'T overwrite model.InterleavedData - it's already correct
		logger.Log.Info("Using standard vertex layout (no separate indices detected)")
	}

	// Build material groups preserving exact face order
	model.MaterialGroups = make([]renderer.MaterialGroup, 0)
	if hasSeparateIndices && len(unifiedIndices) > 0 {
		// Build material groups using unified material map
		var materialRanges []struct {
			materialName string
			indexStart   int32
			indexCount   int32
		}

		// Build ranges by tracking material changes in index buffer
		currentMaterial := ""

		for i := 0; i < len(unifiedIndices); i++ {
			// Get material for this index position in the index buffer
			matName := unifiedMaterialMap[i]
			if matName == "" {
				matName = "default"
			}

			// Check if material changed
			if i == 0 || matName != currentMaterial {
				// Finalize previous range
				if i > 0 && len(materialRanges) > 0 {
					lastRange := &materialRanges[len(materialRanges)-1]
					lastRange.indexCount = int32(i) - lastRange.indexStart
				}

				// Start new material range
				materialRanges = append(materialRanges, struct {
					materialName string
					indexStart   int32
					indexCount   int32
				}{matName, int32(i), 0})

				currentMaterial = matName
			}
		}

		// Finalize last range
		if len(materialRanges) > 0 {
			lastRange := &materialRanges[len(materialRanges)-1]
			lastRange.indexCount = int32(len(unifiedIndices)) - lastRange.indexStart
		}

		// Create MaterialGroups from ranges
		for i, range_ := range materialRanges {
			material, exists := modelMaterials[range_.materialName]
			if !exists {
				logger.Log.Warn("Material not found, using model's main material", zap.String("material", range_.materialName))
				material = model.Material // Use the model's main material instead of default
			}

			// Ensure material is properly initialized
			ensureMaterial(material)

			group := renderer.MaterialGroup{
				Material:   material,
				IndexStart: range_.indexStart,
				IndexCount: range_.indexCount,
			}

			model.MaterialGroups = append(model.MaterialGroups, group)

			logger.Log.Info("Material group created",
				zap.Int("groupIndex", i),
				zap.String("material", range_.materialName),
				zap.Int32("indexStart", range_.indexStart),
				zap.Int32("indexCount", range_.indexCount),
				zap.Int32("indexEnd", range_.indexStart+range_.indexCount))
		}

		// DEBUG: Log first few material groups to verify correctness
		for i := 0; i < len(model.MaterialGroups) && i < 5; i++ {
			group := model.MaterialGroups[i]
			logger.Log.Info("DEBUG Material Group",
				zap.Int("groupIdx", i),
				zap.String("material", group.Material.Name),
				zap.Int32("indexStart", group.IndexStart),
				zap.Int32("indexCount", group.IndexCount),
				zap.Int("firstFewIndices", len(model.Faces)))

			// Log first 3 indices of this group
			if int(group.IndexStart) < len(model.Faces) {
				endIdx := int(group.IndexStart) + 3
				if endIdx > len(model.Faces) {
					endIdx = len(model.Faces)
				}
				if endIdx > int(group.IndexStart) {
					logger.Log.Info("  First indices",
						zap.Int32s("indices", model.Faces[group.IndexStart:endIdx]))
				}
			}
		}

		logger.Log.Info("Multi-material model loaded with index unification",
			zap.Int("materialGroups", len(model.MaterialGroups)),
			zap.Int("originalVertices", len(vertices)/3),
			zap.Int("unifiedVertices", len(unifiedVertices)/8),
			zap.Int("totalIndices", len(unifiedIndices)))

	}

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
			currentMaterial = &renderer.Material{
				Name:      fields[1],
				Alpha:     1.0, // Opaque by default
				Roughness: 0.5, // Mid-range roughness
				Metallic:  0.0, // Non-metallic by default
				Exposure:  1.0, // Standard exposure
			}
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
		case "d": // Dissolve (alpha/opacity)
			if len(fields) == 2 {
				currentMaterial.Alpha = parseFloat(fields[1])
			}
		case "map_Kd": // Diffuse texture map
			if len(fields) >= 2 {
				// Get texture path - it might be the last field if there are options
				texturePath := fields[len(fields)-1]

				// Handle absolute paths or relative paths
				var fullPath string
				if filepath.IsAbs(texturePath) {
					fullPath = texturePath
				} else {
					// Texture path is relative to MTL file location
					fullPath = filepath.Join(filepath.Dir(filename), texturePath)
				}

				// Store the texture path - it will be loaded later when OpenGL is initialized
				currentMaterial.TexturePath = fullPath
				logger.Log.Debug("Stored texture path for material",
					zap.String("material", currentMaterial.Name),
					zap.String("path", fullPath))
			}
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

type FaceVertex struct {
	VertexIdx   int32
	TexCoordIdx int32
	NormalIdx   int32
}

func parseFace(parts []string) ([]FaceVertex, error) {
	var face []FaceVertex
	startIndex := 0

	// Skip the first part if it's "f", adjusting for OBJ face definitions
	if parts[0] == "f" {
		startIndex = 1
	}

	for _, part := range parts[startIndex:] {
		vals := strings.Split(part, "/")

		// Parse vertex index (required)
		vertexIdx, err := strconv.ParseInt(vals[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("Invalid vertex index %v: %v", vals[0], err)
		}

		// Parse texture coordinate index (optional)
		var texCoordIdx int32 = -1
		if len(vals) > 1 && vals[1] != "" {
			texIdx, err := strconv.ParseInt(vals[1], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("Invalid texture coordinate index %v: %v", vals[1], err)
			}
			texCoordIdx = int32(texIdx - 1) // .obj indices start at 1, not 0
		}

		// Parse normal index (optional)
		var normalIdx int32 = -1
		if len(vals) > 2 && vals[2] != "" {
			normIdx, err := strconv.ParseInt(vals[2], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("Invalid normal index %v: %v", vals[2], err)
			}
			normalIdx = int32(normIdx - 1) // .obj indices start at 1, not 0
		}

		face = append(face, FaceVertex{
			VertexIdx:   int32(vertexIdx - 1), // .obj indices start at 1, not 0
			TexCoordIdx: texCoordIdx,
			NormalIdx:   normalIdx,
		})
	}

	// Convert quads to triangles
	// Standard quad triangulation: split into two triangles
	// Triangle 1: v0, v1, v2
	// Triangle 2: v0, v2, v3
	if len(face) == 4 {
		// Use counter-clockwise winding order for both triangles
		return []FaceVertex{face[0], face[1], face[2], face[0], face[2], face[3]}, nil
	} else if len(face) > 4 {
		// Polygons with >4 vertices: triangulate as fan from first vertex
		logger.Log.Warn("Face with more than 4 vertices detected, using fan triangulation", zap.Int("vertexCount", len(face)))
		var triangulated []FaceVertex
		for i := 1; i < len(face)-1; i++ {
			triangulated = append(triangulated, face[0], face[i], face[i+1])
		}
		return triangulated, nil
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
