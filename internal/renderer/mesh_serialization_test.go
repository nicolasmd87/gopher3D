package renderer

import (
	"testing"

	"github.com/go-gl/mathgl/mgl32"
)

func TestMeshSerialization(t *testing.T) {
	// Create a simple model with mesh data
	original := &Model{
		Name:            "TestModel",
		Vertices:        []float32{0, 0, 0, 1, 0, 0, 0, 1, 0},
		InterleavedData: []float32{0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0, 0, 0, 1},
		Faces:           []int32{0, 1, 2},
		IsInstanced:     false,
		Material:        DefaultMaterial,
	}

	// Serialize
	mesh := SerializeMesh(original)
	if mesh == nil {
		t.Fatal("SerializeMesh returned nil")
	}

	// Verify serialized data
	if len(mesh.Vertices) != len(original.Vertices) {
		t.Errorf("Vertices length mismatch: got %d, want %d", len(mesh.Vertices), len(original.Vertices))
	}
	if len(mesh.Faces) != len(original.Faces) {
		t.Errorf("Faces length mismatch: got %d, want %d", len(mesh.Faces), len(original.Faces))
	}

	// Deserialize
	restored := DeserializeMesh(mesh)
	if restored == nil {
		t.Fatal("DeserializeMesh returned nil")
	}

	// Verify restored data
	if len(restored.Vertices) != len(original.Vertices) {
		t.Errorf("Restored vertices length mismatch: got %d, want %d", len(restored.Vertices), len(original.Vertices))
	}
}

func TestMeshBinaryEncoding(t *testing.T) {
	mesh := &SerializedMesh{
		Vertices:        []float32{0, 0, 0, 1, 0, 0, 0, 1, 0},
		InterleavedData: []float32{0, 0, 0, 0, 0, 0, 0, 1},
		Faces:           []int32{0, 1, 2},
		IsInstanced:     false,
	}

	// Encode
	data, err := EncodeMeshBinary(mesh)
	if err != nil {
		t.Fatalf("EncodeMeshBinary failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Encoded data is empty")
	}

	// Decode
	decoded, err := DecodeMeshBinary(data)
	if err != nil {
		t.Fatalf("DecodeMeshBinary failed: %v", err)
	}

	// Verify
	if len(decoded.Vertices) != len(mesh.Vertices) {
		t.Errorf("Vertices length mismatch: got %d, want %d", len(decoded.Vertices), len(mesh.Vertices))
	}
	if len(decoded.Faces) != len(mesh.Faces) {
		t.Errorf("Faces length mismatch: got %d, want %d", len(decoded.Faces), len(mesh.Faces))
	}
}

func TestInstancedMeshSerialization(t *testing.T) {
	// Create instanced model (like voxels)
	instanceCount := 10
	original := &Model{
		Name:                  "VoxelTerrain",
		Vertices:              []float32{0, 0, 0, 1, 0, 0, 0, 1, 0, 1, 1, 0},
		InterleavedData:       []float32{0, 0, 0, 0, 0, 0, 0, 1},
		Faces:                 []int32{0, 1, 2, 1, 2, 3},
		IsInstanced:           true,
		InstanceCount:         instanceCount,
		InstanceModelMatrices: make([]mgl32.Mat4, instanceCount),
		InstanceColors:        make([]mgl32.Vec3, instanceCount),
		Material:              DefaultMaterial,
	}

	// Set up instance data
	for i := 0; i < instanceCount; i++ {
		original.InstanceModelMatrices[i] = mgl32.Translate3D(float32(i), float32(i*2), float32(i*3))
		original.InstanceColors[i] = mgl32.Vec3{float32(i) / 10, 0.5, 0.5}
	}

	// Serialize
	mesh := SerializeMesh(original)
	if !mesh.IsInstanced {
		t.Error("Serialized mesh should be instanced")
	}
	if len(mesh.InstancePositions) != instanceCount {
		t.Errorf("Instance positions length mismatch: got %d, want %d", len(mesh.InstancePositions), instanceCount)
	}
	if len(mesh.InstanceColors) != instanceCount {
		t.Errorf("Instance colors length mismatch: got %d, want %d", len(mesh.InstanceColors), instanceCount)
	}

	// Binary encode/decode
	data, err := EncodeMeshBinary(mesh)
	if err != nil {
		t.Fatalf("EncodeMeshBinary failed: %v", err)
	}

	decoded, err := DecodeMeshBinary(data)
	if err != nil {
		t.Fatalf("DecodeMeshBinary failed: %v", err)
	}

	if !decoded.IsInstanced {
		t.Error("Decoded mesh should be instanced")
	}
	if len(decoded.InstancePositions) != instanceCount {
		t.Errorf("Decoded instance positions length mismatch: got %d, want %d", len(decoded.InstancePositions), instanceCount)
	}

	// Deserialize to model
	restored := DeserializeMesh(decoded)
	if !restored.IsInstanced {
		t.Error("Restored model should be instanced")
	}
	if len(restored.InstanceModelMatrices) != instanceCount {
		t.Errorf("Restored instance matrices length mismatch: got %d, want %d", len(restored.InstanceModelMatrices), instanceCount)
	}

	// Verify positions are restored correctly
	for i := 0; i < instanceCount; i++ {
		pos := restored.InstanceModelMatrices[i].Col(3).Vec3()
		expectedX := float32(i)
		if pos.X() != expectedX {
			t.Errorf("Instance %d X position mismatch: got %f, want %f", i, pos.X(), expectedX)
		}
	}
}
