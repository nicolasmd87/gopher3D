package renderer

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"github.com/go-gl/mathgl/mgl32"
)

// SerializedMesh contains all data needed to reconstruct a mesh at runtime
type SerializedMesh struct {
	// Mesh geometry
	Vertices        []float32 `json:"vertices,omitempty"`
	InterleavedData []float32 `json:"interleaved_data,omitempty"`
	Faces           []int32   `json:"faces,omitempty"`

	// For instanced rendering (voxels, particles, etc.)
	IsInstanced       bool         `json:"is_instanced,omitempty"`
	InstanceCount     int          `json:"instance_count,omitempty"`
	InstancePositions [][3]float32 `json:"instance_positions,omitempty"`
	InstanceColors    [][3]float32 `json:"instance_colors,omitempty"`
}

// SerializedModel is the complete serializable representation of a model
type SerializedModel struct {
	Name string `json:"name"`
	Type string `json:"type"` // "file", "voxel", "primitive", "procedural"

	// For file-based models
	SourcePath string `json:"source_path,omitempty"`

	// For procedural/voxel models - mesh data stored in binary file
	MeshDataFile string `json:"mesh_data_file,omitempty"`

	// Transform
	Position [3]float32 `json:"position"`
	Scale    [3]float32 `json:"scale"`
	Rotation [4]float32 `json:"rotation"` // Quaternion: W, X, Y, Z

	// Material
	Material SerializedMaterial `json:"material"`

	// Metadata for regeneration (optional, for voxels)
	VoxelConfig map[string]interface{} `json:"voxel_config,omitempty"`
}

// SerializedMaterial contains all material properties
type SerializedMaterial struct {
	DiffuseColor  [3]float32 `json:"diffuse_color"`
	SpecularColor [3]float32 `json:"specular_color"`
	Shininess     float32    `json:"shininess"`
	Metallic      float32    `json:"metallic"`
	Roughness     float32    `json:"roughness"`
	Exposure      float32    `json:"exposure"`
	Alpha         float32    `json:"alpha"`
	TexturePath   string     `json:"texture_path,omitempty"`
}

// SerializeMesh converts a Model's mesh data to SerializedMesh
func SerializeMesh(model *Model) *SerializedMesh {
	mesh := &SerializedMesh{
		Vertices:        model.Vertices,
		InterleavedData: model.InterleavedData,
		Faces:           model.Faces,
		IsInstanced:     model.IsInstanced,
		InstanceCount:   model.InstanceCount,
	}

	// Serialize instance data if present
	if model.IsInstanced && len(model.InstanceModelMatrices) > 0 {
		mesh.InstancePositions = make([][3]float32, len(model.InstanceModelMatrices))
		for i, mat := range model.InstanceModelMatrices {
			pos := mat.Col(3).Vec3()
			mesh.InstancePositions[i] = [3]float32{pos.X(), pos.Y(), pos.Z()}
		}
	}

	if len(model.InstanceColors) > 0 {
		mesh.InstanceColors = make([][3]float32, len(model.InstanceColors))
		for i, col := range model.InstanceColors {
			mesh.InstanceColors[i] = [3]float32{col.X(), col.Y(), col.Z()}
		}
	}

	return mesh
}

// DeserializeMesh reconstructs a Model from SerializedMesh
func DeserializeMesh(mesh *SerializedMesh) *Model {
	model := &Model{
		Vertices:        mesh.Vertices,
		InterleavedData: mesh.InterleavedData,
		Faces:           mesh.Faces,
		IsInstanced:     mesh.IsInstanced,
		InstanceCount:   mesh.InstanceCount,
		Position:        mgl32.Vec3{0, 0, 0},
		Scale:           mgl32.Vec3{1, 1, 1},
		Rotation:        mgl32.QuatIdent(),
		Material:        DefaultMaterial,
		IsDirty:         true,
	}

	// Create unique material
	uniqueMaterial := *model.Material
	model.Material = &uniqueMaterial

	// Reconstruct instance matrices from positions
	if mesh.IsInstanced && len(mesh.InstancePositions) > 0 {
		model.InstanceModelMatrices = make([]mgl32.Mat4, len(mesh.InstancePositions))
		for i, pos := range mesh.InstancePositions {
			model.InstanceModelMatrices[i] = mgl32.Translate3D(pos[0], pos[1], pos[2])
		}
		// Update instance count to match actual data
		model.InstanceCount = len(mesh.InstancePositions)
		model.InstanceMatricesUpdated = true
	}

	// Reconstruct instance colors
	if len(mesh.InstanceColors) > 0 {
		model.InstanceColors = make([]mgl32.Vec3, len(mesh.InstanceColors))
		for i, col := range mesh.InstanceColors {
			model.InstanceColors[i] = mgl32.Vec3{col[0], col[1], col[2]}
		}
	}

	// Calculate bounding sphere for frustum culling
	model.CalculateBoundingSphere()

	return model
}

// EncodeMeshBinary encodes mesh data to compressed binary format
func EncodeMeshBinary(mesh *SerializedMesh) ([]byte, error) {
	var buf bytes.Buffer

	// Use gzip compression
	gzWriter := gzip.NewWriter(&buf)

	// Write header magic number
	if err := binary.Write(gzWriter, binary.LittleEndian, uint32(0x4D455348)); err != nil { // "MESH"
		return nil, err
	}

	// Write version
	if err := binary.Write(gzWriter, binary.LittleEndian, uint32(1)); err != nil {
		return nil, err
	}

	// Write flags
	flags := uint32(0)
	if mesh.IsInstanced {
		flags |= 1
	}
	if err := binary.Write(gzWriter, binary.LittleEndian, flags); err != nil {
		return nil, err
	}

	// Write vertices
	if err := writeFloat32Slice(gzWriter, mesh.Vertices); err != nil {
		return nil, err
	}

	// Write interleaved data
	if err := writeFloat32Slice(gzWriter, mesh.InterleavedData); err != nil {
		return nil, err
	}

	// Write faces
	if err := writeInt32Slice(gzWriter, mesh.Faces); err != nil {
		return nil, err
	}

	// Write instance data if present
	if mesh.IsInstanced {
		if err := binary.Write(gzWriter, binary.LittleEndian, int32(mesh.InstanceCount)); err != nil {
			return nil, err
		}

		// Write instance positions
		if err := binary.Write(gzWriter, binary.LittleEndian, int32(len(mesh.InstancePositions))); err != nil {
			return nil, err
		}
		for _, pos := range mesh.InstancePositions {
			if err := binary.Write(gzWriter, binary.LittleEndian, pos); err != nil {
				return nil, err
			}
		}

		// Write instance colors
		if err := binary.Write(gzWriter, binary.LittleEndian, int32(len(mesh.InstanceColors))); err != nil {
			return nil, err
		}
		for _, col := range mesh.InstanceColors {
			if err := binary.Write(gzWriter, binary.LittleEndian, col); err != nil {
				return nil, err
			}
		}
	}

	if err := gzWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// DecodeMeshBinary decodes compressed binary mesh data
func DecodeMeshBinary(data []byte) (*SerializedMesh, error) {
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Read header magic
	var magic uint32
	if err := binary.Read(gzReader, binary.LittleEndian, &magic); err != nil {
		return nil, err
	}
	if magic != 0x4D455348 {
		return nil, fmt.Errorf("invalid mesh file magic: %x", magic)
	}

	// Read version
	var version uint32
	if err := binary.Read(gzReader, binary.LittleEndian, &version); err != nil {
		return nil, err
	}
	if version != 1 {
		return nil, fmt.Errorf("unsupported mesh version: %d", version)
	}

	// Read flags
	var flags uint32
	if err := binary.Read(gzReader, binary.LittleEndian, &flags); err != nil {
		return nil, err
	}

	mesh := &SerializedMesh{
		IsInstanced: flags&1 != 0,
	}

	// Read vertices
	mesh.Vertices, err = readFloat32Slice(gzReader)
	if err != nil {
		return nil, err
	}

	// Read interleaved data
	mesh.InterleavedData, err = readFloat32Slice(gzReader)
	if err != nil {
		return nil, err
	}

	// Read faces
	mesh.Faces, err = readInt32Slice(gzReader)
	if err != nil {
		return nil, err
	}

	// Read instance data if present
	if mesh.IsInstanced {
		var instanceCount int32
		if err := binary.Read(gzReader, binary.LittleEndian, &instanceCount); err != nil {
			return nil, err
		}
		mesh.InstanceCount = int(instanceCount)

		// Read instance positions
		var posCount int32
		if err := binary.Read(gzReader, binary.LittleEndian, &posCount); err != nil {
			return nil, err
		}
		mesh.InstancePositions = make([][3]float32, posCount)
		for i := range mesh.InstancePositions {
			if err := binary.Read(gzReader, binary.LittleEndian, &mesh.InstancePositions[i]); err != nil {
				return nil, err
			}
		}

		// Read instance colors
		var colCount int32
		if err := binary.Read(gzReader, binary.LittleEndian, &colCount); err != nil {
			return nil, err
		}
		mesh.InstanceColors = make([][3]float32, colCount)
		for i := range mesh.InstanceColors {
			if err := binary.Read(gzReader, binary.LittleEndian, &mesh.InstanceColors[i]); err != nil {
				return nil, err
			}
		}
	}

	return mesh, nil
}

// Helper functions for binary encoding
func writeFloat32Slice(w io.Writer, data []float32) error {
	if err := binary.Write(w, binary.LittleEndian, int32(len(data))); err != nil {
		return err
	}
	for _, v := range data {
		if err := binary.Write(w, binary.LittleEndian, v); err != nil {
			return err
		}
	}
	return nil
}

func writeInt32Slice(w io.Writer, data []int32) error {
	if err := binary.Write(w, binary.LittleEndian, int32(len(data))); err != nil {
		return err
	}
	for _, v := range data {
		if err := binary.Write(w, binary.LittleEndian, v); err != nil {
			return err
		}
	}
	return nil
}

func readFloat32Slice(r io.Reader) ([]float32, error) {
	var count int32
	if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
		return nil, err
	}
	data := make([]float32, count)
	for i := range data {
		if err := binary.Read(r, binary.LittleEndian, &data[i]); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func readInt32Slice(r io.Reader) ([]int32, error) {
	var count int32
	if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
		return nil, err
	}
	data := make([]int32, count)
	for i := range data {
		if err := binary.Read(r, binary.LittleEndian, &data[i]); err != nil {
			return nil, err
		}
	}
	return data, nil
}

// SerializeModelToJSON creates a JSON representation of a model for scene files
func SerializeModelToJSON(model *Model) ([]byte, error) {
	serialized := SerializedModel{
		Name:     model.Name,
		Position: [3]float32{model.Position.X(), model.Position.Y(), model.Position.Z()},
		Scale:    [3]float32{model.Scale.X(), model.Scale.Y(), model.Scale.Z()},
		Rotation: [4]float32{model.Rotation.W, model.Rotation.V.X(), model.Rotation.V.Y(), model.Rotation.V.Z()},
	}

	// Determine type
	if model.SourcePath != "" {
		serialized.Type = "file"
		serialized.SourcePath = model.SourcePath
	} else if model.IsInstanced {
		serialized.Type = "procedural"
	} else {
		serialized.Type = "primitive"
	}

	// Material
	if model.Material != nil {
		serialized.Material = SerializedMaterial{
			DiffuseColor:  model.Material.DiffuseColor,
			SpecularColor: model.Material.SpecularColor,
			Shininess:     model.Material.Shininess,
			Metallic:      model.Material.Metallic,
			Roughness:     model.Material.Roughness,
			Exposure:      model.Material.Exposure,
			Alpha:         model.Material.Alpha,
			TexturePath:   model.Material.TexturePath,
		}
	}

	return json.Marshal(serialized)
}
