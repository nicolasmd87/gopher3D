package main

import (
	behaviour "Gopher3D/internal/behaviour"
	"Gopher3D/internal/engine"
	loader "Gopher3D/internal/loader"
	"Gopher3D/internal/renderer"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/go-gl/mathgl/mgl32"
)

const (
	OceanSize       = 1000.0 // Much bigger ocean size for vast feel
	WaterResolution = 2048   // Water mesh resolution (higher = smoother, lower = faster)
	WaveSpeed       = 1.5    // Faster wave movement for more dynamic ocean
	Amplitude       = 0.8    // Wave amplitude - controls wave height (more realistic)
	MaxWaves        = 4      // Fewer waves for more realistic ocean (not too complex)
)

type WaterSimulation struct {
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
	currentTime     float32         // Current elapsed time
}

func NewWaterSimulation(engine *engine.Gopher) {
	ws := &WaterSimulation{
		engine:          engine,
		shader:          renderer.InitWaterShader(), // Initialize your custom shader
		startTime:       time.Now(),
		waveCount:       MaxWaves,
		waveDirections:  make([]mgl32.Vec3, MaxWaves),
		waveAmplitudes:  make([]float32, MaxWaves),
		waveFrequencies: make([]float32, MaxWaves),
		waveSpeeds:      make([]float32, MaxWaves),
	}

	// Initialize wave parameters - much more varied and random ocean patterns
	// Use different base directions instead of just wind variation
	baseDirections := []float32{
		30.0 * math.Pi / 180.0,  // Northeast
		120.0 * math.Pi / 180.0, // Southeast
		200.0 * math.Pi / 180.0, // Southwest
		300.0 * math.Pi / 180.0, // Northwest
	}

	for i := 0; i < MaxWaves; i++ {
		// Use completely different base directions for each wave
		baseAngle := baseDirections[i%len(baseDirections)]

		// Add random variation to each wave direction
		randomVariation := float32(math.Sin(float64(i*7))) * 40.0 * math.Pi / 180.0 // 40 degree random variation

		waveAngle := baseAngle + randomVariation

		dirX := float32(math.Cos(float64(waveAngle)))
		dirZ := float32(math.Sin(float64(waveAngle)))
		ws.waveDirections[i] = mgl32.Vec3{dirX, 0.0, dirZ}.Normalize()

		// Much more varied amplitudes
		amplitudeVariation := []float32{1.0, 0.4, 0.7, 0.2} // Very different sizes
		ws.waveAmplitudes[i] = Amplitude * amplitudeVariation[i%len(amplitudeVariation)]

		// Very different frequencies for varied wave spacing
		frequencyVariation := []float32{0.05, 0.12, 0.08, 0.15} // Much more varied spacing
		ws.waveFrequencies[i] = frequencyVariation[i%len(frequencyVariation)]

		// Much more varied speeds to break synchronization
		speedVariation := []float32{1.0, 1.8, 0.6, 1.3} // Very different speeds
		ws.waveSpeeds[i] = WaveSpeed * speedVariation[i%len(speedVariation)]
	}

	behaviour.GlobalBehaviourManager.Add(ws)
}

func main() {
	fmt.Println("DEBUG: Starting water example")

	engine := engine.NewGopher(engine.OPENGL)
	engine.SetDebugMode(false) // Turn off wireframe for clearer view
	NewWaterSimulation(engine)

	engine.Width = 1980
	engine.Height = 1080

	fmt.Println("DEBUG: About to start rendering...")
	engine.Render(0, 0)
}

func (ws *WaterSimulation) Start() {
	fmt.Println("DEBUG: WaterSimulation Start() called")

	ws.engine.Camera.InvertMouse = false
	ws.engine.SetFrustumCulling(false)

	// Red background was affecting ALL examples - removed
	fmt.Println("DEBUG: Testing default shader with green plane")

	// Initialize camera - positioned to see the large optimized ocean surface
	// Ocean is 1000x1000 centered at (500, 0, 500)
	ws.engine.Camera.Position = mgl32.Vec3{500, 80, 650} // Higher up for bigger ocean overview
	ws.engine.Camera.LookAt(mgl32.Vec3{500, 0, 500})     // Look at ocean center
	ws.engine.Camera.Speed = 120
	fmt.Printf("DEBUG: Camera positioned for %fx%f ocean at %v, looking at %v\n", OceanSize, OceanSize, ws.engine.Camera.Position, ws.engine.Camera.LookAt)

	// Initialize light for bigger ocean
	ws.engine.Light = renderer.CreateLight()
	ws.engine.Light.Type = renderer.STATIC_LIGHT
	ws.engine.Light.Position = mgl32.Vec3{600, 500, 600} // Higher and further for much bigger ocean
	fmt.Printf("DEBUG: Light positioned at %v\n", ws.engine.Light.Position)

	// Load the optimized water surface model - much more efficient than regular plane
	model, err := loader.LoadWaterSurface(OceanSize, OceanSize/2, OceanSize/2, WaterResolution)
	if err != nil {
		panic("Failed to load water surface: " + err.Error())
	}
	fmt.Printf("DEBUG: Optimized water surface loaded with OceanSize=%f, Resolution=%d\n", OceanSize, WaterResolution)

	model.SetDiffuseColor(0.0, 0.4, 0.8) // Blue water color
	model.Shader = ws.shader             // RE-ENABLED - test custom shader with fixed renderer

	// Debug shader assignment
	fmt.Printf("DEBUG: Water shader IsValid(): %v\n", ws.shader.IsValid())
	fmt.Println("DEBUG: Using CUSTOM WATER SHADER with fixed shader switching bug")

	// Force water shader compilation by calling Use() once
	fmt.Println("DEBUG: Forcing simplified water shader compilation...")
	ws.shader.Use() // Force compilation here

	ws.model = model
	ws.engine.AddModel(model)
	fmt.Println("DEBUG: Model added to engine")

	fmt.Println("DEBUG: Ocean surface initialization complete - GPU waves active")
}

func (ws *WaterSimulation) Update() {
	// Update time for wave animation
	ws.currentTime = float32(time.Since(ws.startTime).Seconds())

	// Update water uniforms for the real water shader
	ws.UpdateWaterUniforms() // RE-ENABLED for real water shader with waves

	// Debug every 10 seconds to reduce spam
	if int(ws.currentTime)%10 == 0 && int(ws.currentTime*10)%10 == 0 {
		fmt.Printf("DEBUG: Ocean Update - Time: %.1fs, GPU-based waves active\n", ws.currentTime)
	}
}

func (ws *WaterSimulation) UpdateFixed() {}

// UpdateWaterUniforms updates the model's custom uniforms for the water shader
func (ws *WaterSimulation) UpdateWaterUniforms() {
	if ws.model.CustomUniforms == nil {
		ws.model.CustomUniforms = make(map[string]interface{})
	}

	ws.model.CustomUniforms["time"] = ws.currentTime
	ws.model.CustomUniforms["waveCount"] = int32(ws.waveCount)

	for i := 0; i < ws.waveCount && i < 5; i++ { // Limit to shader array size
		ws.model.CustomUniforms[fmt.Sprintf("waveDirections[%d]", i)] = ws.waveDirections[i]
		ws.model.CustomUniforms[fmt.Sprintf("waveAmplitudes[%d]", i)] = ws.waveAmplitudes[i]
		ws.model.CustomUniforms[fmt.Sprintf("waveFrequencies[%d]", i)] = ws.waveFrequencies[i]
		ws.model.CustomUniforms[fmt.Sprintf("waveSpeeds[%d]", i)] = ws.waveSpeeds[i]
	}
}
