package main

import (
	behaviour "Gopher3D/internal/behaviour"
	"Gopher3D/internal/engine"
	loader "Gopher3D/internal/loader"
	"Gopher3D/internal/renderer"
	"fmt"
	"time"

	"github.com/go-gl/mathgl/mgl32"
)

const (
	AIR   loader.VoxelID = 0
	GRASS loader.VoxelID = 1
	DIRT  loader.VoxelID = 2
	STONE loader.VoxelID = 3
	WATER loader.VoxelID = 4
	SAND  loader.VoxelID = 5
)

type VoxelProperties struct {
	Name        string
	TexturePath string
	Color       mgl32.Vec3
	Solid       bool
	Transparent bool
	Luminous    bool
	Hardness    float32
}

type VoxelRegistry struct {
	voxelTypes map[loader.VoxelID]VoxelProperties
}

func NewVoxelRegistry() *VoxelRegistry {
	registry := &VoxelRegistry{
		voxelTypes: make(map[loader.VoxelID]VoxelProperties),
	}

	registry.Register(AIR, VoxelProperties{
		Name: "Air", Color: mgl32.Vec3{0, 0, 0},
		Solid: false, Transparent: true, Luminous: false, Hardness: 0,
	})
	registry.Register(GRASS, VoxelProperties{
		Name: "Grass", Color: mgl32.Vec3{0.3, 0.7, 0.2},
		Solid: true, Transparent: false, Luminous: false, Hardness: 1.0,
	})
	registry.Register(DIRT, VoxelProperties{
		Name: "Dirt", Color: mgl32.Vec3{0.5, 0.3, 0.1},
		Solid: true, Transparent: false, Luminous: false, Hardness: 1.5,
	})
	registry.Register(STONE, VoxelProperties{
		Name: "Stone", Color: mgl32.Vec3{0.6, 0.6, 0.6},
		Solid: true, Transparent: false, Luminous: false, Hardness: 3.0,
	})
	registry.Register(WATER, VoxelProperties{
		Name: "Water", Color: mgl32.Vec3{0.2, 0.4, 0.8},
		Solid: false, Transparent: true, Luminous: false, Hardness: 0,
	})
	registry.Register(SAND, VoxelProperties{
		Name: "Sand", Color: mgl32.Vec3{0.9, 0.8, 0.6},
		Solid: true, Transparent: false, Luminous: false, Hardness: 0.8,
	})

	return registry
}

func (r *VoxelRegistry) Register(id loader.VoxelID, properties VoxelProperties) {
	r.voxelTypes[id] = properties
}

func (r *VoxelRegistry) Get(id loader.VoxelID) (VoxelProperties, bool) {
	props, exists := r.voxelTypes[id]
	return props, exists
}

func GenerateExampleTerrain(world *loader.VoxelWorld, noiseGen *renderer.ImprovedPerlinNoise) {
	terrainScale := 0.015
	terrainAmplitude := 32.0
	terrainOctaves := 6
	waterLevel := 12

	for chunkX := 0; chunkX < world.WorldSizeX; chunkX++ {
		for chunkZ := 0; chunkZ < world.WorldSizeZ; chunkZ++ {
			for localX := 0; localX < world.ChunkSize; localX++ {
				for localZ := 0; localZ < world.ChunkSize; localZ++ {
					worldX := float64(chunkX*world.ChunkSize + localX)
					worldZ := float64(chunkZ*world.ChunkSize + localZ)

					baseHeight := noiseGen.Turbulence2D(
						worldX*terrainScale,
						worldZ*terrainScale,
						terrainOctaves,
						0.6,
					)

					ridges := noiseGen.Ridge(
						worldX*terrainScale*2.0,
						0,
						worldZ*terrainScale*2.0,
						2,
						0.4,
					) * 0.4

					marble := noiseGen.Marble(
						worldX*terrainScale*3.0,
						0,
						worldZ*terrainScale*3.0,
						12.0,
					) * 0.15

					surfaceDetail := noiseGen.Noise2D(
						worldX*terrainScale*8.0,
						worldZ*terrainScale*8.0,
					) * 0.1

					combinedHeight := baseHeight*0.5 + ridges*0.3 + marble*0.15 + surfaceDetail*0.05
					terrainHeight := int(combinedHeight*terrainAmplitude) + world.MaxHeight/3

					if terrainHeight < 0 {
						terrainHeight = 0
					}
					if terrainHeight >= world.MaxHeight {
						terrainHeight = world.MaxHeight - 1
					}

					temperatureNoise := noiseGen.Noise2D(worldX*0.001, worldZ*0.001)
					moistureNoise := noiseGen.Noise2D(worldX*0.002+1000, worldZ*0.002+1000)

					for y := 0; y < world.MaxHeight; y++ {
						voxelID := AIR

						if y <= terrainHeight {
							if y <= waterLevel {
								if temperatureNoise > 0.3 {
									voxelID = SAND
								} else {
									voxelID = STONE
								}
							} else if y == terrainHeight && y > waterLevel {
								if moistureNoise > 0.2 && temperatureNoise > 0.1 {
									voxelID = GRASS
								} else if temperatureNoise > 0.4 {
									voxelID = SAND
								} else {
									voxelID = DIRT
								}
							} else if y > terrainHeight-3 {
								if moistureNoise > 0.3 {
									voxelID = DIRT
								} else {
									voxelID = STONE
								}
							} else {
								voxelID = STONE
							}
						} else if y <= waterLevel {
							voxelID = WATER
						}

						if voxelID != AIR {
							world.SetVoxel(chunkX*world.ChunkSize+localX, y, chunkZ*world.ChunkSize+localZ, voxelID)
						}
					}
				}
			}
		}
	}
}

type VoxelWorldBehaviour struct {
	engine          *engine.Gopher
	name            string
	voxelWorld      *loader.VoxelWorld
	voxelModel      *renderer.Model
	noiseGen        *renderer.ImprovedPerlinNoise
	registry        *VoxelRegistry
	lastUpdate      time.Time
	updateInterval  time.Duration
	renderingConfig renderer.AdvancedRenderingConfig
}

func NewVoxelWorldBehaviour(engine *engine.Gopher) {
	voxelBehaviour := &VoxelWorldBehaviour{
		engine:         engine,
		name:           "VoxelWorld",
		lastUpdate:     time.Now(),
		updateInterval: time.Millisecond * 100, // Update chunks every 100ms
	}
	behaviour.GlobalBehaviourManager.Add(voxelBehaviour)
}

func main() {
	engine := engine.NewGopher(engine.OPENGL) // or engine.VULKAN

	NewVoxelWorldBehaviour(engine)

	engine.Width = 1920
	engine.Height = 1080

	// WINDOW POS IN X,Y AND MODEL
	engine.Render(100, 100)
}

func (vb *VoxelWorldBehaviour) Start() {
	// Natural directional lighting for voxels
	vb.engine.Light = renderer.CreateDirectionalLight(
		mgl32.Vec3{-0.4, -0.7, -0.3}, // More natural sun angle
		mgl32.Vec3{1.0, 0.98, 0.9},   // Natural sunlight (less orange)
		3.5,                          // Moderate intensity
	)
	vb.engine.Light.AmbientStrength = 0.2 // Higher ambient for visibility
	vb.engine.Light.Temperature = 5500.0  // Natural daylight temperature

	// Configure camera for cinematic voxel exploration
	vb.engine.Camera.InvertMouse = false
	vb.engine.Camera.Position = mgl32.Vec3{120, 45, 120} // Better starting position
	vb.engine.Camera.Speed = 75                          // Faster movement for exploration

	// Enable optimizations for voxel rendering
	vb.engine.SetFaceCulling(true)
	vb.engine.SetFrustumCulling(false) // Disable frustum culling to fix disappearing voxels

	// Natural sky colors
	renderer.SetSkyboxColor(0.7, 0.8, 1.0) // More natural blue sky
	vb.engine.SetSkybox("dark_sky")        // Use existing skybox

	chunkSize := 16
	worldSizeX := 200
	worldSizeZ := 200
	maxHeight := 32
	voxelSize := float32(1.0)

	cubeGeometry := loader.CreateCubeGeometry(voxelSize)
	vb.voxelWorld = loader.NewVoxelWorld(chunkSize, worldSizeX, worldSizeZ, maxHeight, voxelSize, cubeGeometry, loader.InstancedMode)
	vb.noiseGen = renderer.DefaultImprovedPerlinNoise()
	vb.registry = NewVoxelRegistry()

	// CINEMATIC rendering configuration for STUNNING visuals
	vb.renderingConfig = renderer.VoxelAdvancedRenderingConfig()
	// MAXED OUT settings for cinematic quality
	vb.renderingConfig.EnablePerlinNoise = false // We handle noise in terrain generation
	vb.renderingConfig.EnableAmbientOcclusion = true
	vb.renderingConfig.AOIntensity = 0.3 // Natural AO for depth
	vb.renderingConfig.AORadius = 200.0  // Wider AO radius
	vb.renderingConfig.EnableAdvancedShadows = true
	vb.renderingConfig.ShadowIntensity = 0.4 // Natural shadows
	vb.renderingConfig.ShadowSoftness = 0.2  // Softer shadow edges
	vb.renderingConfig.EnableHighQualityFiltering = true
	vb.renderingConfig.FilteringQuality = 3        // MAXIMUM quality
	vb.renderingConfig.AntiAliasing = true         // Smooth edges
	vb.renderingConfig.EnableMeshSmoothing = false // Keep voxels crisp and blocky
	vb.renderingConfig.TessellationQuality = 1     // Low tessellation for blocks

	start := time.Now()

	totalChunks := worldSizeX * worldSizeZ

	vb.voxelWorld.GenerateVoxelsParallel(func(x, y, z int) (loader.VoxelID, bool) {
		worldX := float64(x)
		worldZ := float64(z)
		waterLevel := 8

		terrainNoise := vb.noiseGen.Noise2D(worldX*0.01, worldZ*0.01) * 15
		caveNoise := vb.noiseGen.Noise3D(worldX*0.05, float64(y)*0.05, worldZ*0.05)
		terrainHeight := int(terrainNoise) + 10

		if caveNoise > 0.4 {
			return AIR, false
		}

		temperatureNoise := vb.noiseGen.Noise2D(worldX*0.001+500, worldZ*0.001+500)
		moistureNoise := vb.noiseGen.Noise2D(worldX*0.002+1000, worldZ*0.002+1000)

		var voxelID loader.VoxelID = AIR

		if y <= terrainHeight {
			if y <= waterLevel {
				if temperatureNoise > 0.3 {
					voxelID = SAND
				} else {
					voxelID = STONE
				}
			} else if y == terrainHeight && y > waterLevel {
				if moistureNoise > 0.2 && temperatureNoise > 0.1 {
					voxelID = GRASS
				} else if temperatureNoise > 0.4 {
					voxelID = SAND
				} else {
					voxelID = DIRT
				}
			} else if y > terrainHeight-3 {
				if moistureNoise > 0.3 {
					voxelID = DIRT
				} else {
					voxelID = STONE
				}
			} else {
				voxelID = STONE
			}
		} else if y <= waterLevel {
			voxelID = WATER
		}

		return voxelID, voxelID != AIR
	})

	start = time.Now()

	voxelModel, err := vb.voxelWorld.CreateInstancedModel()
	if err != nil {
		panic(fmt.Sprintf("Failed to create instanced voxel model: %v", err))
	}

	// Apply advanced rendering configuration
	renderer.ApplyAdvancedRenderingConfig(voxelModel, vb.renderingConfig)

	// Natural materials for voxels
	voxelModel.SetTexture("../../resources/textures/Grass.png")
	voxelModel.SetMatte(0.3, 0.7, 0.2) // Natural grass colors
	voxelModel.SetExposure(1.1)        // Natural exposure

	// Store the model and add to engine
	vb.voxelModel = voxelModel
	vb.engine.AddModel(voxelModel)

	instancingTime := time.Since(start)
	fmt.Printf("Instanced model created in %.3fs with %d instances\n",
		instancingTime.Seconds(), vb.voxelWorld.ActiveVoxels)

	fmt.Printf("World stats: %d chunks, %d voxels, 1 draw call!\n",
		totalChunks, vb.voxelWorld.ActiveVoxels)
}

func (vb *VoxelWorldBehaviour) Update() {

}

func (vb *VoxelWorldBehaviour) UpdateFixed() {

}
