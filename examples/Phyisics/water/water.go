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
	OceanSize       = 90000 // Massive photorealistic ocean - 900km
	WaterResolution = 4096  // Higher resolution for massive scale
	WaveSpeed       = 0.6   // Slower, more realistic wave speed for large scale
	MaxWaves        = 4     // Match shader expectation (4 waves)
	WindSpeed       = 7.0   // Natural wind speed
	WaveAge         = 1.3   // Natural wave development
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

	}

	//ws.sunModel = nil

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
	// Disable frustum culling to ensure sun is always rendered
	ws.engine.SetFrustumCulling(false)

	ws.engine.Camera.InvertMouse = false

	oceanCenter := float32(OceanSize / 2)                                           // This matches the center used in LoadWaterSurface
	ws.engine.Camera.Position = mgl32.Vec3{oceanCenter, 15000, oceanCenter + 40000} // Lower position for natural horizon view

	// Configure camera projection for massive ocean scale - engine handles projection updates automatically
	ws.engine.Camera.SetNear(10.0)     // Larger near plane for massive scale
	ws.engine.Camera.SetFar(2000000.0) // Much farther for 900km ocean
	ws.engine.Camera.Speed = 15000     // Much faster speed for exploring the massive ocean

	oceanCenter = float32(OceanSize / 2)
	// Calculate sun direction based on actual sun position for consistency
	cameraPos := mgl32.Vec3{oceanCenter, 20000, oceanCenter + 50000}
	sunPos := mgl32.Vec3{oceanCenter + 200000, 100000.0, oceanCenter + 200000} // Match new distant sun position
	// For directional light, we need direction that light rays travel (from sun to objects)
	sunDirection := cameraPos.Sub(sunPos).Normalize() // Direction from sun toward camera (light ray direction)

	ws.engine.Light = renderer.CreateDirectionalLight(sunDirection, mgl32.Vec3{1.0, 0.98, 0.9}, 4.5) // Use direction as-is
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

	// DEBUG: Check if water model was loaded properly
	fmt.Printf("DEBUG WATER LOADED: Vertices=%d, Faces=%d, InterleavedData=%d\n",
		len(model.Vertices), len(model.Faces), len(model.InterleavedData))

	// Enhanced water material for realistic appearance - natural ocean water
	model.SetDiffuseColor(0.06, 0.22, 0.45) // Natural ocean blue matching shader
	model.SetMaterialPBR(0.0, 0.4)          // Non-metallic (0.0) with higher roughness for matte appearance
	model.SetExposure(1.0)                  // Standard exposure, brightness controlled by lighting

	model.Shader = ws.shader // Apply custom water shader to water surface

	// GPU Gems Chapter 9 & 11: Shadow settings will be applied via CustomUniforms
	ws.model = model
	ws.setupWaterUniforms()
	ws.engine.AddModel(model)

	// Load sun model following the same pattern as other examples
	sunModel, err := loader.LoadObjectWithPath("../../resources/obj/Sphere.obj", true)
	if err != nil {
		fmt.Printf("ERROR: Failed to load sphere sun: %v\n", err)
		return
	}

	// DEBUG: Check if sun model was loaded properly
	fmt.Printf("DEBUG SUN LOADED: Vertices=%d, Faces=%d, InterleavedData=%d\n",
		len(sunModel.Vertices), len(sunModel.Faces), len(sunModel.InterleavedData))

	// Configure sun model properties - Very large and bright sun
	sunModel.Scale = mgl32.Vec3{50000, 50000, 50000} // Large 50km diameter sun
	sunModel.SetDiffuseColor(1.0, 0.95, 0.8)         // Warm sun color
	sunModel.SetMaterialPBR(0.0, 0.0)                // Non-metallic, mirror smooth for maximum brightness
	sunModel.SetExposure(150.0)                      // Very high exposure to trigger emissive mode

	oceanCenter = float32(OceanSize / 2)
	// Position sun high in the sky where it should be visible
	sunModel.SetPosition(oceanCenter+150000, 200000.0, oceanCenter+50000) // High in sky, visible above horizon

	// Ensure sun uses default shader for emissive properties
	ws.sunModel = sunModel
	ws.engine.AddModel(sunModel)

	// DEBUG: Check if VAO was created after AddModel
	fmt.Printf("DEBUG SUN AFTER AddModel: VAO=%d, VBO=%d, EBO=%d\n",
		sunModel.VAO, sunModel.VBO, sunModel.EBO)

	ws.startTime = time.Now()
	ws.currentTime = 0.0
	ws.SetFixedDaylight() // Re-enabled to ensure proper sun visibility
}

func (ws *WaterSimulation) Update() {
	ws.currentTime = float32(time.Since(ws.startTime).Seconds())

	ws.updateDynamicWaterUniforms()

}

func (ws *WaterSimulation) UpdateFixed() {}

// SetFixedDaylight sets up a fixed bright daylight scene for water reflection
func (ws *WaterSimulation) SetFixedDaylight() {
	// Set natural daylight colors and intensity
	lightColor := mgl32.Vec3{1.0, 0.98, 0.95} // Bright white sunlight
	lightIntensity := float32(2.8)            // Natural intensity for realistic water

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
	// Apply photorealistic Advanced Rendering configuration for professional water
	waterRenderConfig := renderer.WaterPhotorealisticConfig()
	// Professional settings for smooth, natural appearance
	waterRenderConfig.MeshSmoothingIntensity = 0.85 // High smoothing without artifacts
	waterRenderConfig.FilteringQuality = 3          // High quality filtering
	waterRenderConfig.AntiAliasing = true
	waterRenderConfig.NormalSmoothingRadius = 1.2 // Natural smoothing radius
	waterRenderConfig.EnableCaustics = false      // Disable for clean appearance
	waterRenderConfig.NoiseIntensity = 0.0        // No surface noise for uniform color
	renderer.ApplyAdvancedRenderingConfig(ws.model, waterRenderConfig)

	// Water-specific uniforms
	ws.model.CustomUniforms["waterPlaneHeight"] = float32(5.0) // Water surface height

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
