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

	"math/rand"

	"github.com/go-gl/mathgl/mgl32"
)

const (
	OceanSize       = 30000 // Reasonable ocean size for testing centering
	WaterResolution = 2048  // Water mesh resolution (higher = smoother, lower = faster)
	WaveSpeed       = 0.8   // Much slower for realistic ocean movement
	Amplitude       = 3     // Moderate waves for realistic ocean
	MaxWaves        = 8     // Reasonable number of waves
	WindSpeed       = 8.0   // Moderate wind speed
	WaveAge         = 1.4   // Wave development factor (1.0 = young, 2.0 = mature)
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
	wavePhases      []float32       // Wave phase offsets for variety
	waveSteepness   []float32       // Wave steepness for shape control
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
		wavePhases:      make([]float32, MaxWaves),
		waveSteepness:   make([]float32, MaxWaves),
	}

	// Initialize wave parameters with simpler, more visible setup for debugging
	for i := 0; i < MaxWaves; i++ {
		// Create moderate wave layers
		var baseAmplitude, baseFreq float32

		if i < 2 {
			// Primary ocean swells (moderate)
			baseAmplitude = Amplitude * (0.8 + rand.Float32()*0.4)
			baseFreq = 0.01 + rand.Float32()*0.005
		} else if i < 4 {
			// Medium waves
			baseAmplitude = Amplitude * 0.6 * (0.7 + rand.Float32()*0.3)
			baseFreq = 0.02 + rand.Float32()*0.01
		} else {
			// Small detail waves
			baseAmplitude = Amplitude * 0.3 * (0.6 + rand.Float32()*0.4)
			baseFreq = 0.04 + rand.Float32()*0.02
		}

		// Moderate directional spread
		baseAngle := float32(i) * 60.0 * math.Pi / 180.0                // 60-degree spread between waves
		randomOffset := (rand.Float32() - 0.5) * 40.0 * math.Pi / 180.0 // ±20 degree random offset
		waveAngle := baseAngle + randomOffset

		dirX := float32(math.Cos(float64(waveAngle)))
		dirZ := float32(math.Sin(float64(waveAngle)))
		ws.waveDirections[i] = mgl32.Vec3{dirX, 0.0, dirZ}.Normalize()

		ws.waveAmplitudes[i] = baseAmplitude
		ws.waveFrequencies[i] = baseFreq

		// Much more reasonable wave speeds
		ws.waveSpeeds[i] = WaveSpeed * (0.8 + rand.Float32()*0.4)

		// Random phase offsets for natural variation
		ws.wavePhases[i] = rand.Float32() * 2.0 * math.Pi

		// Moderate steepness
		ws.waveSteepness[i] = 0.3 + rand.Float32()*0.3

		fmt.Printf("DEBUG: Wave %d - Dir: [%.2f, %.2f, %.2f], Amp: %.3f, Freq: %.5f, Speed: %.2f\n",
			i, ws.waveDirections[i].X(), ws.waveDirections[i].Y(), ws.waveDirections[i].Z(),
			ws.waveAmplitudes[i], ws.waveFrequencies[i], ws.waveSpeeds[i])
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

	// Initialize camera - positioned to look at actual center of water plane
	oceanCenter := float32(OceanSize / 2)                                     // This matches the center used in LoadWaterSurface
	ws.engine.Camera.Position = mgl32.Vec3{oceanCenter, 20, oceanCenter + 50} // Above the actual center
	ws.engine.Camera.LookAt(mgl32.Vec3{oceanCenter, 0, oceanCenter})          // Look at the actual center
	ws.engine.Camera.Speed = 60                                               // Reasonable speed for exploration
	fmt.Printf("DEBUG: Camera positioned at %v, looking at ocean center (%.0f,0,%.0f)\n", ws.engine.Camera.Position, oceanCenter, oceanCenter)

	// Initialize sun-like directional light
	ws.engine.Light = renderer.CreateLight()
	ws.engine.Light.Type = renderer.STATIC_LIGHT
	ws.engine.Light.Mode = "directional"                                             // Set as directional light
	ws.engine.Light.Position = mgl32.Vec3{oceanCenter + 500, 800, oceanCenter + 500} // Far away and high like the sun
	ws.engine.Light.Color = mgl32.Vec3{1.0, 0.95, 0.8}                               // Warm sunlight color
	ws.engine.Light.Intensity = 1.2                                                  // Bright like the sun
	fmt.Printf("DEBUG: Directional sun light positioned at %v\n", ws.engine.Light.Position)

	// Load the optimized water surface model - much more efficient than regular plane
	model, err := loader.LoadWaterSurface(OceanSize, oceanCenter, oceanCenter, WaterResolution)
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

	// Pass all wave parameters to the shader
	ws.model.CustomUniforms["waveCount"] = int32(ws.waveCount)
	ws.model.CustomUniforms["time"] = ws.currentTime

	// Convert wave data to arrays for OpenGL uniforms
	directions := make([]float32, MaxWaves*3) // Vec3 = 3 floats each
	amplitudes := make([]float32, MaxWaves)
	frequencies := make([]float32, MaxWaves)
	speeds := make([]float32, MaxWaves)
	phases := make([]float32, MaxWaves)
	steepness := make([]float32, MaxWaves)

	for i := 0; i < MaxWaves; i++ {
		// Wave directions (Vec3)
		directions[i*3] = ws.waveDirections[i].X()
		directions[i*3+1] = ws.waveDirections[i].Y()
		directions[i*3+2] = ws.waveDirections[i].Z()

		// Wave parameters
		amplitudes[i] = ws.waveAmplitudes[i]
		frequencies[i] = ws.waveFrequencies[i]
		speeds[i] = ws.waveSpeeds[i]
		phases[i] = ws.wavePhases[i]
		steepness[i] = ws.waveSteepness[i]
	}

	// Pass arrays to shader
	ws.model.CustomUniforms["waveDirections"] = directions
	ws.model.CustomUniforms["waveAmplitudes"] = amplitudes
	ws.model.CustomUniforms["waveFrequencies"] = frequencies
	ws.model.CustomUniforms["waveSpeeds"] = speeds
	ws.model.CustomUniforms["wavePhases"] = phases
	ws.model.CustomUniforms["waveSteepness"] = steepness

}
