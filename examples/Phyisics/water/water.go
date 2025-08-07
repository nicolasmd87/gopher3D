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
	OceanSize        = 5000  // Much bigger ocean for realistic scale
	WaterResolution  = 1024  // Higher resolution for smoother water
	WaveSpeed        = 0.6   // Natural wave speed
	Amplitude        = 2.5   // Natural wave height
	MaxWaves         = 4     // Match shader expectation (4 waves)
	WindSpeed        = 7.0   // Natural wind speed
	WaveAge          = 1.3   // Natural wave development
	DayCycleDuration = 300.0 // 5 minutes for full day cycle
)

type WaterSimulation struct {
	model           *renderer.Model // Water surface model
	sunModel        *renderer.Model // Visual sun sphere
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
		sunAngle:        0.0, // Start sun at dawn
	}

	// Initialize wave parameters for 4 waves
	for i := 0; i < MaxWaves; i++ {
		// Create different wave layers
		var baseAmplitude, baseFreq float32

		if i == 0 {
			// Primary ocean swell
			baseAmplitude = Amplitude * 1.0
			baseFreq = 0.01
		} else if i == 1 {
			// Secondary swell
			baseAmplitude = Amplitude * 0.7
			baseFreq = 0.015
		} else if i == 2 {
			// Medium waves
			baseAmplitude = Amplitude * 0.5
			baseFreq = 0.025
		} else {
			// Small detail waves
			baseAmplitude = Amplitude * 0.3
			baseFreq = 0.04
		}

		// Simple directional spread for 4 waves
		baseAngle := float32(i) * 90.0 * math.Pi / 180.0 // 90-degree spread between waves
		dirX := float32(math.Cos(float64(baseAngle)))
		dirZ := float32(math.Sin(float64(baseAngle)))
		ws.waveDirections[i] = mgl32.Vec3{dirX, 0.0, dirZ}.Normalize()

		ws.waveAmplitudes[i] = baseAmplitude
		ws.waveFrequencies[i] = baseFreq

		// Simple wave speeds
		ws.waveSpeeds[i] = WaveSpeed * (0.8 + float32(i)*0.1)

		// Simple phase offsets
		ws.wavePhases[i] = float32(i) * math.Pi / 2.0

		// Simple steepness
		ws.waveSteepness[i] = 0.3 + float32(i)*0.1

		fmt.Printf("DEBUG: Wave %d - Dir: [%.2f, %.2f, %.2f], Amp: %.3f, Freq: %.5f, Speed: %.2f\n",
			i, ws.waveDirections[i].X(), ws.waveDirections[i].Y(), ws.waveDirections[i].Z(),
			ws.waveAmplitudes[i], ws.waveFrequencies[i], ws.waveSpeeds[i])
	}

	// Store references to models to create later when OpenGL context is ready
	ws.sunModel = nil // Will be created in Start() method

	behaviour.GlobalBehaviourManager.Add(ws)
}

func main() {
	fmt.Println("DEBUG: Starting water example")

	engine := engine.NewGopher(engine.OPENGL)
	engine.SetDebugMode(false) // Turn off wireframe for clearer view
	NewWaterSimulation(engine)

	fmt.Println("DEBUG: About to start rendering...")
	engine.Render(0, 0)
}

func (ws *WaterSimulation) Start() {
	fmt.Println("DEBUG: WaterSimulation Start() called")

	ws.engine.Camera.InvertMouse = false
	ws.engine.SetFrustumCulling(false) // Disable frustum culling to ensure sun sphere is visible
	ws.engine.SetFaceCulling(false)    // Disable face culling for debugging

	// Red background was affecting ALL examples - removed
	fmt.Println("DEBUG: Testing default shader with green plane")

	// Initialize camera - positioned to see the massive ocean and sun
	oceanCenter := float32(OceanSize / 2) // This matches the center used in LoadWaterSurface
	ws.engine.Camera.Speed = 300          // Faster speed for exploring the massive ocean

	fmt.Printf("DEBUG: Camera positioned at %v, looking at ocean center (%.0f,0,%.0f)\n", ws.engine.Camera.Position, oceanCenter, oceanCenter)

	// Bright ocean lighting for visibility
	ws.engine.Light = renderer.CreateLight() // Basic light for simplicity
	ws.engine.Light.Mode = "directional"
	ws.engine.Light.Direction = mgl32.Vec3{0.2, -0.7, 0.3}.Normalize() // Natural sun angle
	ws.engine.Light.Color = mgl32.Vec3{1.0, 0.98, 0.95}                // Natural white sunlight
	ws.engine.Light.Intensity = 3.0                                    // Brighter lighting
	ws.engine.Light.AmbientStrength = 0.4                              // Balanced ambient for realistic scene
	ws.engine.Light.Type = renderer.STATIC_LIGHT
	fmt.Printf("DEBUG: Directional sun light positioned at %v\n", ws.engine.Light.Position)

	// Bright skybox - plain day
	renderer.SetSkyboxColor(0.8, 0.9, 1.0) // Bright day sky blue

	// Load the optimized water surface model - much more efficient than regular plane
	model, err := loader.LoadWaterSurface(OceanSize, oceanCenter, oceanCenter, WaterResolution)
	if err != nil {
		panic("Failed to load water surface: " + err.Error())
	}
	fmt.Printf("DEBUG: Optimized water surface loaded with OceanSize=%f, Resolution=%d\n", OceanSize, WaterResolution)

	// Natural water material for realistic appearance
	model.SetDiffuseColor(0.15, 0.35, 0.7) // Natural ocean blue
	// No exposure setting - keep it simple
	model.Shader = ws.shader // Apply custom water shader to water surface

	// Debug shader assignment
	fmt.Printf("DEBUG: Water shader IsValid(): %v\n", ws.shader.IsValid())
	fmt.Println("DEBUG: Using CUSTOM WATER SHADER")

	ws.model = model

	// Set up static water uniforms once (handled automatically by engine)
	ws.setupWaterUniforms()

	ws.engine.AddModel(model)
	fmt.Println("DEBUG: Model added to engine")

	// Create sun sphere
	fmt.Println("DEBUG: Creating sun sphere...")
	sunModel, err := loader.LoadObjectWithPath("../../resources/obj/Sphere_Low.obj", true) // Use simple sphere without material dependencies
	if err != nil {
		fmt.Printf("ERROR: Failed to load sun sphere: %v\n", err)
	} else {
		fmt.Printf("DEBUG: Sun sphere loaded successfully - Vertices: %d, Faces: %d\n",
			len(sunModel.InterleavedData)/8, len(sunModel.Faces))
		ws.sunModel = sunModel

		// Scale the sun to be reasonable size
		ws.sunModel.SetScale(200.0, 200.0, 200.0) // Reasonable sun sphere size

		// Set initial sun position - center of ocean, low
		oceanCenter := float32(OceanSize / 2)
		ws.sunModel.SetPosition(oceanCenter, 100.0, oceanCenter) // Center of ocean, low to horizon

		// Make sun EXTREMELY bright for visibility
		ws.sunModel.SetDiffuseColor(1.0, 1.0, 0.0) // BRIGHT YELLOW for maximum visibility

		// Ensure sun uses default shader
		ws.sunModel.Shader = renderer.GetDefaultShader()

		// Add sun sphere to engine
		ws.engine.AddModel(ws.sunModel)
		fmt.Printf("DEBUG: HUGE sun sphere added to scene - Scale: %v, Position: %v, Color: %v\n",
			ws.sunModel.Scale, ws.sunModel.Position, ws.sunModel.Material.DiffuseColor)
		fmt.Printf("DEBUG: Sun sphere model - Vertices: %d, Faces: %d, VAO: %d\n",
			len(ws.sunModel.InterleavedData)/8, len(ws.sunModel.Faces), ws.sunModel.VAO)
	}

	fmt.Println("DEBUG: Ocean surface initialization complete - GPU waves active")
}

func (ws *WaterSimulation) Update() {
	// Update time for wave animation
	ws.currentTime = float32(time.Since(ws.startTime).Seconds())

	// Update moving sun
	ws.UpdateMovingSun()

	// Only update dynamic water uniforms (time and light direction)
	ws.updateDynamicWaterUniforms()

	// TODO: Add space key to look at sun (need to implement key input)

	// Debug every 30 seconds
	if int(ws.currentTime)%30 == 0 && int(ws.currentTime*10)%10 == 0 {
		fmt.Printf("DEBUG: Time: %.1fs, Sun Angle: %.1f°\n", ws.currentTime, ws.sunAngle)
	}
}

func (ws *WaterSimulation) UpdateFixed() {}

// UpdateMovingSun updates the sun position and lighting based on time for full day cycle
func (ws *WaterSimulation) UpdateMovingSun() {
	// Full day cycle: 0° = dawn, 90° = noon, 180° = dusk, 270° = midnight
	sunSpeed := float32(360.0 / DayCycleDuration) // Full 360° cycle
	ws.sunAngle = float32(math.Mod(float64(ws.currentTime*sunSpeed), 360.0))

	// Calculate sun position in 3D space
	sunRad := mgl32.DegToRad(ws.sunAngle)

	// Calculate sun height for lighting (simple fixed height)
	sunHeight := float32(200.0) / 1500.0 // Normalize height for lighting calculations

	// Keep sun sphere in fixed position (don't update position every frame)
	// The sun sphere is already positioned in Start() and should stay there

	// Simple light direction from sun to ocean center
	lightDir := mgl32.Vec3{0, -1, 0}.Normalize() // Light shines down from sun
	ws.engine.Light.Direction = lightDir

	// Calculate sun height for lighting and sky color (0 = horizon, 1 = zenith)
	sunHeight = float32(math.Sin(float64(sunRad)))

	// Set light color and intensity based on sun angle for realistic day cycle
	var lightIntensity float32
	var lightColor mgl32.Vec3

	if ws.sunAngle < 10.0 || ws.sunAngle > 350.0 {
		// Night - moon lighting (sun below horizon)
		lightColor = mgl32.Vec3{0.8, 0.85, 1.0} // Cool moonlight
		lightIntensity = 0.05
	} else if ws.sunAngle < 30.0 || ws.sunAngle > 330.0 {
		// Dawn/Dusk - warm colors (sun low on horizon)
		lightColor = mgl32.Vec3{1.0, 0.7, 0.4} // Warm sunrise/sunset
		lightIntensity = 0.2 + sunHeight*0.4
	} else if ws.sunAngle < 60.0 || ws.sunAngle > 300.0 {
		// Early morning/Late afternoon
		lightColor = mgl32.Vec3{1.0, 0.9, 0.7} // Soft daylight
		lightIntensity = 0.4 + sunHeight*0.6
	} else {
		// Midday - bright natural sunlight
		lightColor = mgl32.Vec3{1.0, 0.98, 0.95} // Bright white sunlight
		lightIntensity = 0.8 + sunHeight*0.4
	}

	ws.engine.Light.Color = lightColor
	ws.engine.Light.Intensity = lightIntensity

	// Keep sky simple - no dynamic changes

	// Update sun sphere color based on time of day
	if ws.sunModel != nil {
		// Keep sun bright and shining
		ws.sunModel.SetDiffuseColor(1.0, 0.9, 0.6) // Bright yellow sun

		// Debug sun position every 30 seconds
		if int(ws.currentTime)%30 == 0 && int(ws.currentTime*10)%10 == 0 {
			fmt.Printf("SUN: Time=%.1fs, Angle=%.1f°, Pos=(%.0f, %.0f, %.0f), Light=%.2f\n",
				ws.currentTime, ws.sunAngle,
				ws.sunModel.Position.X(), ws.sunModel.Position.Y(), ws.sunModel.Position.Z(),
				ws.engine.Light.Intensity)
		}
	}

	// Keep skybox simple - no changes
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
	// Only update time and light direction (dynamic values)
	ws.model.CustomUniforms["time"] = ws.currentTime
	ws.model.CustomUniforms["lightDirection"] = ws.engine.Light.Direction
}
