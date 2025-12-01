package editor

import (
	"Gopher3D/internal/behaviour"
	"Gopher3D/internal/loader"
	"Gopher3D/internal/renderer"
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/inkyblackness/imgui-go/v4"
	"time"
)

var (
	voxelScale       = float32(0.05)
	voxelAmplitude   = float32(15.0)
	voxelSeed        = int32(time.Now().Unix())
	voxelThreshold   = float32(0.2)
	voxelOctaves     = int32(4)
	voxelChunkSize   = int32(32)
	voxelWorldSize   = int32(2)
	voxelBiome       = int32(0)
	voxelTreeDensity = float32(0.02)

	voxelColorGrass  = [3]float32{0.3, 0.7, 0.2}
	voxelColorDirt   = [3]float32{0.6, 0.4, 0.2}
	voxelColorStone  = [3]float32{0.5, 0.5, 0.5}
	voxelColorSand   = [3]float32{0.9, 0.8, 0.5}
	voxelColorWood   = [3]float32{0.4, 0.25, 0.1}
	voxelColorLeaves = [3]float32{0.2, 0.6, 0.2}
)

type VoxelConfig struct {
	// Legacy fields
	Scale       float32    `json:"scale"`
	Amplitude   float32    `json:"amplitude"`
	Seed        int32      `json:"seed"`
	Threshold   float32    `json:"threshold"`
	Octaves     int32      `json:"octaves"`
	ChunkSize   int32      `json:"chunk_size"`
	WorldSize   int32      `json:"world_size"`
	Biome       int32      `json:"biome"`
	TreeDensity float32    `json:"tree_density"`
	ColorGrass  [3]float32 `json:"color_grass"`
	ColorDirt   [3]float32 `json:"color_dirt"`
	ColorStone  [3]float32 `json:"color_stone"`
	ColorSand   [3]float32 `json:"color_sand"`
	ColorWood   [3]float32 `json:"color_wood"`
	ColorLeaves [3]float32 `json:"color_leaves"`

	// New component-based fields
	WorldSizeX  int     `json:"world_size_x,omitempty"`
	WorldSizeY  int     `json:"world_size_y,omitempty"`
	WorldSizeZ  int     `json:"world_size_z,omitempty"`
	VoxelSize   float32 `json:"voxel_size,omitempty"`
	NoiseScale  float32 `json:"noise_scale,omitempty"`
	HeightScale float32 `json:"height_scale,omitempty"`
	TerrainType string  `json:"terrain_type,omitempty"`
}

func renderAddVoxelDialog() {
	if Eng == nil {
		return
	}
	imgui.OpenPopup("Add Voxel Terrain")
	centerX := float32(Eng.Width) / 2
	centerY := float32(Eng.Height) / 2
	imgui.SetNextWindowPosV(imgui.Vec2{X: centerX - 250, Y: centerY - 280}, imgui.ConditionAppearing, imgui.Vec2{})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 500, Y: 560}, imgui.ConditionAppearing)

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

		if imgui.CollapsingHeaderV("Tile Colors", imgui.TreeNodeFlagsNone) {
			imgui.ColorEdit3V("Grass", &voxelColorGrass, 0)
			imgui.ColorEdit3V("Dirt", &voxelColorDirt, 0)
			imgui.ColorEdit3V("Stone", &voxelColorStone, 0)
			imgui.ColorEdit3V("Sand", &voxelColorSand, 0)
			imgui.ColorEdit3V("Wood (Trunk)", &voxelColorWood, 0)
			imgui.ColorEdit3V("Leaves", &voxelColorLeaves, 0)

			imgui.Spacing()
			if imgui.Button("Reset Colors") {
				voxelColorGrass = [3]float32{0.3, 0.7, 0.2}
				voxelColorDirt = [3]float32{0.6, 0.4, 0.2}
				voxelColorStone = [3]float32{0.5, 0.5, 0.5}
				voxelColorSand = [3]float32{0.9, 0.8, 0.5}
				voxelColorWood = [3]float32{0.4, 0.25, 0.1}
				voxelColorLeaves = [3]float32{0.2, 0.6, 0.2}
			}

			imgui.Spacing()
			imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.6, Y: 0.6, Z: 0.6, W: 1})
			imgui.Text("Note: Texture support planned.")
			imgui.Text("Currently only solid colors.")
			imgui.PopStyleColor()
		}

		imgui.Spacing()
		imgui.Separator()

		if imgui.Button("Generate Terrain") {
			// Create GameObject with VoxelTerrainComponent
			obj := createVoxelTerrainGameObject()
			if obj != nil {
				logToConsole("Voxel terrain GameObject created", "info")
			}
			ShowAddVoxel = false
			imgui.CloseCurrentPopup()
		}
		imgui.SameLine()
		if imgui.Button("Cancel") {
			ShowAddVoxel = false
			imgui.CloseCurrentPopup()
		}
		imgui.EndPopup()
	}
}

// createVoxelTerrainGameObject creates a GameObject with a VoxelTerrainComponent
// using the current editor settings
func createVoxelTerrainGameObject() *behaviour.GameObject {
	// Create the component with current editor settings
	voxelComp := behaviour.NewVoxelTerrainComponent()
	voxelComp.Scale = voxelScale
	voxelComp.Amplitude = voxelAmplitude
	voxelComp.Seed = voxelSeed
	voxelComp.Threshold = voxelThreshold
	voxelComp.Octaves = voxelOctaves
	voxelComp.ChunkSize = voxelChunkSize
	voxelComp.WorldSize = voxelWorldSize
	voxelComp.Biome = voxelBiome
	voxelComp.TreeDensity = voxelTreeDensity
	voxelComp.GrassColor = voxelColorGrass
	voxelComp.DirtColor = voxelColorDirt
	voxelComp.StoneColor = voxelColorStone
	voxelComp.SandColor = voxelColorSand
	voxelComp.WoodColor = voxelColorWood
	voxelComp.LeavesColor = voxelColorLeaves

	// Create GameObject
	obj := behaviour.NewGameObject(fmt.Sprintf("Voxel Terrain (%dx%d)", voxelWorldSize, voxelWorldSize))
	obj.AddComponent(voxelComp)

	// Generate the terrain mesh
	model := generateVoxelTerrainFromComponent(voxelComp)
	if model != nil {
		voxelComp.Model = model
		voxelComp.Generated = true
		obj.SetModel(model)
		Eng.AddModel(model)
		// Register model-to-GameObject mapping for scene saving
		registerModelToGameObject(model, obj)
	}

	// Register the GameObject
	behaviour.GlobalComponentManager.RegisterGameObject(obj)

	return obj
}

// generateVoxelTerrainFromComponent generates voxel terrain from a component's settings
func generateVoxelTerrainFromComponent(comp *behaviour.VoxelTerrainComponent) *renderer.Model {
	chunkSize := int(comp.ChunkSize)
	worldSize := int(comp.WorldSize)
	voxelSizeVal := float32(1.0)

	// Set voxel colors from component
	loader.ClearCustomVoxelColors()
	loader.SetVoxelColor(1, mgl32.Vec3{comp.GrassColor[0], comp.GrassColor[1], comp.GrassColor[2]})
	loader.SetVoxelColor(2, mgl32.Vec3{comp.DirtColor[0], comp.DirtColor[1], comp.DirtColor[2]})
	loader.SetVoxelColor(3, mgl32.Vec3{comp.StoneColor[0], comp.StoneColor[1], comp.StoneColor[2]})
	loader.SetVoxelColor(4, mgl32.Vec3{comp.SandColor[0], comp.SandColor[1], comp.SandColor[2]})
	loader.SetVoxelColor(5, mgl32.Vec3{comp.WoodColor[0], comp.WoodColor[1], comp.WoodColor[2]})
	loader.SetVoxelColor(6, mgl32.Vec3{comp.LeavesColor[0], comp.LeavesColor[1], comp.LeavesColor[2]})

	noise := renderer.NewImprovedPerlinNoise(int64(comp.Seed))
	world := loader.NewVoxelWorld(chunkSize, worldSize, worldSize, 64, voxelSizeVal, loader.CreateCubeGeometry(voxelSizeVal), loader.InstancedMode)

	// Use component settings for generation
	scale := float64(comp.Scale)
	amplitude := comp.Amplitude
	threshold := float64(comp.Threshold)
	octaves := int(comp.Octaves)
	biome := comp.Biome
	treeDensity := comp.TreeDensity

	world.GenerateVoxelsParallel(func(x, y, z int) (loader.VoxelID, bool) {
		fx, fy, fz := float64(x)*scale, float64(y)*scale, float64(z)*scale
		n3d := noise.Turbulence(fx, fy, fz, octaves, 0.5)

		switch biome {
		case 0: // Plains
			h := noise.Turbulence2D(fx, fz, octaves, 0.5)
			height := int(float32(h)*amplitude) + 15
			if y <= height {
				if n3d > threshold {
					return 0, false
				}
				if y == height {
					return 1, true
				} else if y > height-3 {
					return 2, true
				}
				return 3, true
			}
		case 1: // Mountains
			h := noise.Turbulence2D(fx, fz, octaves, 0.6)
			height := int(float32(h)*amplitude) + 20
			peakNoise := noise.Turbulence(fx, fy*0.3, fz, octaves, 0.5)
			if peakNoise > 0.3 {
				height += int(float32(peakNoise-0.3) * 15.0)
			}
			if y <= height {
				if n3d > threshold && y < height-5 {
					return 0, false
				}
				if y == height && y < 35 {
					return 1, true
				} else if y == height {
					return 3, true
				} else if y > height-2 {
					return 2, true
				}
				return 3, true
			}
		case 2: // Desert
			h := noise.Turbulence2D(fx*2.0, fz*2.0, octaves-1, 0.4)
			height := int(float32(h)*amplitude) + 12
			if y <= height {
				return 4, true
			}
		case 3: // Islands
			h := noise.Turbulence2D(fx, fz, octaves, 0.5)
			islandNoise := noise.Turbulence2D(fx*0.5, fz*0.5, 2, 0.5)
			waterLevel := 18
			height := int(float32(h)*amplitude) + 10
			if islandNoise > 0.2 {
				height += int((float32(islandNoise) - 0.2) * 20.0)
			}
			if y <= height {
				if n3d > threshold && y > waterLevel {
					return 0, false
				}
				if y == height && y > waterLevel {
					return 1, true
				} else if y > height-2 && y > waterLevel {
					return 2, true
				} else if y <= waterLevel {
					return 3, true
				}
				return 3, true
			}
		case 4: // Caves
			h := noise.Turbulence2D(fx, fz, octaves, 0.5)
			height := int(float32(h)*amplitude) + 15
			cave1 := noise.Turbulence(fx, fy, fz, octaves, 0.5)
			cave2 := noise.Turbulence(fx*2.0, fy*2.0, fz*2.0, octaves-1, 0.5)
			cave3 := noise.Turbulence(fx*0.5, fy*0.5, fz*0.5, 2, 0.5)
			combinedCave := (cave1 + cave2*0.5 + cave3*0.3) / 1.8
			if y <= height {
				if combinedCave > threshold {
					return 0, false
				}
				if y == height {
					return 1, true
				} else if y > height-3 {
					return 2, true
				}
				return 3, true
			}
		}
		return 0, false
	})

	// Generate trees if enabled
	if treeDensity > 0.001 {
		generateVoxelTrees(world, noise, int(comp.Seed))
	}

	model, err := world.CreateInstancedModel()
	if err != nil {
		logToConsole(fmt.Sprintf("Failed to create voxel mesh: %v", err), "error")
		return nil
	}

	// Log the voxel count
	totalChunks := worldSize * worldSize
	logToConsole(fmt.Sprintf("Voxel terrain created: %d voxels, %d chunks (%dx%d)",
		world.ActiveVoxels, totalChunks, worldSize, worldSize), "success")

	model.Name = fmt.Sprintf("Voxel Terrain (%dx%d)", worldSize, worldSize)
	model.SetPosition(0, 0, 0)

	// Ensure material has proper values for lighting
	if model.Material == nil {
		model.Material = &renderer.Material{
			Name:         "VoxelMaterial",
			DiffuseColor: [3]float32{1.0, 1.0, 1.0}, // White - instance colors provide the actual color
			Metallic:     0.0,
			Roughness:    0.9,
			Alpha:        1.0,
			Exposure:     1.0,
		}
	} else {
		// CRITICAL: Set all material properties, not just exposure/alpha
		model.Material.DiffuseColor = [3]float32{1.0, 1.0, 1.0} // White - instance colors provide the actual color
		model.Material.Metallic = 0.0
		model.Material.Roughness = 0.9
		model.Material.Exposure = 1.0
		model.Material.Alpha = 1.0
	}

	if model.Metadata == nil {
		model.Metadata = make(map[string]interface{})
	}
	model.Metadata["isVoxel"] = true
	model.Metadata["type"] = "voxel_terrain"
	// Store voxel config for scene saving/loading
	model.Metadata["voxelConfig"] = VoxelConfig{
		Scale:       comp.Scale,
		Amplitude:   comp.Amplitude,
		Seed:        comp.Seed,
		Threshold:   comp.Threshold,
		Octaves:     comp.Octaves,
		ChunkSize:   comp.ChunkSize,
		WorldSize:   comp.WorldSize,
		Biome:       comp.Biome,
		TreeDensity: comp.TreeDensity,
		ColorGrass:  comp.GrassColor,
		ColorDirt:   comp.DirtColor,
		ColorStone:  comp.StoneColor,
		ColorSand:   comp.SandColor,
		ColorWood:   comp.WoodColor,
		ColorLeaves: comp.LeavesColor,
	}

	return model
}

// generateVoxelTerrainForGameObject generates terrain for a new GameObject
func generateVoxelTerrainForGameObject(c *behaviour.VoxelTerrainComponent, obj *behaviour.GameObject) {
	model := generateVoxelTerrainFromComponent(c)
	if model != nil {
		c.Model = model
		c.Generated = true
		obj.SetModel(model)
		Eng.AddModel(model)
		logToConsole(fmt.Sprintf("Generated voxel terrain: %s", obj.Name), "info")
	}
}

// regenerateVoxelTerrainForComponent regenerates terrain for an existing component
func regenerateVoxelTerrainForComponent(c *behaviour.VoxelTerrainComponent, obj *behaviour.GameObject) {
	// Remove old model if exists
	if oldModel, ok := c.Model.(*renderer.Model); ok && oldModel != nil {
		openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
		if ok {
			openglRenderer.RemoveModel(oldModel)
		}
	}

	// Generate new terrain
	model := generateVoxelTerrainFromComponent(c)
	if model != nil {
		c.Model = model
		c.Generated = true
		obj.SetModel(model)
		Eng.AddModel(model)
		logToConsole(fmt.Sprintf("Regenerated voxel terrain: %s", obj.Name), "info")
	}
}

func generateVoxelTerrain() *renderer.Model {
	chunkSize := int(voxelChunkSize)
	worldSize := int(voxelWorldSize)
	voxelSizeVal := float32(1.0)

	loader.ClearCustomVoxelColors()
	loader.SetVoxelColor(1, mgl32.Vec3{voxelColorGrass[0], voxelColorGrass[1], voxelColorGrass[2]})
	loader.SetVoxelColor(2, mgl32.Vec3{voxelColorDirt[0], voxelColorDirt[1], voxelColorDirt[2]})
	loader.SetVoxelColor(3, mgl32.Vec3{voxelColorStone[0], voxelColorStone[1], voxelColorStone[2]})
	loader.SetVoxelColor(4, mgl32.Vec3{voxelColorSand[0], voxelColorSand[1], voxelColorSand[2]})
	loader.SetVoxelColor(5, mgl32.Vec3{voxelColorWood[0], voxelColorWood[1], voxelColorWood[2]})
	loader.SetVoxelColor(6, mgl32.Vec3{voxelColorLeaves[0], voxelColorLeaves[1], voxelColorLeaves[2]})

	noise := renderer.NewImprovedPerlinNoise(int64(voxelSeed))
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
	model.Metadata["type"] = "voxel_terrain"
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
		ColorGrass:  voxelColorGrass,
		ColorDirt:   voxelColorDirt,
		ColorStone:  voxelColorStone,
		ColorSand:   voxelColorSand,
		ColorWood:   voxelColorWood,
		ColorLeaves: voxelColorLeaves,
	}

	// Ensure material has correct exposure and lighting properties (fixes dark voxels after deletion)
	if model.Material != nil {
		model.Material.Exposure = 1.0
		model.Material.Alpha = 1.0
		// Ensure diffuse color is not black
		if model.Material.DiffuseColor[0] == 0 && model.Material.DiffuseColor[1] == 0 && model.Material.DiffuseColor[2] == 0 {
			model.Material.DiffuseColor = [3]float32{0.8, 0.8, 0.8}
		}
	}

	// Force model matrix recalculation
	model.IsDirty = true

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
					if treeNoise > (0.8-float64(voxelTreeDensity)*5.0) && treeNoise < (0.85+float64(voxelTreeDensity)*5.0) {
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

// generateVoxelTerrainNew creates a voxel terrain from new-style config (for component system)
func generateVoxelTerrainNew(config *VoxelConfig, name string) *renderer.Model {
	if config == nil {
		return nil
	}

	// Use new fields if available, otherwise fall back to legacy
	chunkSize := int(config.ChunkSize)
	if chunkSize == 0 {
		chunkSize = 32
	}

	worldSizeX := config.WorldSizeX
	worldSizeZ := config.WorldSizeZ
	maxHeight := config.WorldSizeY

	if worldSizeX == 0 {
		worldSizeX = int(config.WorldSize)
		if worldSizeX == 0 {
			worldSizeX = 2
		}
	}
	if worldSizeZ == 0 {
		worldSizeZ = worldSizeX
	}
	if maxHeight == 0 {
		maxHeight = 32
	}

	voxelSize := config.VoxelSize
	if voxelSize == 0 {
		voxelSize = 1.0
	}

	noiseScale := config.NoiseScale
	if noiseScale == 0 {
		noiseScale = float32(config.Scale)
		if noiseScale == 0 {
			noiseScale = 0.05
		}
	}

	heightScale := config.HeightScale
	if heightScale == 0 {
		heightScale = float32(config.Amplitude)
		if heightScale == 0 {
			heightScale = 20.0
		}
	}

	seed := config.Seed
	if seed == 0 {
		seed = voxelSeed
	}

	// Create voxel world using the existing API
	geometry := loader.CreateCubeGeometry(voxelSize)
	world := loader.NewVoxelWorld(chunkSize, worldSizeX, worldSizeZ, maxHeight, voxelSize, geometry, loader.InstancedMode)
	noise := renderer.NewImprovedPerlinNoise(int64(seed))

	// Generate terrain
	world.GenerateVoxelsParallel(func(x, y, z int) (loader.VoxelID, bool) {
		fx := float64(x) * float64(noiseScale)
		fz := float64(z) * float64(noiseScale)

		h := noise.Turbulence2D(fx, fz, 4, 0.5)
		height := int(float32(h) * heightScale)

		if y <= height {
			if y == height {
				return loader.VoxelID(1), true // Grass
			} else if y > height-3 {
				return loader.VoxelID(2), true // Dirt
			}
			return loader.VoxelID(3), true // Stone
		}
		return loader.VoxelID(0), false
	})

	model, err := world.CreateInstancedModel()
	if err != nil {
		logToConsole(fmt.Sprintf("Failed to create voxel mesh: %v", err), "error")
		return nil
	}

	if name == "" {
		name = fmt.Sprintf("Voxel Terrain (%dx%dx%d)", worldSizeX*chunkSize, maxHeight, worldSizeZ*chunkSize)
	}
	model.Name = name
	model.SetPosition(0, 0, 0)
	model.SetDiffuseColor(0.5, 0.5, 0.5)
	model.SetMaterialPBR(0.0, 0.9)
	model.SetExposure(1.0)
	model.Material.Alpha = 1.0
	model.IsDirty = true

	// Store configuration in metadata
	if model.Metadata == nil {
		model.Metadata = make(map[string]interface{})
	}
	model.Metadata["isVoxel"] = true
	model.Metadata["voxelConfig"] = config

	return model
}
