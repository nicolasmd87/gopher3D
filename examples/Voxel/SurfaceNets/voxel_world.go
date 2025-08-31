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

type SmoothTerrainBehaviour struct {
	engine          *engine.Gopher
	name            string
	voxelWorld      *loader.VoxelWorld
	noiseGen        *renderer.ImprovedPerlinNoise
	terrainModel    *renderer.Model
	lastUpdate      time.Time
	updateInterval  time.Duration
	renderingConfig renderer.AdvancedRenderingConfig
}

func NewSmoothTerrainBehaviour(engine *engine.Gopher) {
	terrainBehaviour := &SmoothTerrainBehaviour{
		engine:         engine,
		name:           "SmoothTerrain",
		lastUpdate:     time.Now(),
		updateInterval: time.Millisecond * 100,
	}
	behaviour.GlobalBehaviourManager.Add(terrainBehaviour)
}

func (st *SmoothTerrainBehaviour) GetName() string {
	return st.name
}

func (st *SmoothTerrainBehaviour) Start() {
	fmt.Println("Initializing Smooth Surface Nets Terrain...")

	st.engine.Light = renderer.CreateDirectionalLight(
		mgl32.Vec3{-0.4, -0.7, -0.3},
		mgl32.Vec3{1.0, 0.98, 0.9},
		3.5,
	)
	st.engine.Light.AmbientStrength = 0.2
	st.engine.Light.Temperature = 5500.0

	st.engine.Camera.InvertMouse = false
	st.engine.Camera.Position = mgl32.Vec3{64, 25, 64}
	st.engine.Camera.Speed = 50

	st.engine.SetFaceCulling(true)
	st.engine.SetFrustumCulling(false)
	st.engine.SetDebugMode(false)

	renderer.SetSkyboxColor(0.7, 0.8, 1.0)
	err := st.engine.SetSkybox("dark_sky")
	if err != nil {
		fmt.Printf("Could not set skybox: %v\n", err)
	}

	chunkSize := 16
	worldSizeX := 200
	worldSizeZ := 200
	maxHeight := 64
	voxelSize := float32(1.0)

	fmt.Printf("Creating Surface Nets world: %dx%d chunks, %d max height\n",
		worldSizeX, worldSizeZ, maxHeight)

	st.voxelWorld = loader.NewVoxelWorld(chunkSize, worldSizeX, worldSizeZ, maxHeight, voxelSize, nil, loader.SurfaceNetsMode)
	st.noiseGen = renderer.DefaultImprovedPerlinNoise()

	st.renderingConfig = renderer.VoxelAdvancedRenderingConfig()
	st.renderingConfig.EnablePerlinNoise = false
	st.renderingConfig.EnableAmbientOcclusion = true
	st.renderingConfig.AOIntensity = 0.3
	st.renderingConfig.AORadius = 200.0
	st.renderingConfig.EnableAdvancedShadows = true
	st.renderingConfig.ShadowIntensity = 0.4
	st.renderingConfig.ShadowSoftness = 0.2
	st.renderingConfig.EnableHighQualityFiltering = true
	st.renderingConfig.FilteringQuality = 3
	st.renderingConfig.AntiAliasing = true
	st.renderingConfig.EnableMeshSmoothing = true
	st.renderingConfig.TessellationQuality = 2

	fmt.Println("Generating smooth terrain using signed distance fields...")
	start := time.Now()

	st.generateTerrainSDF()

	fmt.Printf("Terrain SDF generation complete in %.2fs\n", time.Since(start).Seconds())

	fmt.Println("Creating Surface Nets mesh...")
	start = time.Now()

	terrainModel, err := st.voxelWorld.CreateInstancedModel()
	if err != nil || terrainModel == nil {
		fmt.Printf("Failed to create terrain model: %v\n", err)
		return
	}

	renderer.ApplyAdvancedRenderingConfig(terrainModel, st.renderingConfig)
	terrainModel.SetTexture("../../resources/textures/Grass.png")
	terrainModel.SetMatte(0.3, 0.7, 0.2)
	terrainModel.SetExposure(1.1)

	st.terrainModel = terrainModel
	st.engine.AddModel(terrainModel)

	meshTime := time.Since(start)
	fmt.Printf("Surface Nets mesh created in %.3fs\n", meshTime.Seconds())

	fmt.Println("Smooth Surface Nets Terrain initialized!")
	fmt.Println("Use WASD to move, mouse to look around")
}

func (st *SmoothTerrainBehaviour) generateTerrainSDF() {
	terrainScale := 0.02
	terrainAmplitude := 16.0

	st.voxelWorld.GenerateSDFParallel(func(x, y, z int) float32 {
		worldX := float64(x)
		worldZ := float64(z)

		surfaceHeight := st.noiseGen.Noise2D(
			worldX*terrainScale,
			worldZ*terrainScale,
		) * terrainAmplitude

		targetHeight := surfaceHeight + float64(st.voxelWorld.MaxHeight)/3
		distanceToSurface := float64(y) - targetHeight

		return float32(distanceToSurface)
	})
}

func (st *SmoothTerrainBehaviour) Update() {
}

func (st *SmoothTerrainBehaviour) UpdateFixed() {
}

func main() {
	engine := engine.NewGopher(engine.OPENGL)

	NewSmoothTerrainBehaviour(engine)

	engine.Width = 1920
	engine.Height = 1080

	engine.Render(100, 100)
}
