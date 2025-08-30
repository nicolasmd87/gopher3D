package main

import (
	behaviour "Gopher3D/internal/behaviour"
	"Gopher3D/internal/engine"
	loader "Gopher3D/internal/loader"
	"Gopher3D/internal/renderer"
	"fmt"
	"math"
	"time"

	"github.com/go-gl/mathgl/mgl32"
)

const (
	OceanSize        = 900000 // Massive photorealistic ocean - 100x bigger
	WaterResolution  = 4096   // Higher resolution for massive scale
	WaveSpeed        = 0.6    // Slower, more realistic wave speed for large scale
	Amplitude        = 8.0    // Larger waves for massive ocean scale
	MaxWaves         = 4      // Match shader expectation (4 waves)
	WindSpeed        = 7.0    // Natural wind speed
	WaveAge          = 1.3    // Natural wave development
	DayCycleDuration = 300.0  // 5 minutes for full day cycle
)

type WaterSimulation struct {
	model           *renderer.Model // Water surface model
	sunModel        *renderer.Model // Visual sun sphere
	engine          *engine.Gopher  // Engine instance
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
	sunAngle        float32         // Current sun angle for moving sun
	lastSkyColor    mgl32.Vec3      // Track last sky color to avoid unnecessary updates
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
		sunAngle:        0.0,
	}

	// Initialize wave parameters for 4 waves
	for i := 0; i < MaxWaves; i++ {
		var baseAmplitude, baseFreq float32

		if i == 0 {
			baseAmplitude = Amplitude * 1.0
			baseFreq = 0.0001 // Much lower frequency for massive ocean
		} else if i == 1 {
			baseAmplitude = Amplitude * 0.7
			baseFreq = 0.0002
		} else if i == 2 {
			baseAmplitude = Amplitude * 0.5
			baseFreq = 0.0005
		} else {
			baseAmplitude = Amplitude * 0.3
			baseFreq = 0.001
		}

		baseAngle := float32(i) * 90.0 * math.Pi / 180.0 // 90-degree spread between waves
		dirX := float32(math.Cos(float64(baseAngle)))
		dirZ := float32(math.Sin(float64(baseAngle)))
		ws.waveDirections[i] = mgl32.Vec3{dirX, 0.0, dirZ}.Normalize()

		ws.waveAmplitudes[i] = baseAmplitude
		ws.waveFrequencies[i] = baseFreq

		ws.waveSpeeds[i] = WaveSpeed * (0.8 + float32(i)*0.1)

		ws.wavePhases[i] = float32(i) * math.Pi / 2.0

		ws.waveSteepness[i] = 0.5 + float32(i)*0.15

		fmt.Printf("DEBUG: Wave %d - Dir: [%.2f, %.2f, %.2f], Amp: %.3f, Freq: %.5f, Speed: %.2f\n",
			i, ws.waveDirections[i].X(), ws.waveDirections[i].Y(), ws.waveDirections[i].Z(),
			ws.waveAmplitudes[i], ws.waveFrequencies[i], ws.waveSpeeds[i])
	}

	ws.sunModel = nil

	behaviour.GlobalBehaviourManager.Add(ws)
}

func main() {

	engine := engine.NewGopher(engine.OPENGL)
	engine.SetDebugMode(false) // Turn off wireframe for clearer view
	NewWaterSimulation(engine)
	engine.Width = 1920
	engine.Height = 1080
	engine.Render(0, 0) // Proper window position
}

func (ws *WaterSimulation) Start() {

	ws.engine.Camera.InvertMouse = false

	oceanCenter := float32(OceanSize / 2)                                           // This matches the center used in LoadWaterSurface
	ws.engine.Camera.Position = mgl32.Vec3{oceanCenter, 20000, oceanCenter + 50000} // Much higher position for 900km ocean

	// Configure camera projection for massive ocean scale - engine handles projection updates automatically
	ws.engine.Camera.SetNear(10.0)     // Larger near plane for massive scale
	ws.engine.Camera.SetFar(2000000.0) // Much farther for 900km ocean
	ws.engine.Camera.Speed = 15000     // Much faster speed for exploring the massive ocean

	oceanCenter = float32(OceanSize / 2)
	sunPosition := mgl32.Vec3{oceanCenter, 100000.0, oceanCenter}                                        // Much higher sun position
	ws.engine.Light = renderer.CreatePointLight(sunPosition, mgl32.Vec3{1.0, 0.98, 0.9}, 3.0, 1000000.0) // Brighter light with much larger range for massive scale
	ws.engine.Light.AmbientStrength = 0.08                                                               // Much lower ambient for photorealistic contrast
	ws.engine.Light.Type = renderer.STATIC_LIGHT

	// Skybox - follow the same API used in other examples
	ws.lastSkyColor = mgl32.Vec3{0.5, 0.7, 1.0}
	renderer.SetSkyboxColor(ws.lastSkyColor.X(), ws.lastSkyColor.Y(), ws.lastSkyColor.Z())
	if err := ws.engine.SetSkybox("dark_sky"); err != nil {
		fmt.Printf("Could not set skybox: %v\n", err)
	}

	// Load the optimized water surface model - much more efficient than regular plane
	model, err := loader.LoadWaterSurface(OceanSize, oceanCenter, oceanCenter, WaterResolution)
	if err != nil {
		panic("Failed to load water surface: " + err.Error())
	}

	// Enhanced water material for realistic appearance
	model.SetDiffuseColor(0.05, 0.25, 0.55) // Deeper, more realistic ocean blue
	model.SetMaterialPBR(0.02, 0.1)         // Slightly metallic with low roughness for realistic water
	model.SetExposure(1.2)                  // Slightly enhanced exposure for better light reflection
	model.Shader = ws.shader                // Apply custom water shader to water surface

	ws.model = model

	ws.setupWaterUniforms()

	ws.engine.AddModel(model)
	sunModel, err := loader.LoadObjectWithPath("../../resources/obj/Sphere.obj", true)
	if err != nil {
		fmt.Printf("ERROR: Failed to load sun sphere: %v\n", err)
	} else {
		sunModel.Scale = mgl32.Vec3{10000, 10000, 10000} // Massive sun - 20x bigger for visibility at 900km ocean scale
		sunModel.SetDiffuseColor(1.0, 0.95, 0.8)         // Bright natural sun color
		sunModel.SetMaterialPBR(0.0, 0.0)                // Non-metallic, mirror smooth for maximum brightness
		sunModel.SetExposure(30.0)                       // Even higher exposure for maximum brightness

		oceanCenter := float32(OceanSize / 2)
		sunModel.SetPosition(oceanCenter, 15000.0, oceanCenter) // Closer to camera for better visibility

		ws.sunModel = sunModel

		ws.engine.AddModel(sunModel)
	}

	ws.startTime = time.Now()
	ws.currentTime = 0.0
	ws.SetFixedDaylight()
}

func (ws *WaterSimulation) Update() {
	ws.currentTime = float32(time.Since(ws.startTime).Seconds())

	ws.updateDynamicWaterUniforms()

}

func (ws *WaterSimulation) UpdateFixed() {}

// SetFixedDaylight sets up a fixed bright daylight scene for water reflection
func (ws *WaterSimulation) SetFixedDaylight() {
	// Set photorealistic daylight colors and intensity
	lightColor := mgl32.Vec3{1.0, 0.98, 0.95} // Bright white sunlight
	lightIntensity := float32(2.2)            // Dimmer overall lighting for photorealism

	ws.engine.Light.Color = lightColor
	ws.engine.Light.Intensity = lightIntensity

	// Set photorealistic daylight sky color
	sky := mgl32.Vec3{0.4, 0.6, 0.9} // More natural, slightly dimmer sky for photorealism
	ws.lastSkyColor = sky
	ws.engine.UpdateSkyboxColor(sky.X(), sky.Y(), sky.Z())

	// Ensure sun sphere stays bright
	if ws.sunModel != nil {
		ws.sunModel.SetDiffuseColor(1.0, 0.9, 0.6) // Bright yellow sun
		fmt.Printf("SUN: Fixed daylight - Pos=(%.0f, %.0f, %.0f), Light Intensity=%.2f\n",
			ws.sunModel.Position.X(), ws.sunModel.Position.Y(), ws.sunModel.Position.Z(),
			ws.engine.Light.Intensity)
	}

	fmt.Println("DEBUG: Fixed daylight scene configured - bright sun for water reflections")
}

// setupWaterUniforms sets up static water uniforms once (handled automatically by engine)
func (ws *WaterSimulation) setupWaterUniforms() {
	if ws.model.CustomUniforms == nil {
		ws.model.CustomUniforms = make(map[string]interface{})
	}

	// Set static wave parameters (these don't change)
	ws.model.CustomUniforms["waveCount"] = int32(ws.waveCount)

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

	// Pass arrays to shader (static)
	ws.model.CustomUniforms["waveDirections"] = directions
	ws.model.CustomUniforms["waveAmplitudes"] = amplitudes
	ws.model.CustomUniforms["waveFrequencies"] = frequencies
	ws.model.CustomUniforms["waveSpeeds"] = speeds
	ws.model.CustomUniforms["wavePhases"] = phases
	ws.model.CustomUniforms["waveSteepness"] = steepness

	// Configure water with clean API - minimal fog
	waterConfig := renderer.WaterConfig{
		EnableFog:    true,                      // Enable fog with minimal intensity
		FogStart:     20.0,                      // Start fog much closer for gradual transition
		FogEnd:       800.0,                     // End fog further for smoother transition
		FogIntensity: 0.05,                      // Minimal fog intensity to prevent any sky influence
		FogColor:     mgl32.Vec3{0.4, 0.5, 0.6}, // Very neutral fog color
		SkyColor:     ws.lastSkyColor,
		HorizonColor: mgl32.Vec3{
			ws.lastSkyColor.X() * 0.85,
			ws.lastSkyColor.Y() * 0.85,
			ws.lastSkyColor.Z() * 0.85,
		},
	}
	renderer.ApplyWaterConfig(ws.model, waterConfig)

	fmt.Println("DEBUG: Static water uniforms set up (handled automatically by engine)")
}

// updateDynamicWaterUniforms updates only the time-based uniforms
func (ws *WaterSimulation) updateDynamicWaterUniforms() {
	// Update time, light position (for point light) and sky colors dynamically
	ws.model.CustomUniforms["time"] = ws.currentTime
	ws.model.CustomUniforms["lightPos"] = ws.engine.Light.Position        // Point light position
	ws.model.CustomUniforms["lightColor"] = ws.engine.Light.Color         // Light color
	ws.model.CustomUniforms["lightIntensity"] = ws.engine.Light.Intensity // Light intensity
	ws.model.CustomUniforms["skyColor"] = ws.lastSkyColor
	ws.model.CustomUniforms["horizonColor"] = mgl32.Vec3{
		ws.lastSkyColor.X() * 0.85,
		ws.lastSkyColor.Y() * 0.85,
		ws.lastSkyColor.Z() * 0.85,
	}
}
