package main

import (
	behaviour "Gopher3D/internal/Behaviour"
	loader "Gopher3D/internal/Loader"
	"Gopher3D/internal/engine"
	"Gopher3D/internal/renderer"
	"fmt"
	"sync"
	"time"

	"github.com/go-gl/mathgl/mgl32"
)

const (
	GridSize      = 100   // Grid resolution
	GridSpacing   = 1.0   // Distance between vertices
	WaveSpeed     = 1.0   // Speed of wave propagation
	DampingFactor = 0.99  // Damping factor for wave energy
	Amplitude     = 700.0 // Initial wave height
	TimeStep      = 0.1   // Time step for wave simulation
	MaxWaves      = 100   // Maximum number of waves in the simulation
)

type WaterSimulation struct {
	vertices        [][]float32     // Heights of the grid vertices
	velocities      [][]float32     // Velocities of the grid vertices
	model           *renderer.Model // Water surface model
	engine          *engine.Gopher  // Engine instance
	mutex           sync.Mutex      // Mutex for concurrency
	shader          renderer.Shader // Custom water shader
	startTime       time.Time       // Time tracking for wave animation
	waveCount       int             // Number of waves
	waveDirections  []mgl32.Vec3    // Wave directions
	waveAmplitudes  []float32       // Wave amplitudes
	waveFrequencies []float32       // Wave frequencies
	waveSpeeds      []float32       // Wave speeds
}

func NewWaterSimulation(engine *engine.Gopher) {
	ws := &WaterSimulation{
		engine:          engine,
		vertices:        make([][]float32, GridSize),
		velocities:      make([][]float32, GridSize),
		shader:          renderer.InitWaterShader(), // Initialize your custom shader
		startTime:       time.Now(),
		waveCount:       MaxWaves,
		waveDirections:  make([]mgl32.Vec3, MaxWaves),
		waveAmplitudes:  make([]float32, MaxWaves),
		waveFrequencies: make([]float32, MaxWaves),
		waveSpeeds:      make([]float32, MaxWaves),
	}

	// Initialize grid
	for i := 0; i < GridSize; i++ {
		ws.vertices[i] = make([]float32, GridSize)
		ws.velocities[i] = make([]float32, GridSize)
	}

	// Initialize wave parameters
	for i := 0; i < MaxWaves; i++ {
		ws.waveDirections[i] = mgl32.Vec3{float32(i+1) * 0.3, 0.0, float32(i+1) * 0.3}.Normalize()
		ws.waveAmplitudes[i] = 5.0 / float32(i+1)
		ws.waveFrequencies[i] = 2.5 + 0.2*float32(i)
		ws.waveSpeeds[i] = 0.8 + 0.1*float32(i)
	}

	behaviour.GlobalBehaviourManager.Add(ws)
}

func main() {
	engine := engine.NewGopher(engine.OPENGL)
	engine.SetDebugMode(true)
	NewWaterSimulation(engine)

	engine.Width = 1980
	engine.Height = 1080

	engine.Render(0, 0)
}

func (ws *WaterSimulation) Start() {
	ws.engine.Camera.InvertMouse = false
	ws.engine.SetFrustumCulling(false)

	// Initialize camera
	ws.engine.Camera.Position = mgl32.Vec3{50, 50, 150}
	ws.engine.Camera.LookAt(mgl32.Vec3{50, 0, 50})
	ws.engine.Camera.Speed = 50

	// Initialize light
	ws.engine.Light = renderer.CreateLight()
	ws.engine.Light.Type = renderer.STATIC_LIGHT
	ws.engine.Light.Position = mgl32.Vec3{0, 200, 0}

	// Load the water surface model
	model, err := loader.LoadPlane(GridSize, GridSpacing)
	if err != nil {
		panic("Failed to load plane model: " + err.Error())
	}
	model.SetDiffuseColor(0.0, 0.0, 1.0) // Blue for water
	model.Shader = ws.shader             // Apply the water shader
	ws.model = model
	ws.engine.AddModel(model)

	// Add an initial disturbance at the center
	center := GridSize / 2
	ws.vertices[center][center] = Amplitude
}

func (ws *WaterSimulation) Update() {
	ws.UpdateShaderUniforms()
	ws.UpdateVertices()
	ws.UpdateModel()
}

func (ws *WaterSimulation) UpdateFixed() {}

// UpdateVertices calculates the wave propagation
func (ws *WaterSimulation) UpdateVertices() {
	newVertices := make([][]float32, GridSize)
	for i := 0; i < GridSize; i++ {
		newVertices[i] = make([]float32, GridSize)
	}

	for x := 1; x < GridSize-1; x++ {
		for z := 1; z < GridSize-1; z++ {
			// Average height of neighbors
			neighbors := ws.vertices[x-1][z] + ws.vertices[x+1][z] +
				ws.vertices[x][z-1] + ws.vertices[x][z+1]
			averageHeight := neighbors / 4.0

			// Compute velocity and new height
			acceleration := (averageHeight - ws.vertices[x][z]) * WaveSpeed
			ws.velocities[x][z] = (ws.velocities[x][z] + acceleration) * DampingFactor
			newVertices[x][z] = ws.vertices[x][z] + ws.velocities[x][z]*TimeStep
		}
	}

	ws.vertices = newVertices
}

// UpdateModel updates the vertex positions of the water model
func (ws *WaterSimulation) UpdateModel() {
	index := 0
	for x := 0; x < GridSize; x++ {
		for z := 0; z < GridSize; z++ {
			y := ws.vertices[x][z]
			ws.model.Vertices[index*3+1] = y
			index++
		}
	}
	ws.model.IsDirty = true
}

// UpdateShaderUniforms passes dynamic data to the shader
func (ws *WaterSimulation) UpdateShaderUniforms() {

	elapsedTime := float32(time.Since(ws.startTime).Seconds())
	ws.shader.SetFloat("time", elapsedTime)
	// Pass wave parameters
	ws.shader.SetInt("waveCount", int32(ws.waveCount))
	for i := 0; i < ws.waveCount; i++ {
		ws.shader.SetVec3(fmt.Sprintf("waveDirections[%d]", i), ws.waveDirections[i])
		ws.shader.SetFloat(fmt.Sprintf("waveAmplitudes[%d]", i), ws.waveAmplitudes[i])
		ws.shader.SetFloat(fmt.Sprintf("waveFrequencies[%d]", i), ws.waveFrequencies[i])
		ws.shader.SetFloat(fmt.Sprintf("waveSpeeds[%d]", i), ws.waveSpeeds[i])
	}

}
