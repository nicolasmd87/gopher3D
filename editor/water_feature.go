package main

import (
	"Gopher3D/internal/behaviour"
	"Gopher3D/internal/engine"
	"Gopher3D/internal/loader"
	"Gopher3D/internal/renderer"
	"fmt"
	"math"
	"time"

	mgl32 "github.com/go-gl/mathgl/mgl32"
)

const (
	WaterResolution = 256 // Reasonable resolution for editor (256x256 = 65k vertices)
	MaxWaves        = 4
)

type WaterSimulation struct {
	model           *renderer.Model
	engine          *engine.Gopher
	shader          renderer.Shader
	startTime       time.Time
	waveCount       int
	waveDirections  []mgl32.Vec3
	waveAmplitudes  []float32
	waveFrequencies []float32
	waveSpeeds      []float32
	wavePhases      []float32
	waveSteepness   []float32
	currentTime     float32
	lastSkyColor    mgl32.Vec3
	
	// Config
	oceanSize       float32
	baseAmplitude   float32
	
	// Editable properties
	WaterColor          mgl32.Vec3
	Transparency        float32
	WaveSpeedMultiplier float32
	WaveHeight          float32 // NEW: Control wave amplitude
	WaveRandomness      float32 // NEW: Add randomness to waves
	
	// Advanced Appearance
	FoamEnabled         bool
	FoamIntensity       float32
	
	// Caustics
	CausticsEnabled     bool
	CausticsIntensity   float32
	CausticsScale       float32
	
	// Lighting & Shadows
	SpecularIntensity   float32
	ShadowStrength      float32
	
	// Distortions
	DistortionStrength  float32
	NormalStrength      float32
	
	// Texture
	TexturePath         string
}

// Exportable config for saving/loading
type WaterSimulationConfig struct {
	OceanSize           float32    `json:"ocean_size"`
	BaseAmplitude       float32    `json:"base_amplitude"`
	WaterColor          [3]float32 `json:"water_color"`
	Transparency        float32    `json:"transparency"`
	WaveSpeedMultiplier float32    `json:"wave_speed_multiplier"`
	FoamEnabled         bool       `json:"foam_enabled"`
	FoamIntensity       float32    `json:"foam_intensity"`
	SpecularIntensity   float32    `json:"specular_intensity"`
	ShadowStrength      float32    `json:"shadow_strength"`
	DistortionStrength  float32    `json:"distortion_strength"`
	NormalStrength      float32    `json:"normal_strength"`
	TexturePath         string     `json:"texture_path"`
}

func NewWaterSimulation(engine *engine.Gopher, size float32, amplitude float32) *WaterSimulation {
	ws := &WaterSimulation{
		engine:              engine,
		shader:              renderer.InitWaterShader(),
		startTime:           time.Now(),
		waveCount:           MaxWaves,
		waveDirections:      make([]mgl32.Vec3, MaxWaves),
		waveAmplitudes:      make([]float32, MaxWaves),
		waveFrequencies:     make([]float32, MaxWaves),
		waveSpeeds:          make([]float32, MaxWaves),
		wavePhases:          make([]float32, MaxWaves),
		waveSteepness:       make([]float32, MaxWaves),
		oceanSize:           size,
		baseAmplitude:       amplitude,
		WaterColor:          mgl32.Vec3{0.06, 0.22, 0.45}, // Natural ocean blue (from working example)
		Transparency:        0.85, // Visual transparency in shader (not alpha blending)
		WaveSpeedMultiplier: 1.0,
		WaveHeight:          1.0,  // Default wave amplitude multiplier
		WaveRandomness:      0.0,  // Default: no randomness (smooth waves)
		FoamEnabled:         true,
		FoamIntensity:       0.5,
		CausticsEnabled:     false, // Disabled by default (performance)
		CausticsIntensity:   0.3,
		CausticsScale:       0.003,
		SpecularIntensity:   1.0,
		ShadowStrength:      0.5,
		DistortionStrength:  0.2,
		NormalStrength:      1.0,
	}

	for i := 0; i < MaxWaves; i++ {
		var amp, freq float32
		if i == 0 {
			amp = ws.baseAmplitude * 1.2
			freq = 0.00008
		} else if i == 1 {
			amp = ws.baseAmplitude * 0.8
			freq = 0.00015
		} else if i == 2 {
			amp = ws.baseAmplitude * 0.6
			freq = 0.0004
		} else {
			amp = ws.baseAmplitude * 0.4
			freq = 0.0008
		}

		baseAngle := float32(i) * 45.0 * math.Pi / 180.0
		dirX := float32(math.Cos(float64(baseAngle)))
		dirZ := float32(math.Sin(float64(baseAngle)))
		ws.waveDirections[i] = mgl32.Vec3{dirX, 0.0, dirZ}.Normalize()
		ws.waveAmplitudes[i] = amp
		ws.waveFrequencies[i] = freq

		wavelength := 2.0 * math.Pi / float64(freq)
		physicalSpeed := float32(0.002 * math.Sqrt(wavelength))
		ws.waveSpeeds[i] = physicalSpeed
		ws.wavePhases[i] = float32(i) * math.Pi / 3.0
		ws.waveSteepness[i] = 0.2 + float32(i)*0.1
	}
	return ws
}

func (ws *WaterSimulation) Start() {
	ws.engine.SetFaceCulling(false)

	// Prevent re-initialization if model already exists
	if ws.model != nil {
		return
	}

	// Center water at (0,0,0) for the editor
	oceanCenter := float32(0)
	
	// Load water surface
	model, err := loader.LoadWaterSurface(ws.oceanSize, oceanCenter, oceanCenter, WaterResolution)
	if err != nil {
		fmt.Printf("Failed to load water: %v\n", err)
		return
	}

	// Reset position to origin and set Name
	model.SetPosition(0, 0, 0)
	model.Name = "Water Surface"

	model.SetDiffuseColor(ws.WaterColor.X(), ws.WaterColor.Y(), ws.WaterColor.Z())
	model.SetMaterialPBR(0.0, 0.4)
	model.SetExposure(1.0)
	model.SetAlpha(1.0) // CRITICAL: Water must be OPAQUE (like the working example)
	model.Shader = ws.shader
	
	// Tag as water for Inspector
	if model.Metadata == nil {
		model.Metadata = make(map[string]interface{})
	}
	model.Metadata["type"] = "water"

	ws.model = model
	ws.setupWaterUniforms()
	ws.engine.AddModel(model)
	createGameObjectForModel(model)
	
	// Set initial sky color
	ws.lastSkyColor = mgl32.Vec3{skyboxSolidColor[0], skyboxSolidColor[1], skyboxSolidColor[2]}
}

func (ws *WaterSimulation) Update() {
	ws.currentTime = float32(time.Since(ws.startTime).Seconds())
	
	if ws.model == nil || ws.model.CustomUniforms == nil {
		return
	}

	// Sync with global sky color from editor
	ws.lastSkyColor = mgl32.Vec3{skyboxSolidColor[0], skyboxSolidColor[1], skyboxSolidColor[2]}

	ws.model.CustomUniforms["time"] = ws.currentTime
	
	// Update editable uniforms
	ws.model.CustomUniforms["waterBaseColor"] = ws.WaterColor
	ws.model.CustomUniforms["waterTransparency"] = ws.Transparency
	ws.model.CustomUniforms["waveSpeedMultiplier"] = ws.WaveSpeedMultiplier
	
	// Keep water opaque for depth testing (shader handles visual transparency)
	// The waterTransparency uniform controls the shader's appearance, not actual alpha blending
	if ws.model.Material != nil {
		ws.model.Material.Alpha = 1.0 // Always opaque
	}
	
	// Update water plane height dynamic to model position
	// This fixes the issue where water visual effects (fog/caustics) don't move with the model
	ws.model.CustomUniforms["waterPlaneHeight"] = ws.model.Position.Y() + 5.0
	
	// Send advanced uniforms
	ws.model.CustomUniforms["enableFog"] = ws.FoamEnabled
	ws.model.CustomUniforms["fogIntensity"] = ws.FoamIntensity
	ws.model.CustomUniforms["enableCaustics"] = ws.CausticsEnabled
	ws.model.CustomUniforms["causticsIntensity"] = ws.CausticsIntensity
	ws.model.CustomUniforms["causticsScale"] = ws.CausticsScale
	ws.model.CustomUniforms["waterReflectionIntensity"] = ws.SpecularIntensity
	ws.model.CustomUniforms["shadowIntensity"] = ws.ShadowStrength
	ws.model.CustomUniforms["waterDistortionIntensity"] = ws.DistortionStrength
	ws.model.CustomUniforms["waterNormalIntensity"] = ws.NormalStrength
	
	// Get the active light - prefer renderer's lights array as source of truth
	var activeLight *renderer.Light
	if openglRenderer, ok := ws.engine.GetRenderer().(*renderer.OpenGLRenderer); ok {
		lights := openglRenderer.GetLights()
		if len(lights) > 0 {
			activeLight = lights[0]
		}
	}
	
	// Fallback to engine.Light if no lights in renderer
	if activeLight == nil && ws.engine.Light != nil {
		activeLight = ws.engine.Light
	}
	
	// Set light uniforms for water shader (uses different names than default shader)
	if activeLight != nil {
		ws.model.CustomUniforms["lightPos"] = activeLight.Position
		ws.model.CustomUniforms["lightColor"] = activeLight.Color  
		ws.model.CustomUniforms["lightIntensity"] = activeLight.Intensity
		
		// Debug once per second
		if int(ws.currentTime)%5 == 0 && ws.currentTime-float32(int(ws.currentTime)) < 0.016 {
			fmt.Printf("[WATER] Light: I=%.2f, C=(%.2f,%.2f,%.2f), Mode=%s, Dir=(%.3f,%.3f,%.3f)\n",
				activeLight.Intensity,
				activeLight.Color.X(), activeLight.Color.Y(), activeLight.Color.Z(),
				activeLight.Mode,
				activeLight.Direction.X(), activeLight.Direction.Y(), activeLight.Direction.Z())
		}
		
		// Light direction handling - CRITICAL for reflections
		if activeLight.Mode == "directional" {
			dir := activeLight.Direction
			if dir.Len() > 0.0001 {
				// Ensure direction points FROM light TO surface (shader expectation)
				normalizedDir := dir.Normalize()
				ws.model.CustomUniforms["lightDirection"] = normalizedDir
			} else {
				ws.model.CustomUniforms["lightDirection"] = mgl32.Vec3{0, -1, 0}
			}
		} else {
			ws.model.CustomUniforms["lightDirection"] = mgl32.Vec3{0, 0, 0}
		}
	} else {
		fmt.Printf("[WATER ERROR] No light found!\n")
		// Emergency fallback if no light exists
		ws.model.CustomUniforms["lightPos"] = mgl32.Vec3{0, 1000, 0}
		ws.model.CustomUniforms["lightColor"] = mgl32.Vec3{1.0, 0.95, 0.85}
		ws.model.CustomUniforms["lightIntensity"] = float32(4.5)
		ws.model.CustomUniforms["lightDirection"] = mgl32.Vec3{-0.3, -1.0, -0.5}.Normalize()
	}
	ws.model.CustomUniforms["skyColor"] = ws.lastSkyColor
	ws.model.CustomUniforms["horizonColor"] = mgl32.Vec3{
		ws.lastSkyColor.X() * 0.85,
		ws.lastSkyColor.Y() * 0.85,
		ws.lastSkyColor.Z() * 0.85,
	}
}

func (ws *WaterSimulation) UpdateFixed() {}

func (ws *WaterSimulation) setupWaterUniforms() {
	if ws.model.CustomUniforms == nil {
		ws.model.CustomUniforms = make(map[string]interface{})
	}

	ws.model.CustomUniforms["waveCount"] = int32(ws.waveCount)

	directions := make([]float32, MaxWaves*3)
	amplitudes := make([]float32, MaxWaves)
	frequencies := make([]float32, MaxWaves)
	speeds := make([]float32, MaxWaves)
	phases := make([]float32, MaxWaves)
	steepness := make([]float32, MaxWaves)

	for i := 0; i < MaxWaves; i++ {
		directions[i*3] = ws.waveDirections[i].X()
		directions[i*3+1] = ws.waveDirections[i].Y()
		directions[i*3+2] = ws.waveDirections[i].Z()
		amplitudes[i] = ws.waveAmplitudes[i]
		frequencies[i] = ws.waveFrequencies[i]
		speeds[i] = ws.waveSpeeds[i]
		phases[i] = ws.wavePhases[i]
		steepness[i] = ws.waveSteepness[i]
	}

	ws.model.CustomUniforms["waveDirections"] = directions
	ws.model.CustomUniforms["waveAmplitudes"] = amplitudes
	ws.model.CustomUniforms["waveFrequencies"] = frequencies
	ws.model.CustomUniforms["waveSpeeds"] = speeds
	ws.model.CustomUniforms["wavePhases"] = phases
	ws.model.CustomUniforms["waveSteepness"] = steepness
	
	// CRITICAL: Set wave height multiplier (default 1.0) - without this, waves have zero amplitude!
	ws.model.CustomUniforms["waveHeightMultiplier"] = ws.WaveHeight
	ws.model.CustomUniforms["waveRandomness"] = ws.WaveRandomness
	ws.model.CustomUniforms["time"] = float32(0.0) // Initial time

	// Apply photorealistic water rendering configuration (BASE DEFAULTS)
	// This ensures all hidden uniforms are set correctly, matching the working example
	waterRenderConfig := renderer.WaterPhotorealisticConfig()
	waterRenderConfig.MeshSmoothingIntensity = 0.85
	waterRenderConfig.FilteringQuality = 3
	waterRenderConfig.AntiAliasing = true
	waterRenderConfig.NormalSmoothingRadius = 1.2
	waterRenderConfig.EnableCaustics = ws.CausticsEnabled
	waterRenderConfig.NoiseIntensity = 0.0
	
	renderer.ApplyWaterRenderingConfig(ws.model, waterRenderConfig)
	
	// Water-specific uniforms
	ws.model.CustomUniforms["waterPlaneHeight"] = float32(5.0)

	// Initial height - dynamic update in Update()
	ws.model.CustomUniforms["waterBaseColor"] = ws.WaterColor
	ws.model.CustomUniforms["waterTransparency"] = ws.Transparency
	ws.model.CustomUniforms["waveSpeedMultiplier"] = ws.WaveSpeedMultiplier
	
	// Fog config
	waterConfig := renderer.WaterConfig{
		EnableFog:    true,
		FogStart:     ws.oceanSize * 0.1,
		FogEnd:       ws.oceanSize * 0.8,
		FogIntensity: 0.3,
		FogColor:     mgl32.Vec3{0.5, 0.7, 0.9},
		SkyColor:     ws.lastSkyColor,
		HorizonColor: mgl32.Vec3{
			ws.lastSkyColor.X() * 0.85,
			ws.lastSkyColor.Y() * 0.85,
			ws.lastSkyColor.Z() * 0.85,
		},
	}
	renderer.ApplyWaterConfig(ws.model, waterConfig)
}

// RestoreWaterSimulation restores a water simulation from a saved configuration
func RestoreWaterSimulation(eng *engine.Gopher, model *renderer.Model, config WaterSimulationConfig) {
	// Create sim instance
	ws := NewWaterSimulation(eng, config.OceanSize, config.BaseAmplitude)
	
	// Apply config
	ws.WaterColor = mgl32.Vec3{config.WaterColor[0], config.WaterColor[1], config.WaterColor[2]}
	ws.Transparency = config.Transparency
	ws.WaveSpeedMultiplier = config.WaveSpeedMultiplier
	ws.FoamEnabled = config.FoamEnabled
	ws.FoamIntensity = config.FoamIntensity
	ws.SpecularIntensity = config.SpecularIntensity
	ws.ShadowStrength = config.ShadowStrength
	ws.DistortionStrength = config.DistortionStrength
	ws.NormalStrength = config.NormalStrength
	ws.TexturePath = config.TexturePath
	
	// Link model
	ws.model = model
	model.Shader = ws.shader
	model.Name = "Water Surface"
	
	// Ensure metadata is set
	if model.Metadata == nil {
		model.Metadata = make(map[string]interface{})
	}
	model.Metadata["type"] = "water"
	
	// Re-setup uniforms
	ws.setupWaterUniforms()
	
	// Set as active
	activeWaterSim = ws
	
	// Register behavior (Start() will be called by manager)
	behaviour.GlobalBehaviourManager.Add(ws)
}

// ApplyChanges forces water uniform updates when properties change via UI
func (ws *WaterSimulation) ApplyChanges() {
	if ws.model == nil || ws.model.CustomUniforms == nil {
		return
	}
	
	// Update editable properties directly (same as Update() but without time/light)
	ws.model.CustomUniforms["waterBaseColor"] = ws.WaterColor
	ws.model.CustomUniforms["waterTransparency"] = ws.Transparency
	ws.model.CustomUniforms["waveSpeedMultiplier"] = ws.WaveSpeedMultiplier
	ws.model.CustomUniforms["waveHeightMultiplier"] = ws.WaveHeight
	ws.model.CustomUniforms["waveRandomness"] = ws.WaveRandomness
	ws.model.CustomUniforms["enableFog"] = ws.FoamEnabled
	ws.model.CustomUniforms["fogIntensity"] = ws.FoamIntensity
	ws.model.CustomUniforms["enableCaustics"] = ws.CausticsEnabled
	ws.model.CustomUniforms["causticsIntensity"] = ws.CausticsIntensity
	ws.model.CustomUniforms["causticsScale"] = ws.CausticsScale
	ws.model.CustomUniforms["waterReflectionIntensity"] = ws.SpecularIntensity
	ws.model.CustomUniforms["shadowIntensity"] = ws.ShadowStrength
	ws.model.CustomUniforms["waterDistortionIntensity"] = ws.DistortionStrength
	ws.model.CustomUniforms["waterNormalIntensity"] = ws.NormalStrength
	
	// Update material alpha if needed
	if ws.model.Material != nil {
		ws.model.Material.Alpha = 1.0
	}
}
