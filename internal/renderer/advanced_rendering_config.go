package renderer

import "github.com/go-gl/mathgl/mgl32"

// AdvancedRenderingConfig represents configurable advanced rendering features
// Based on techniques from NVIDIA GPU Gems, Real-Time Rendering, and other sources
type AdvancedRenderingConfig struct {
	// Water Caustics (GPU Gems Chapter 2)
	EnableCaustics    bool       `json:"enableCaustics"`
	CausticsIntensity float32    `json:"causticsIntensity"`
	CausticsScale     float32    `json:"causticsScale"`
	CausticsSpeed     mgl32.Vec2 `json:"causticsSpeed"`

	// Procedural Noise (GPU Gems Chapter 5)
	EnablePerlinNoise bool    `json:"enablePerlinNoise"`
	NoiseScale        float32 `json:"noiseScale"`
	NoiseOctaves      int     `json:"noiseOctaves"`
	NoiseIntensity    float32 `json:"noiseIntensity"`

	// Advanced Shadows (GPU Gems Chapters 9 & 11)
	EnableAdvancedShadows bool    `json:"enableAdvancedShadows"`
	ShadowIntensity       float32 `json:"shadowIntensity"`
	ShadowSoftness        float32 `json:"shadowSoftness"`

	// Perspective Shadow Maps (GPU Gems Chapter 14)
	EnablePerspectiveShadows bool `json:"enablePerspectiveShadows"`
	ShadowMapQuality         int  `json:"shadowMapQuality"`

	// Level of Detail and Performance (GPU Gems Chapter 15)
	EnableLOD             bool    `json:"enableLOD"`
	LODTransitionDistance float32 `json:"lodTransitionDistance"`
	PerformanceScaling    float32 `json:"performanceScaling"`

	// Subsurface Scattering (GPU Gems Chapter 16)
	EnableSubsurfaceScattering bool    `json:"enableSubsurfaceScattering"`
	ScatteringIntensity        float32 `json:"scatteringIntensity"`
	ScatteringDepth            float32 `json:"scatteringDepth"`

	// Ambient Occlusion (GPU Gems Chapter 17)
	EnableAmbientOcclusion bool    `json:"enableAmbientOcclusion"`
	AOIntensity            float32 `json:"aoIntensity"`
	AORadius               float32 `json:"aoRadius"`

	// Real-Time Glow Effects (GPU Gems Chapter 21)
	EnableGlow    bool    `json:"enableGlow"`
	GlowIntensity float32 `json:"glowIntensity"`
	GlowRadius    float32 `json:"glowRadius"`

	// High-Quality Filtering and Anti-Aliasing (GPU Gems Chapter 24)
	EnableHighQualityFiltering bool `json:"enableHighQualityFiltering"`
	FilteringQuality           int  `json:"filteringQuality"`
	AntiAliasing               bool `json:"antiAliasing"`

	// Mesh Quality and Tessellation Controls
	EnableMeshSmoothing    bool    `json:"enableMeshSmoothing"`
	MeshSmoothingIntensity float32 `json:"meshSmoothingIntensity"`
	TessellationQuality    int     `json:"tessellationQuality"`
	NormalSmoothingRadius  float32 `json:"normalSmoothingRadius"`

	// Water Reflection and Refraction (based on Medium article techniques)
	EnableWaterReflection    bool    `json:"enableWaterReflection"`
	EnableWaterRefraction    bool    `json:"enableWaterRefraction"`
	WaterReflectionIntensity float32 `json:"waterReflectionIntensity"`
	WaterRefractionIntensity float32 `json:"waterRefractionIntensity"`
	EnableWaterDistortion    bool    `json:"enableWaterDistortion"`
	WaterDistortionIntensity float32 `json:"waterDistortionIntensity"`
	EnableWaterNormalMapping bool    `json:"enableWaterNormalMapping"`
	WaterNormalIntensity     float32 `json:"waterNormalIntensity"`
}

// DefaultAdvancedRenderingConfig returns sensible defaults for all advanced rendering features
func DefaultAdvancedRenderingConfig() AdvancedRenderingConfig {
	return AdvancedRenderingConfig{
		// Caustics - disabled by default for performance
		EnableCaustics:    false,
		CausticsIntensity: 0.3,
		CausticsScale:     0.003,
		CausticsSpeed:     mgl32.Vec2{0.02, 0.015},

		// Perlin Noise - enabled for surface detail
		EnablePerlinNoise: true,
		NoiseScale:        0.0002,
		NoiseOctaves:      3,
		NoiseIntensity:    0.05,

		// Advanced Shadows - enabled for realism
		EnableAdvancedShadows: true,
		ShadowIntensity:       0.3,
		ShadowSoftness:        0.2,

		// Perspective Shadows - disabled by default for performance
		EnablePerspectiveShadows: false,
		ShadowMapQuality:         1024,

		// LOD/Visibility - enabled for performance
		EnableLOD:             true,
		LODTransitionDistance: 50000.0,
		PerformanceScaling:    0.3,

		// Subsurface Scattering - enabled for water realism
		EnableSubsurfaceScattering: true,
		ScatteringIntensity:        0.15,
		ScatteringDepth:            0.005,

		// Ambient Occlusion - enabled for depth perception
		EnableAmbientOcclusion: true,
		AOIntensity:            0.25,
		AORadius:               150.0,

		// Glow effects - disabled by default
		EnableGlow:    false,
		GlowIntensity: 0.3,
		GlowRadius:    0.4,

		// High-Quality Filtering - enabled for smooth results
		EnableHighQualityFiltering: true,
		FilteringQuality:           2, // Multi-pass smoothing
		AntiAliasing:               true,

		// Mesh Quality - enabled for smooth surfaces
		EnableMeshSmoothing:    true,
		MeshSmoothingIntensity: 0.7,
		TessellationQuality:    2, // Medium quality
		NormalSmoothingRadius:  1.0,
	}
}

// WaterAdvancedRenderingConfig returns optimized settings for water rendering
func WaterAdvancedRenderingConfig() AdvancedRenderingConfig {
	config := DefaultAdvancedRenderingConfig()

	// Enable water-specific features
	config.EnableCaustics = false // Can be enabled via API
	config.EnableSubsurfaceScattering = true
	config.EnableAmbientOcclusion = true
	config.EnablePerlinNoise = true
	config.EnableLOD = true
	config.EnableHighQualityFiltering = true
	config.EnableMeshSmoothing = true // Important for water surfaces

	// Balanced settings for beautiful water
	config.CausticsIntensity = 0.3      // Available if enabled
	config.ScatteringIntensity = 0.1    // Subtle
	config.AOIntensity = 0.15           // Gentle depth
	config.NoiseIntensity = 0.02        // Very subtle surface detail
	config.ShadowIntensity = 0.2        // Soft shadows
	config.ShadowSoftness = 0.3         // Smooth shadow edges
	config.MeshSmoothingIntensity = 0.8 // High smoothing for water
	config.TessellationQuality = 3      // High quality for water

	// Water reflection and refraction settings (inspired by Medium article)
	config.EnableWaterReflection = true
	config.EnableWaterRefraction = true
	config.WaterReflectionIntensity = 0.8
	config.WaterRefractionIntensity = 0.6
	config.EnableWaterDistortion = true
	config.WaterDistortionIntensity = 0.3
	config.EnableWaterNormalMapping = true
	config.WaterNormalIntensity = 1.0

	return config
}

// WaterPhotorealisticConfig returns settings optimized for maximum realism
func WaterPhotorealisticConfig() AdvancedRenderingConfig {
	config := WaterAdvancedRenderingConfig()

	// Enable all advanced features for maximum quality
	config.EnableCaustics = true
	config.EnablePerspectiveShadows = true
	config.EnableGlow = true

	// Higher quality settings
	config.CausticsIntensity = 0.4
	config.ScatteringIntensity = 0.2
	config.AOIntensity = 0.2
	config.GlowIntensity = 0.2
	config.FilteringQuality = 3         // Maximum quality
	config.MeshSmoothingIntensity = 0.9 // Maximum smoothing
	config.TessellationQuality = 4      // Maximum quality

	return config
}

// VoxelAdvancedRenderingConfig returns optimized settings for voxel rendering
func VoxelAdvancedRenderingConfig() AdvancedRenderingConfig {
	config := DefaultAdvancedRenderingConfig()

	// Enable voxel-specific features
	config.EnablePerlinNoise = true
	config.EnableAdvancedShadows = true
	config.EnableAmbientOcclusion = true
	config.EnableLOD = true
	config.EnableMeshSmoothing = false // Voxels should stay crisp

	// Optimize for voxel scenes
	config.NoiseIntensity = 0.1
	config.ShadowIntensity = 0.4
	config.AOIntensity = 0.3
	config.PerformanceScaling = 0.5
	config.TessellationQuality = 1 // Low for voxels

	return config
}

// ApplyAdvancedRenderingConfig applies advanced rendering configuration to a model
func ApplyAdvancedRenderingConfig(model *Model, config AdvancedRenderingConfig) {
	if model.CustomUniforms == nil {
		model.CustomUniforms = make(map[string]interface{})
	}

	// Apply caustics settings
	model.CustomUniforms["enableCaustics"] = config.EnableCaustics
	model.CustomUniforms["causticsIntensity"] = config.CausticsIntensity
	model.CustomUniforms["causticsScale"] = config.CausticsScale
	model.CustomUniforms["causticsSpeed"] = config.CausticsSpeed

	// Apply noise settings
	model.CustomUniforms["enablePerlinNoise"] = config.EnablePerlinNoise
	model.CustomUniforms["noiseScale"] = config.NoiseScale
	model.CustomUniforms["noiseOctaves"] = int32(config.NoiseOctaves)
	model.CustomUniforms["noiseIntensity"] = config.NoiseIntensity

	// Apply shadow settings
	model.CustomUniforms["enableShadows"] = config.EnableAdvancedShadows
	model.CustomUniforms["shadowIntensity"] = config.ShadowIntensity
	model.CustomUniforms["shadowSoftness"] = config.ShadowSoftness

	// Apply LOD settings
	model.CustomUniforms["enableLOD"] = config.EnableLOD
	model.CustomUniforms["lodTransitionDistance"] = config.LODTransitionDistance
	model.CustomUniforms["performanceScaling"] = config.PerformanceScaling

	// Apply subsurface scattering
	model.CustomUniforms["enableSubsurfaceScattering"] = config.EnableSubsurfaceScattering
	model.CustomUniforms["scatteringIntensity"] = config.ScatteringIntensity
	model.CustomUniforms["scatteringDepth"] = config.ScatteringDepth

	// Apply ambient occlusion
	model.CustomUniforms["enableAmbientOcclusion"] = config.EnableAmbientOcclusion
	model.CustomUniforms["aoIntensity"] = config.AOIntensity
	model.CustomUniforms["aoRadius"] = config.AORadius

	// Apply glow settings
	model.CustomUniforms["enableGlow"] = config.EnableGlow
	model.CustomUniforms["glowIntensity"] = config.GlowIntensity
	model.CustomUniforms["glowRadius"] = config.GlowRadius

	// Apply filtering settings
	model.CustomUniforms["enableHighQualityFiltering"] = config.EnableHighQualityFiltering
	model.CustomUniforms["filteringQuality"] = int32(config.FilteringQuality)
	model.CustomUniforms["antiAliasing"] = config.AntiAliasing

	// Apply mesh quality settings
	model.CustomUniforms["enableMeshSmoothing"] = config.EnableMeshSmoothing
	model.CustomUniforms["meshSmoothingIntensity"] = config.MeshSmoothingIntensity
	model.CustomUniforms["tessellationQuality"] = int32(config.TessellationQuality)
	model.CustomUniforms["normalSmoothingRadius"] = config.NormalSmoothingRadius
}
