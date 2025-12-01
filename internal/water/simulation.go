// Package water provides water simulation with Gerstner waves for realistic ocean rendering.
// This package is shared between the editor and exported games.
package water

import (
	"Gopher3D/internal/engine"
	"Gopher3D/internal/loader"
	"Gopher3D/internal/renderer"
	"fmt"
	"math"
	"time"

	mgl32 "github.com/go-gl/mathgl/mgl32"
)

const (
	// WaterResolution is the grid resolution for water mesh (256x256 = 65k vertices)
	WaterResolution = 256
	// MaxWaves is the maximum number of Gerstner waves
	MaxWaves = 4
)

// Simulation handles water rendering with Gerstner waves
type Simulation struct {
	Model           *renderer.Model
	Engine          *engine.Gopher
	Shader          renderer.Shader
	StartTime       time.Time
	WaveCount       int
	WaveDirections  []mgl32.Vec3
	WaveAmplitudes  []float32
	WaveFrequencies []float32
	WaveSpeeds      []float32
	WavePhases      []float32
	WaveSteepness   []float32
	CurrentTime     float32
	LastSkyColor    mgl32.Vec3

	// Config
	OceanSize     float32
	BaseAmplitude float32

	// Editable properties
	WaterColor          mgl32.Vec3
	Transparency        float32
	WaveSpeedMultiplier float32
	WaveHeight          float32
	WaveRandomness      float32

	// Advanced Appearance
	FoamEnabled   bool
	FoamIntensity float32

	// Caustics
	CausticsEnabled   bool
	CausticsIntensity float32
	CausticsScale     float32

	// Lighting & Shadows
	SpecularIntensity float32
	ShadowStrength    float32

	// Distortions
	DistortionStrength float32
	NormalStrength     float32

	// Texture
	TexturePath string
}

// Config is an exportable config for saving/loading water settings
type Config struct {
	OceanSize           float32    `json:"ocean_size"`
	BaseAmplitude       float32    `json:"base_amplitude"`
	WaterColor          [3]float32 `json:"water_color"`
	Transparency        float32    `json:"transparency"`
	WaveSpeedMultiplier float32    `json:"wave_speed_multiplier"`
	WaveHeight          float32    `json:"wave_height"`
	WaveRandomness      float32    `json:"wave_randomness"`
	FoamEnabled         bool       `json:"foam_enabled"`
	FoamIntensity       float32    `json:"foam_intensity"`
	CausticsEnabled     bool       `json:"caustics_enabled"`
	CausticsIntensity   float32    `json:"caustics_intensity"`
	CausticsScale       float32    `json:"caustics_scale"`
	SpecularIntensity   float32    `json:"specular_intensity"`
	ShadowStrength      float32    `json:"shadow_strength"`
	DistortionStrength  float32    `json:"distortion_strength"`
	NormalStrength      float32    `json:"normal_strength"`
	TexturePath         string     `json:"texture_path"`
}

// NewSimulation creates a new water simulation with the given size and amplitude
func NewSimulation(eng *engine.Gopher, size float32, amplitude float32) *Simulation {
	ws := &Simulation{
		Engine:              eng,
		Shader:              renderer.InitWaterShader(),
		StartTime:           time.Now(),
		WaveCount:           MaxWaves,
		WaveDirections:      make([]mgl32.Vec3, MaxWaves),
		WaveAmplitudes:      make([]float32, MaxWaves),
		WaveFrequencies:     make([]float32, MaxWaves),
		WaveSpeeds:          make([]float32, MaxWaves),
		WavePhases:          make([]float32, MaxWaves),
		WaveSteepness:       make([]float32, MaxWaves),
		OceanSize:           size,
		BaseAmplitude:       amplitude,
		WaterColor:          mgl32.Vec3{0.06, 0.22, 0.45}, // Natural ocean blue
		Transparency:        0.85,
		WaveSpeedMultiplier: 1.0,
		WaveHeight:          1.0,
		WaveRandomness:      0.0,
		FoamEnabled:         true,
		FoamIntensity:       0.5,
		CausticsEnabled:     false,
		CausticsIntensity:   0.3,
		CausticsScale:       0.003,
		SpecularIntensity:   1.0,
		ShadowStrength:      0.5,
		DistortionStrength:  0.2,
		NormalStrength:      1.0,
		LastSkyColor:        mgl32.Vec3{0.5, 0.7, 1.0}, // Default sky blue
	}

	// Initialize wave parameters
	for i := 0; i < MaxWaves; i++ {
		var amp, freq float32
		if i == 0 {
			amp = ws.BaseAmplitude * 1.2
			freq = 0.00008
		} else if i == 1 {
			amp = ws.BaseAmplitude * 0.8
			freq = 0.00015
		} else if i == 2 {
			amp = ws.BaseAmplitude * 0.6
			freq = 0.0004
		} else {
			amp = ws.BaseAmplitude * 0.4
			freq = 0.0008
		}

		baseAngle := float32(i) * 45.0 * math.Pi / 180.0
		dirX := float32(math.Cos(float64(baseAngle)))
		dirZ := float32(math.Sin(float64(baseAngle)))
		ws.WaveDirections[i] = mgl32.Vec3{dirX, 0.0, dirZ}.Normalize()
		ws.WaveAmplitudes[i] = amp
		ws.WaveFrequencies[i] = freq

		wavelength := 2.0 * math.Pi / float64(freq)
		physicalSpeed := float32(0.002 * math.Sqrt(wavelength))
		ws.WaveSpeeds[i] = physicalSpeed
		ws.WavePhases[i] = float32(i) * math.Pi / 3.0
		ws.WaveSteepness[i] = 0.2 + float32(i)*0.1
	}
	return ws
}

// InitializeMesh creates the water mesh without adding it to the engine.
// Returns the model so the caller can handle registration.
func (ws *Simulation) InitializeMesh() *renderer.Model {
	ws.Engine.SetFaceCulling(false)

	// Prevent re-initialization if model already exists
	if ws.Model != nil {
		return ws.Model
	}

	// Compile shader early to ensure it's ready for uniforms
	if !ws.Shader.IsCompiled() {
		ws.Shader.Compile()
	}

	// Center water at (0,0,0)
	oceanCenter := float32(0)

	// Load water surface
	model, err := loader.LoadWaterSurface(ws.OceanSize, oceanCenter, oceanCenter, WaterResolution)
	if err != nil {
		fmt.Printf("Failed to load water: %v\n", err)
		return nil
	}

	// Reset position to origin and set Name
	model.SetPosition(0, 0, 0)
	model.Name = "Water Surface"

	model.SetDiffuseColor(ws.WaterColor.X(), ws.WaterColor.Y(), ws.WaterColor.Z())
	model.SetMaterialPBR(0.0, 0.4)
	model.SetExposure(1.0)
	model.SetAlpha(1.0) // Water must be OPAQUE for proper depth testing
	model.Shader = ws.Shader

	// Tag as water
	if model.Metadata == nil {
		model.Metadata = make(map[string]interface{})
	}
	model.Metadata["type"] = "water"

	ws.Model = model
	ws.SetupWaterUniforms()

	return model
}

// SetSkyColor updates the sky color used for reflections
func (ws *Simulation) SetSkyColor(color mgl32.Vec3) {
	ws.LastSkyColor = color
}

// Start implements the Behaviour interface - initializes and adds model to engine
func (ws *Simulation) Start() {
	model := ws.InitializeMesh()
	if model == nil {
		return
	}

	// Add to engine if not already added
	if openglRenderer, ok := ws.Engine.GetRenderer().(*renderer.OpenGLRenderer); ok {
		models := openglRenderer.GetModels()
		alreadyAdded := false
		for _, m := range models {
			if m == model {
				alreadyAdded = true
				break
			}
		}
		if !alreadyAdded {
			ws.Engine.AddModel(model)
		}
	}
}

// Update implements the Behaviour interface - called every frame
func (ws *Simulation) Update() {
	ws.CurrentTime = float32(time.Since(ws.StartTime).Seconds())

	if ws.Model == nil || ws.Model.CustomUniforms == nil {
		return
	}

	ws.Model.CustomUniforms["time"] = ws.CurrentTime

	// Update editable uniforms
	ws.Model.CustomUniforms["waterBaseColor"] = ws.WaterColor
	ws.Model.CustomUniforms["waterTransparency"] = ws.Transparency
	ws.Model.CustomUniforms["waveSpeedMultiplier"] = ws.WaveSpeedMultiplier

	// Keep water opaque for depth testing
	if ws.Model.Material != nil {
		ws.Model.Material.Alpha = 1.0
	}

	// Update water plane height dynamically
	ws.Model.CustomUniforms["waterPlaneHeight"] = ws.Model.Position.Y() + 5.0

	// Note: enableFog is for atmospheric fog, not foam. Foam is calculated internally by shader.
	ws.Model.CustomUniforms["enableFog"] = false // Disable fog by default for cleaner water
	ws.Model.CustomUniforms["fogIntensity"] = float32(0.3)
	ws.Model.CustomUniforms["enableCaustics"] = ws.CausticsEnabled
	ws.Model.CustomUniforms["causticsIntensity"] = ws.CausticsIntensity
	ws.Model.CustomUniforms["causticsScale"] = ws.CausticsScale
	ws.Model.CustomUniforms["waterReflectionIntensity"] = ws.SpecularIntensity
	ws.Model.CustomUniforms["shadowIntensity"] = ws.ShadowStrength
	ws.Model.CustomUniforms["waterDistortionIntensity"] = ws.DistortionStrength
	ws.Model.CustomUniforms["waterNormalIntensity"] = ws.NormalStrength

	// Get the active light
	var activeLight *renderer.Light
	if openglRenderer, ok := ws.Engine.GetRenderer().(*renderer.OpenGLRenderer); ok {
		lights := openglRenderer.GetLights()
		if len(lights) > 0 {
			activeLight = lights[0]
		}
	}

	// Fallback to engine.Light
	if activeLight == nil && ws.Engine.Light != nil {
		activeLight = ws.Engine.Light
	}

	// Set light uniforms for water shader
	if activeLight != nil {
		ws.Model.CustomUniforms["lightPos"] = activeLight.Position
		ws.Model.CustomUniforms["lightColor"] = activeLight.Color
		ws.Model.CustomUniforms["lightIntensity"] = activeLight.Intensity

		// Light direction handling
		if activeLight.Mode == "directional" {
			dir := activeLight.Direction
			if dir.Len() > 0.0001 {
				normalizedDir := dir.Normalize()
				ws.Model.CustomUniforms["lightDirection"] = normalizedDir
			} else {
				ws.Model.CustomUniforms["lightDirection"] = mgl32.Vec3{0, -1, 0}
			}
		} else {
			ws.Model.CustomUniforms["lightDirection"] = mgl32.Vec3{0, 0, 0}
		}
	} else {
		// Emergency fallback if no light exists
		ws.Model.CustomUniforms["lightPos"] = mgl32.Vec3{0, 1000, 0}
		ws.Model.CustomUniforms["lightColor"] = mgl32.Vec3{1.0, 0.95, 0.85}
		ws.Model.CustomUniforms["lightIntensity"] = float32(4.5)
		ws.Model.CustomUniforms["lightDirection"] = mgl32.Vec3{-0.3, -1.0, -0.5}.Normalize()
	}

	ws.Model.CustomUniforms["skyColor"] = ws.LastSkyColor
	ws.Model.CustomUniforms["horizonColor"] = mgl32.Vec3{
		ws.LastSkyColor.X() * 0.85,
		ws.LastSkyColor.Y() * 0.85,
		ws.LastSkyColor.Z() * 0.85,
	}
}

// UpdateFixed implements the Behaviour interface
func (ws *Simulation) UpdateFixed() {}

// SetupWaterUniforms initializes all water shader uniforms
func (ws *Simulation) SetupWaterUniforms() {
	if ws.Model.CustomUniforms == nil {
		ws.Model.CustomUniforms = make(map[string]interface{})
	}

	ws.Model.CustomUniforms["waveCount"] = int32(ws.WaveCount)

	directions := make([]float32, MaxWaves*3)
	amplitudes := make([]float32, MaxWaves)
	frequencies := make([]float32, MaxWaves)
	speeds := make([]float32, MaxWaves)
	phases := make([]float32, MaxWaves)
	steepness := make([]float32, MaxWaves)

	for i := 0; i < MaxWaves; i++ {
		directions[i*3] = ws.WaveDirections[i].X()
		directions[i*3+1] = ws.WaveDirections[i].Y()
		directions[i*3+2] = ws.WaveDirections[i].Z()
		amplitudes[i] = ws.WaveAmplitudes[i]
		frequencies[i] = ws.WaveFrequencies[i]
		speeds[i] = ws.WaveSpeeds[i]
		phases[i] = ws.WavePhases[i]
		steepness[i] = ws.WaveSteepness[i]
	}

	ws.Model.CustomUniforms["waveDirections"] = directions
	ws.Model.CustomUniforms["waveAmplitudes"] = amplitudes
	ws.Model.CustomUniforms["waveFrequencies"] = frequencies
	ws.Model.CustomUniforms["waveSpeeds"] = speeds
	ws.Model.CustomUniforms["wavePhases"] = phases
	ws.Model.CustomUniforms["waveSteepness"] = steepness

	ws.Model.CustomUniforms["waveHeightMultiplier"] = ws.WaveHeight
	ws.Model.CustomUniforms["waveRandomness"] = ws.WaveRandomness
	ws.Model.CustomUniforms["time"] = float32(0.0)

	// Apply photorealistic water rendering configuration
	waterRenderConfig := renderer.WaterPhotorealisticConfig()
	waterRenderConfig.MeshSmoothingIntensity = 0.85
	waterRenderConfig.FilteringQuality = 3
	waterRenderConfig.AntiAliasing = true
	waterRenderConfig.NormalSmoothingRadius = 1.2
	waterRenderConfig.EnableCaustics = ws.CausticsEnabled
	waterRenderConfig.NoiseIntensity = 0.0

	renderer.ApplyWaterRenderingConfig(ws.Model, waterRenderConfig)

	// Explicitly disable shadows for water (the shadow patterns look bad)
	ws.Model.CustomUniforms["enableShadows"] = false

	// Water-specific uniforms
	ws.Model.CustomUniforms["waterPlaneHeight"] = float32(5.0)
	ws.Model.CustomUniforms["waterBaseColor"] = ws.WaterColor
	ws.Model.CustomUniforms["waterTransparency"] = ws.Transparency
	ws.Model.CustomUniforms["waveSpeedMultiplier"] = ws.WaveSpeedMultiplier

	// Fog config
	waterConfig := renderer.WaterConfig{
		EnableFog:    true,
		FogStart:     ws.OceanSize * 0.1,
		FogEnd:       ws.OceanSize * 0.8,
		FogIntensity: 0.3,
		FogColor:     mgl32.Vec3{0.5, 0.7, 0.9},
		SkyColor:     ws.LastSkyColor,
		HorizonColor: mgl32.Vec3{
			ws.LastSkyColor.X() * 0.85,
			ws.LastSkyColor.Y() * 0.85,
			ws.LastSkyColor.Z() * 0.85,
		},
	}
	renderer.ApplyWaterConfig(ws.Model, waterConfig)
}

// ApplyChanges forces water uniform updates when properties change
func (ws *Simulation) ApplyChanges() {
	if ws.Model == nil || ws.Model.CustomUniforms == nil {
		return
	}

	ws.Model.CustomUniforms["waterBaseColor"] = ws.WaterColor
	ws.Model.CustomUniforms["waterTransparency"] = ws.Transparency
	ws.Model.CustomUniforms["waveSpeedMultiplier"] = ws.WaveSpeedMultiplier
	ws.Model.CustomUniforms["waveHeightMultiplier"] = ws.WaveHeight
	ws.Model.CustomUniforms["waveRandomness"] = ws.WaveRandomness
	// Note: enableFog is for atmospheric fog, disabled for cleaner water
	ws.Model.CustomUniforms["enableFog"] = false
	ws.Model.CustomUniforms["fogIntensity"] = float32(0.3)
	ws.Model.CustomUniforms["enableCaustics"] = ws.CausticsEnabled
	ws.Model.CustomUniforms["causticsIntensity"] = ws.CausticsIntensity
	ws.Model.CustomUniforms["causticsScale"] = ws.CausticsScale
	ws.Model.CustomUniforms["waterReflectionIntensity"] = ws.SpecularIntensity
	ws.Model.CustomUniforms["shadowIntensity"] = ws.ShadowStrength
	ws.Model.CustomUniforms["waterDistortionIntensity"] = ws.DistortionStrength
	ws.Model.CustomUniforms["waterNormalIntensity"] = ws.NormalStrength

	if ws.Model.Material != nil {
		ws.Model.Material.Alpha = 1.0
	}
}

// GetConfig returns the current configuration for saving
func (ws *Simulation) GetConfig() Config {
	return Config{
		OceanSize:           ws.OceanSize,
		BaseAmplitude:       ws.BaseAmplitude,
		WaterColor:          [3]float32{ws.WaterColor.X(), ws.WaterColor.Y(), ws.WaterColor.Z()},
		Transparency:        ws.Transparency,
		WaveSpeedMultiplier: ws.WaveSpeedMultiplier,
		WaveHeight:          ws.WaveHeight,
		WaveRandomness:      ws.WaveRandomness,
		FoamEnabled:         ws.FoamEnabled,
		FoamIntensity:       ws.FoamIntensity,
		CausticsEnabled:     ws.CausticsEnabled,
		CausticsIntensity:   ws.CausticsIntensity,
		CausticsScale:       ws.CausticsScale,
		SpecularIntensity:   ws.SpecularIntensity,
		ShadowStrength:      ws.ShadowStrength,
		DistortionStrength:  ws.DistortionStrength,
		NormalStrength:      ws.NormalStrength,
		TexturePath:         ws.TexturePath,
	}
}

// ApplyConfig applies a saved configuration to the simulation
func (ws *Simulation) ApplyConfig(config Config) {
	ws.OceanSize = config.OceanSize
	ws.BaseAmplitude = config.BaseAmplitude
	ws.WaterColor = mgl32.Vec3{config.WaterColor[0], config.WaterColor[1], config.WaterColor[2]}
	ws.Transparency = config.Transparency
	ws.WaveSpeedMultiplier = config.WaveSpeedMultiplier
	ws.WaveHeight = config.WaveHeight
	ws.WaveRandomness = config.WaveRandomness
	ws.FoamEnabled = config.FoamEnabled
	ws.FoamIntensity = config.FoamIntensity
	ws.CausticsEnabled = config.CausticsEnabled
	ws.CausticsIntensity = config.CausticsIntensity
	ws.CausticsScale = config.CausticsScale
	ws.SpecularIntensity = config.SpecularIntensity
	ws.ShadowStrength = config.ShadowStrength
	ws.DistortionStrength = config.DistortionStrength
	ws.NormalStrength = config.NormalStrength
	ws.TexturePath = config.TexturePath
}
