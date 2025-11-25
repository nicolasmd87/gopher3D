package main

import (
	"Gopher3D/internal/loader"
	"Gopher3D/internal/renderer"
	"fmt"
	"github.com/inkyblackness/imgui-go/v4"
	"time"
)

var (
	// Voxel generation parameters
	voxelScale     = float32(0.05)
	voxelAmplitude = float32(15.0)
	voxelSeed      = int32(time.Now().Unix())
	voxelThreshold = float32(0.2)
	voxelOctaves   = int32(4)
	voxelChunkSize = int32(32)
	voxelWorldSize = int32(2)
)

type VoxelConfig struct {
	Scale     float32 `json:"scale"`
	Amplitude float32 `json:"amplitude"`
	Seed      int32   `json:"seed"`
	Threshold float32 `json:"threshold"`
	Octaves   int32   `json:"octaves"`
	ChunkSize int32   `json:"chunk_size"`
	WorldSize int32   `json:"world_size"`
}

func renderAddVoxelDialog() {
	if eng == nil { return }
	imgui.OpenPopup("Add Voxel Terrain")
	centerX := float32(eng.Width) / 2
	centerY := float32(eng.Height) / 2
	imgui.SetNextWindowPosV(imgui.Vec2{X: centerX - 250, Y: centerY - 200}, imgui.ConditionAppearing, imgui.Vec2{})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 500, Y: 400}, imgui.ConditionAppearing)

	if imgui.BeginPopupModalV("Add Voxel Terrain", nil, imgui.WindowFlagsNoResize) {
		imgui.Text("Generate Realistic Voxel Terrain using Perlin Noise")
		imgui.Separator()
		
		imgui.Spacing()
		imgui.DragFloatV("Scale", &voxelScale, 0.001, 0.001, 1.0, "%.3f", 1.0)
		imgui.DragFloatV("Amplitude", &voxelAmplitude, 0.5, 1.0, 100.0, "%.1f", 1.0)
		imgui.DragIntV("Seed", &voxelSeed, 1, 0, 100000, "%d", 0)
		imgui.DragFloatV("Threshold", &voxelThreshold, 0.01, -1.0, 1.0, "%.2f", 1.0)
		imgui.SliderInt("Octaves", &voxelOctaves, 1, 8)
		imgui.SliderInt("Chunk Size", &voxelChunkSize, 16, 64)
		imgui.SliderInt("World Size (Chunks)", &voxelWorldSize, 1, 8)
		
		imgui.Spacing()
		imgui.Separator()
		
		if imgui.Button("Generate Terrain") {
			model := generateVoxelTerrain()
			if model != nil {
				eng.AddModel(model)
				logToConsole("Voxel terrain added to scene", "info")
			}
			showAddVoxel = false
			imgui.CloseCurrentPopup()
		}
		imgui.SameLine()
		if imgui.Button("Cancel") {
			showAddVoxel = false
			imgui.CloseCurrentPopup()
		}
		imgui.EndPopup()
	}
}

func generateVoxelTerrain() *renderer.Model {
	chunkSize := int(voxelChunkSize)
	worldSize := int(voxelWorldSize)
	voxelSizeVal := float32(1.0)
	
	// Create noise generator
	noise := renderer.NewImprovedPerlinNoise(int64(voxelSeed))
	
	// Use the internal loader library to create the voxel world structure
	// This leverages the existing @internal/loader/voxel_core.go implementation
	world := loader.NewVoxelWorld(chunkSize, worldSize, worldSize, 64, voxelSizeVal, loader.CreateCubeGeometry(voxelSizeVal), loader.InstancedMode)
	
	logToConsole(fmt.Sprintf("Generating voxel terrain (Seed: %d, Scale: %.3f)...", voxelSeed, voxelScale), "info")

	// Terrain generation using Perlin noise
	world.GenerateVoxelsParallel(func(x, y, z int) (loader.VoxelID, bool) {
		fx, fy, fz := float64(x)*float64(voxelScale), float64(y)*float64(voxelScale), float64(z)*float64(voxelScale)
		
		// 3D Noise for caves/overhangs
		n3d := noise.Turbulence(fx, fy, fz, int(voxelOctaves), 0.5)
		
		// Heightmap for base terrain
		h := noise.Turbulence2D(fx, fz, int(voxelOctaves), 0.5)
		height := int(float32(h) * voxelAmplitude) + 15 // Base height offset
		
		// Combine: Solid below heightmap, but cut out caves using 3D noise
		if y <= height {
			// Cave carving
			if n3d > float64(voxelThreshold) {
				return 0, false // Cave/Air
			}
			
			if y == height {
				return 1, true // Grass/Surface
			} else if y > height-3 {
				return 2, true // Dirt
			}
			return 3, true // Stone
		}
		
		return 0, false
	})
	
	model, err := world.CreateInstancedModel()
	if err != nil {
		logToConsole(fmt.Sprintf("Failed to create voxel mesh: %v", err), "error")
		return nil
	}
	
	model.Name = fmt.Sprintf("Voxel Terrain (%dx%d)", worldSize, worldSize)
	model.SetPosition(0, 0, 0)
	
	// Voxel terrain uses vertex colors or texture atlas usually, but here we use global material
	// In a real engine, voxels would have per-instance material/color data
	model.SetDiffuseColor(0.5, 0.5, 0.5) 
	model.SetMaterialPBR(0.0, 0.9)
	model.SetExposure(1.0) // Fix lighting issue (was defaulting to 0/black)
	
	// Store configuration in metadata for saving/loading
	if model.Metadata == nil {
		model.Metadata = make(map[string]interface{})
	}
	model.Metadata["isVoxel"] = true
	model.Metadata["voxelConfig"] = VoxelConfig{
		Scale:     voxelScale,
		Amplitude: voxelAmplitude,
		Seed:      voxelSeed,
		Threshold: voxelThreshold,
		Octaves:   voxelOctaves,
		ChunkSize: voxelChunkSize,
		WorldSize: voxelWorldSize,
	}
	
	// eng.AddModel(model) -- Removed to avoid duplicate add or control issues
	logToConsole("Voxel terrain created", "info")
	return model
}

func regenerateVoxelTerrain(config VoxelConfig) *renderer.Model {
	voxelScale = config.Scale
	voxelAmplitude = config.Amplitude
	voxelSeed = config.Seed
	voxelThreshold = config.Threshold
	voxelOctaves = config.Octaves
	voxelChunkSize = config.ChunkSize
	voxelWorldSize = config.WorldSize
	
	return generateVoxelTerrain()
}
