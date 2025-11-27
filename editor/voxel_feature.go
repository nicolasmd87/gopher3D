package main

import (
	"Gopher3D/internal/loader"
	"Gopher3D/internal/renderer"
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/inkyblackness/imgui-go/v4"
	"time"
)

var (
	// Voxel generation parameters
	voxelScale       = float32(0.05)
	voxelAmplitude   = float32(15.0)
	voxelSeed        = int32(time.Now().Unix())
	voxelThreshold   = float32(0.2)
	voxelOctaves     = int32(4)
	voxelChunkSize   = int32(32)
	voxelWorldSize   = int32(2)
	voxelBiome       = int32(0) // 0=Plains, 1=Mountains, 2=Desert, 3=Islands, 4=Caves
	voxelTreeDensity = float32(0.02)
)

type VoxelConfig struct {
	Scale       float32 `json:"scale"`
	Amplitude   float32 `json:"amplitude"`
	Seed        int32   `json:"seed"`
	Threshold   float32 `json:"threshold"`
	Octaves     int32   `json:"octaves"`
	ChunkSize   int32   `json:"chunk_size"`
	WorldSize   int32   `json:"world_size"`
	Biome       int32   `json:"biome"`
	TreeDensity float32 `json:"tree_density"`
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
		
		// Biome selection
		biomeNames := []string{"Plains", "Mountains", "Desert", "Islands", "Caves"}
		if imgui.BeginCombo("Biome", biomeNames[voxelBiome]) {
			for i := int32(0); i < 5; i++ {
				isSelected := voxelBiome == i
				if imgui.SelectableV(biomeNames[i], isSelected, 0, imgui.Vec2{}) {
					voxelBiome = i
					// Apply biome presets
					switch i {
					case 0: // Plains
						voxelAmplitude = 8.0
						voxelThreshold = 0.3
						voxelTreeDensity = 0.02
					case 1: // Mountains
						voxelAmplitude = 25.0
						voxelThreshold = 0.1
						voxelTreeDensity = 0.01
					case 2: // Desert
						voxelAmplitude = 5.0
						voxelThreshold = 0.5
						voxelTreeDensity = 0.0
					case 3: // Islands
						voxelAmplitude = 15.0
						voxelThreshold = 0.0
						voxelTreeDensity = 0.03
					case 4: // Caves
						voxelAmplitude = 12.0
						voxelThreshold = 0.35
						voxelTreeDensity = 0.0
					}
				}
				if isSelected {
					imgui.SetItemDefaultFocus()
				}
			}
			imgui.EndCombo()
		}
		
		imgui.Separator()
		imgui.DragFloatV("Scale", &voxelScale, 0.001, 0.001, 10.0, "%.3f", 1.0)
		imgui.DragFloatV("Amplitude", &voxelAmplitude, 0.5, 1.0, 500.0, "%.1f", 1.0)
		imgui.DragIntV("Seed", &voxelSeed, 1, 0, 999999, "%d", 0)
		imgui.DragFloatV("Cave Threshold", &voxelThreshold, 0.01, -2.0, 2.0, "%.2f", 1.0)
		imgui.SliderInt("Octaves", &voxelOctaves, 1, 12)
		imgui.SliderInt("Chunk Size", &voxelChunkSize, 8, 128)
		imgui.SliderInt("World Size (Chunks)", &voxelWorldSize, 1, 32)
		imgui.DragFloatV("Tree Density", &voxelTreeDensity, 0.001, 0.0, 1.0, "%.3f", 1.0)
		
		imgui.Spacing()
		imgui.Separator()
		
		if imgui.Button("Generate Terrain") {
			model := generateVoxelTerrain()
			if model != nil {
				eng.AddModel(model)
				createGameObjectForModel(model)
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
	
	totalChunks := worldSize * worldSize
	logToConsole(fmt.Sprintf("Generating voxel terrain: %d chunks (%dx%d), Seed: %d, Scale: %.3f", 
		totalChunks, worldSize, worldSize, voxelSeed, voxelScale), "info")

	// Terrain generation using Perlin noise with biome support
	world.GenerateVoxelsParallel(func(x, y, z int) (loader.VoxelID, bool) {
		fx, fy, fz := float64(x)*float64(voxelScale), float64(y)*float64(voxelScale), float64(z)*float64(voxelScale)
		
		// 3D Noise for caves/overhangs
		n3d := noise.Turbulence(fx, fy, fz, int(voxelOctaves), 0.5)
		
		// Biome-specific generation
		switch voxelBiome {
		case 0: // Plains - Rolling hills
			h := noise.Turbulence2D(fx, fz, int(voxelOctaves), 0.5)
			height := int(float32(h)*voxelAmplitude) + 15
			
			if y <= height {
				if n3d > float64(voxelThreshold) {
					return 0, false // Cave
				}
				if y == height {
					return 1, true // Grass
				} else if y > height-3 {
					return 2, true // Dirt
				}
				return 3, true // Stone
			}
			
		case 1: // Mountains - Tall peaks with overhangs
			h := noise.Turbulence2D(fx, fz, int(voxelOctaves), 0.6)
			height := int(float32(h)*voxelAmplitude) + 20
			
			// Add mountain peaks using 3D noise
			peakNoise := noise.Turbulence(fx, fy*0.3, fz, int(voxelOctaves), 0.5)
			if peakNoise > 0.3 {
				height += int(float32(peakNoise-0.3) * 15.0)
			}
			
			if y <= height {
				if n3d > float64(voxelThreshold) {
					return 0, false // Cave
				}
				if y == height && y < 35 {
					return 1, true // Grass (lower elevations)
				} else if y == height {
					return 3, true // Stone (peaks)
				} else if y > height-2 {
					return 2, true // Dirt
				}
				return 3, true // Stone
			}
			
		case 2: // Desert - Flat with dunes
			h := noise.Turbulence2D(fx*2.0, fz*2.0, int(voxelOctaves-1), 0.4)
			height := int(float32(h)*voxelAmplitude) + 12
			
			// Sand dunes (no caves in desert)
			if y <= height {
				return 4, true // Sand (VoxelID 4)
			}
			
		case 3: // Islands - Water level with islands
			h := noise.Turbulence2D(fx, fz, int(voxelOctaves), 0.5)
			islandNoise := noise.Turbulence2D(fx*0.5, fz*0.5, 2, 0.5)
			
			waterLevel := 18
			height := int(float32(h)*voxelAmplitude) + 10
			
			// Create islands using low-frequency noise
			if islandNoise > 0.2 {
				height += int((float32(islandNoise) - 0.2) * 20.0)
			}
			
			if y <= height {
				if n3d > float64(voxelThreshold) && y > waterLevel {
					return 0, false // Cave (only above water)
				}
				if y == height && y > waterLevel {
					return 1, true // Grass
				} else if y > height-2 && y > waterLevel {
					return 2, true // Dirt
				} else if y <= waterLevel {
					return 3, true // Stone/Underwater
				}
				return 3, true // Stone
			}
			
		case 4: // Caves - Complex cave systems
			h := noise.Turbulence2D(fx, fz, int(voxelOctaves), 0.5)
			height := int(float32(h)*voxelAmplitude) + 15
			
			// Multiple cave layers using different noise scales
			cave1 := noise.Turbulence(fx, fy, fz, int(voxelOctaves), 0.5)
			cave2 := noise.Turbulence(fx*2.0, fy*2.0, fz*2.0, int(voxelOctaves-1), 0.5)
			cave3 := noise.Turbulence(fx*0.5, fy*0.5, fz*0.5, 2, 0.5)
			
			// Combine cave noises for complex systems
			combinedCave := (cave1 + cave2*0.5 + cave3*0.3) / 1.8
			
			if y <= height {
				if combinedCave > float64(voxelThreshold) {
					return 0, false // Cave
				}
				if y == height {
					return 1, true // Grass
				} else if y > height-3 {
					return 2, true // Dirt
				}
				return 3, true // Stone
			}
		}
		
		return 0, false
	})
	
	// Generate trees after terrain if enabled
	if voxelTreeDensity > 0.001 {
		generateVoxelTrees(world, noise, int(voxelSeed))
	}
	
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
		Scale:       voxelScale,
		Amplitude:   voxelAmplitude,
		Seed:        voxelSeed,
		Threshold:   voxelThreshold,
		Octaves:     voxelOctaves,
		ChunkSize:   voxelChunkSize,
		WorldSize:   voxelWorldSize,
		Biome:       voxelBiome,
		TreeDensity: voxelTreeDensity,
	}
	
	// Ensure material has correct exposure (fixes dark voxels after deletion)
	if model.Material != nil {
		model.Material.Exposure = 1.0
	}
	
	// PERFORMANCE: Apply optimized rendering config for voxels (disables expensive effects)
	voxelConfig := renderer.PerformanceRenderingConfig()
	renderer.ApplyAdvancedRenderingConfig(model, voxelConfig)
	
	// Log the actual voxel count
	logToConsole(fmt.Sprintf("Voxel terrain created: %d voxels, %d chunks (%dx%d)", 
		world.ActiveVoxels, totalChunks, worldSize, worldSize), "success")
	
	return model
}

// generateVoxelTrees adds simple voxel trees to the world
func generateVoxelTrees(world *loader.VoxelWorld, noise *renderer.ImprovedPerlinNoise, seed int) {
	chunkSize := world.ChunkSize
	worldSizeX := world.WorldSizeX
	worldSizeZ := world.WorldSizeZ
	
	// Tree generation parameters
	trunkHeight := 4
	canopyRadius := 2
	
	// Try to place trees at intervals based on density
	gridSpacing := int(1.0 / float64(voxelTreeDensity))
	if gridSpacing < 3 {
		gridSpacing = 3
	}
	
	for cx := 0; cx < worldSizeX; cx++ {
		for cz := 0; cz < worldSizeZ; cz++ {
			for lx := 0; lx < chunkSize; lx += gridSpacing {
				for lz := 0; lz < chunkSize; lz += gridSpacing {
				// Add randomness to tree placement
				treeNoise := noise.Noise3D(float64(cx*chunkSize+lx+seed)*0.1, float64(cz*chunkSize+lz+seed)*0.1, float64(seed))
				
				// Use density directly as threshold (0.02 = 2% chance, 0.01 = 1% chance)
				if treeNoise > (0.8 - float64(voxelTreeDensity)*5.0) && treeNoise < (0.85 + float64(voxelTreeDensity)*5.0) {
						// Find surface height at this position
						gx := cx*chunkSize + lx
						gz := cz*chunkSize + lz
						
						// Find the highest solid voxel (surface)
						surfaceY := -1
						for y := world.MaxHeight - 1; y >= 0; y-- {
							if world.Chunks[cx][cz].Voxels[lx][y][lz].Active {
								surfaceY = y
								break
							}
						}
						
						// Only place trees on grass (VoxelID 1) and above water
						if surfaceY > 0 && surfaceY < world.MaxHeight-trunkHeight-canopyRadius-2 {
							if world.Chunks[cx][cz].Voxels[lx][surfaceY][lz].ID == 1 { // Grass
								placeTree(world, gx, surfaceY+1, gz, trunkHeight, canopyRadius)
							}
						}
					}
				}
			}
		}
	}
}

// placeTree places a simple voxel tree at the given position
func placeTree(world *loader.VoxelWorld, x, y, z, trunkHeight, canopyRadius int) {
	chunkSize := world.ChunkSize
	
	// Place trunk (VoxelID 5 = wood)
	for i := 0; i < trunkHeight; i++ {
		ty := y + i
		if ty < world.MaxHeight {
			cx := x / chunkSize
			cz := z / chunkSize
			lx := x % chunkSize
			lz := z % chunkSize
			
			if cx >= 0 && cx < world.WorldSizeX && cz >= 0 && cz < world.WorldSizeZ {
				world.Chunks[cx][cz].Voxels[lx][ty][lz] = loader.VoxelData{
					ID:       5, // Wood
					Position: mgl32.Vec3{float32(x), float32(ty), float32(z)},
					Active:   true,
				}
			}
		}
	}
	
	// Place canopy (VoxelID 6 = leaves) - spherical shape
	canopyY := y + trunkHeight
	for dx := -canopyRadius; dx <= canopyRadius; dx++ {
		for dy := -canopyRadius; dy <= canopyRadius; dy++ {
			for dz := -canopyRadius; dz <= canopyRadius; dz++ {
				// Spherical shape
				dist := float32(dx*dx + dy*dy + dz*dz)
				if dist <= float32(canopyRadius*canopyRadius) {
					tx := x + dx
					ty := canopyY + dy
					tz := z + dz
					
					if ty < world.MaxHeight && ty > 0 {
						cx := tx / chunkSize
						cz := tz / chunkSize
						lx := tx % chunkSize
						lz := tz % chunkSize
						
						if cx >= 0 && cx < world.WorldSizeX && cz >= 0 && cz < world.WorldSizeZ &&
							lx >= 0 && lx < chunkSize && lz >= 0 && lz < chunkSize {
							// Only place leaves if not replacing trunk
							if !(dx == 0 && dz == 0 && dy <= 0) {
							world.Chunks[cx][cz].Voxels[lx][ty][lz] = loader.VoxelData{
								ID:       6, // Leaves
								Position: mgl32.Vec3{float32(tx), float32(ty), float32(tz)},
								Active:   true,
							}
							}
						}
					}
				}
			}
		}
	}
}

func regenerateVoxelTerrain(config VoxelConfig) *renderer.Model {
	voxelScale = config.Scale
	voxelAmplitude = config.Amplitude
	voxelSeed = config.Seed
	voxelThreshold = config.Threshold
	voxelOctaves = config.Octaves
	voxelChunkSize = config.ChunkSize
	voxelWorldSize = config.WorldSize
	voxelBiome = config.Biome
	voxelTreeDensity = config.TreeDensity
	
	biomeNames := []string{"Plains", "Mountains", "Desert", "Islands", "Caves"}
	biomeName := "Unknown"
	if config.Biome >= 0 && config.Biome < 5 {
		biomeName = biomeNames[config.Biome]
	}
	
	logToConsole(fmt.Sprintf("Regenerating %s voxel terrain: Seed=%d, Scale=%.3f, Amp=%.1f", biomeName, voxelSeed, voxelScale, voxelAmplitude), "info")
	
	return generateVoxelTerrain()
}
