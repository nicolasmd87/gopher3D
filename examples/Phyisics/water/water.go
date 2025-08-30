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
	OceanSize       = 900000 // Massive photorealistic ocean - 900km
	WaterResolution = 2048   // Higher resolution for massive scale
	WaveSpeed       = 0.6    // Slower, more realistic wave speed for large scale
	MaxWaves        = 4      // Match shader expectation (4 waves)
	WindSpeed       = 7.0    // Natural wind speed
	WaveAge         = 1.3    // Natural wave development
)

// Configurable wave parameters - modify these to change wave behavior
var (
	Amplitude = float32(500.0)
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

		// GPU Gems: Large geometric waves with realistic physics
		if i == 0 {
			baseAmplitude = Amplitude * 1.2 // Primary ocean swell
			baseFreq = 0.00008              // Very long wavelength (~12.5km)
		} else if i == 1 {
			baseAmplitude = Amplitude * 0.8 // Secondary swell
			baseFreq = 0.00015              // Long wavelength (~6.7km)
		} else if i == 2 {
			baseAmplitude = Amplitude * 0.6 // Wind waves
			baseFreq = 0.0004               // Medium wavelength (~2.5km)
		} else {
			baseAmplitude = Amplitude * 0.4 // Small wind waves
			baseFreq = 0.0008               // Shorter wavelength (~1.25km)
		}

		// GPU Gems: Wave directions with realistic 45° separation
		baseAngle := float32(i) * 45.0 * math.Pi / 180.0 // 45-degree spread for interference
		dirX := float32(math.Cos(float64(baseAngle)))
		dirZ := float32(math.Sin(float64(baseAngle)))
		ws.waveDirections[i] = mgl32.Vec3{dirX, 0.0, dirZ}.Normalize()

		ws.waveAmplitudes[i] = baseAmplitude
		ws.waveFrequencies[i] = baseFreq

		// GPU Gems: Physics-based wave speed (deep water: speed ∝ sqrt(wavelength))
		wavelength := 2.0 * math.Pi / float64(baseFreq)
		physicalSpeed := float32(0.002 * math.Sqrt(wavelength)) // Realistic physics
		ws.waveSpeeds[i] = physicalSpeed

		ws.wavePhases[i] = float32(i) * math.Pi / 3.0 // 60° phase offset

		ws.waveSteepness[i] = 0.2 + float32(i)*0.1 // Gentle progressive steepness

		fmt.Printf("DEBUG: GPU Gems Wave %d - Dir: [%.2f, %.2f, %.2f], Amp: %.3f, Freq: %.5f, Speed: %.2f, λ=%.0fm\n",
			i, ws.waveDirections[i].X(), ws.waveDirections[i].Y(), ws.waveDirections[i].Z(),
			ws.waveAmplitudes[i], ws.waveFrequencies[i], ws.waveSpeeds[i], wavelength)
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
	// Create SUN as a directional light (not point light!)
	sunDirection := mgl32.Vec3{0.3, -1.0, 0.2}.Normalize()                                           // Sun coming from above at slight angle
	ws.engine.Light = renderer.CreateDirectionalLight(sunDirection, mgl32.Vec3{1.0, 0.98, 0.9}, 4.5) // Much brighter directional sun light
	ws.engine.Light.AmbientStrength = 0.25                                                           // Higher ambient for natural ocean lighting
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

	// GPU Gems Chapter 9 & 11: Shadow settings will be applied via CustomUniforms
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
		// Position sun much higher for 900km ocean scale - visible from anywhere
		sunModel.SetPosition(oceanCenter, 200000.0, oceanCenter) // Much higher sun position for massive scale

		ws.sunModel = sunModel

		ws.engine.AddModel(sunModel)
	}

	// Add underwater objects for depth perception
	ws.addUnderwaterObjects()

	ws.startTime = time.Now()
	ws.currentTime = 0.0
	ws.SetFixedDaylight()
}

func (ws *WaterSimulation) Update() {
	ws.currentTime = float32(time.Since(ws.startTime).Seconds())

	ws.updateDynamicWaterUniforms()

}

func (ws *WaterSimulation) UpdateFixed() {}

// addUnderwaterObjects adds objects beneath the water surface to show depth
func (ws *WaterSimulation) addUnderwaterObjects() {
	oceanCenter := float32(OceanSize / 2)

	// Add several underwater cubes at different depths and positions
	underwaterPositions := []mgl32.Vec3{
		{oceanCenter - 15000, -25.0, oceanCenter - 10000}, // Shallow cube
		{oceanCenter + 20000, -45.0, oceanCenter + 15000}, // Medium depth cube
		{oceanCenter - 25000, -65.0, oceanCenter + 20000}, // Deep cube
		{oceanCenter + 10000, -35.0, oceanCenter - 18000}, // Another medium cube
		{oceanCenter, -55.0, oceanCenter + 5000},          // Central deep cube
	}

	for i, pos := range underwaterPositions {
		cube, err := loader.LoadObjectWithPath("../../resources/obj/Cube.obj", true)
		if err != nil {
			fmt.Printf("Warning: Could not load underwater cube %d: %v\n", i, err)
			continue
		}

		// Scale based on depth (deeper objects appear smaller due to perspective)
		depth := -pos.Y()
		scale := mgl32.Vec3{800 + depth*5, 800 + depth*5, 800 + depth*5} // Larger cubes, scaled by depth
		cube.Scale = scale
		cube.SetPosition(pos.X(), pos.Y(), pos.Z())

		// Set underwater coloration (blue-green tint, less red due to water absorption)
		redChannel := float32(0.2 - depth*0.003)   // Red disappears with depth
		greenChannel := float32(0.4 - depth*0.002) // Green reduces slower
		blueChannel := float32(0.6)                // Blue remains strong underwater

		cube.SetDiffuseColor(redChannel, greenChannel, blueChannel)
		cube.SetMaterialPBR(0.1, 0.8) // Slightly metallic, rough surface for underwater look
		cube.SetExposure(1.0)

		// GPU Gems: Enable shadows for underwater objects via CustomUniforms
		if cube.CustomUniforms == nil {
			cube.CustomUniforms = make(map[string]interface{})
		}
		cube.CustomUniforms["enableShadows"] = true
		cube.CustomUniforms["shadowIntensity"] = float32(0.4) // Slightly darker shadows underwater
		cube.CustomUniforms["shadowSoftness"] = float32(0.2)

		ws.engine.AddModel(cube)
		fmt.Printf("Added underwater cube %d at depth %.1f with scale %.0f\n", i, depth, scale.X())
	}
}

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
	// Caustics configuration - easily adjustable
	ws.model.CustomUniforms["enableCaustics"] = false                  // Set to true to enable caustics
	ws.model.CustomUniforms["causticsIntensity"] = float32(0.3)        // Caustics strength
	ws.model.CustomUniforms["causticsScale"] = float32(0.003)          // Caustics pattern scale
	ws.model.CustomUniforms["waterPlaneHeight"] = float32(5.0)         // Water surface height
	ws.model.CustomUniforms["causticsSpeed"] = mgl32.Vec2{0.02, 0.015} // Animation speed

	// GPU Gems Chapter 9 & 11: Shadow configuration for water
	ws.model.CustomUniforms["enableShadows"] = true
	ws.model.CustomUniforms["shadowIntensity"] = float32(0.3) // 30% shadow darkness
	ws.model.CustomUniforms["shadowSoftness"] = float32(0.2)  // Chapter 11: Soft shadow edges

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

	// Don't override caustics settings every frame - they're set once in setupWaterUniforms
	// Only update time-based uniforms here
	ws.model.CustomUniforms["skyColor"] = ws.lastSkyColor
	ws.model.CustomUniforms["horizonColor"] = mgl32.Vec3{
		ws.lastSkyColor.X() * 0.85,
		ws.lastSkyColor.Y() * 0.85,
		ws.lastSkyColor.Z() * 0.85,
	}

	// Apply caustics uniforms to all models in the scene (for underwater objects)
	// Note: This is a simplified approach for the water example.
	// In a production engine, you'd want a cleaner API for global uniforms.
	// For now, we'll just apply to our known underwater objects via their individual uniforms.
}
