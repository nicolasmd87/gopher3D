package loader

import (
	"testing"

	"github.com/go-gl/mathgl/mgl32"
)

func TestNewVoxelWorld(t *testing.T) {
	world := NewVoxelWorld(16, 2, 2, 64, 1.0, nil, InstancedMode)

	if world == nil {
		t.Fatal("NewVoxelWorld returned nil")
	}

	if world.ChunkSize != 16 {
		t.Errorf("Expected ChunkSize 16, got %d", world.ChunkSize)
	}
	if world.WorldSizeX != 2 {
		t.Errorf("Expected WorldSizeX 2, got %d", world.WorldSizeX)
	}
	if world.WorldSizeZ != 2 {
		t.Errorf("Expected WorldSizeZ 2, got %d", world.WorldSizeZ)
	}
	if world.MaxHeight != 64 {
		t.Errorf("Expected MaxHeight 64, got %d", world.MaxHeight)
	}
	if world.VoxelSize != 1.0 {
		t.Errorf("Expected VoxelSize 1.0, got %f", world.VoxelSize)
	}
}

func TestVoxelID(t *testing.T) {
	var air VoxelID = 0
	var grass VoxelID = 1
	var dirt VoxelID = 2
	var stone VoxelID = 3

	if grass <= air {
		t.Error("grass should be > air")
	}
	if dirt <= air {
		t.Error("dirt should be > air")
	}
	if stone <= air {
		t.Error("stone should be > air")
	}
}

func TestGetVoxelColor(t *testing.T) {
	grassColor := GetVoxelColor(1)

	if grassColor[0] == 0 && grassColor[1] == 0 && grassColor[2] == 0 {
		t.Error("Grass color should not be black")
	}

	airColor := GetVoxelColor(0)
	if airColor[0] != 0 || airColor[1] != 0 || airColor[2] != 0 {
		t.Error("Air color should be black (0,0,0)")
	}
}

func TestSetVoxelColor(t *testing.T) {
	var stoneID VoxelID = 3
	originalColor := GetVoxelColor(stoneID)

	SetVoxelColor(stoneID, mgl32.Vec3{1.0, 0.0, 0.0})
	newColor := GetVoxelColor(stoneID)

	if newColor[0] != 1.0 || newColor[1] != 0.0 || newColor[2] != 0.0 {
		t.Errorf("Expected red (1,0,0), got %v", newColor)
	}

	SetVoxelColor(stoneID, originalColor)
}

func TestClearCustomVoxelColors(t *testing.T) {
	SetVoxelColor(3, mgl32.Vec3{1.0, 1.0, 1.0})
	ClearCustomVoxelColors()

	color := GetVoxelColor(3)
	if color[0] == 1.0 && color[1] == 1.0 && color[2] == 1.0 {
		t.Error("ClearCustomVoxelColors should reset to default")
	}
}

func TestVoxelRenderModes(t *testing.T) {
	if InstancedMode == SurfaceNetsMode {
		t.Error("Render modes should be different")
	}
}
