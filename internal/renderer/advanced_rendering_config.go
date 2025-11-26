package renderer

import "github.com/go-gl/mathgl/mgl32"

// AdvancedRenderingConfig represents configurable advanced rendering features
// Based on techniques from NVIDIA GPU Gems, Real-Time Rendering, and modern research
type AdvancedRenderingConfig struct {
	// Core Lighting Features
	EnableAdvancedLighting bool `json:"enableAdvancedLighting"`

	// Modern PBR Extensions
	EnableClearcoat    bool       `json:"enableClearcoat"`
	ClearcoatRoughness float32    `json:"clearcoatRoughness"`
	ClearcoatIntensity float32    `json:"clearcoatIntensity"`
	EnableSheen        bool       `json:"enableSheen"`
	SheenColor         mgl32.Vec3 `json:"sheenColor"`
	SheenRoughness     float32    `json:"sheenRoughness"`
	EnableTransmission bool       `json:"enableTransmission"`
	TransmissionFactor float32    `json:"transmissionFactor"`

	// Advanced Lighting Models
	EnableMultipleScattering bool    `json:"enableMultipleScattering"`
	EnableEnergyConservation bool    `json:"enableEnergyConservation"`
	EnableImageBasedLighting bool    `json:"enableImageBasedLighting"`
	IBLIntensity             float32 `json:"iblIntensity"`

	// Procedural Noise (GPU Gems Chapter 5)
	EnablePerlinNoise bool    `json:"enablePerlinNoise"`
	NoiseScale        float32 `json:"noiseScale"`
	NoiseOctaves      int     `json:"noiseOctaves"`
	NoiseIntensity    float32 `json:"noiseIntensity"`

	// Advanced Shadows with Modern Techniques
	EnableAdvancedShadows bool    `json:"enableAdvancedShadows"`
	ShadowIntensity       float32 `json:"shadowIntensity"`
	ShadowSoftness        float32 `json:"shadowSoftness"`

	// Volumetric Lighting
	EnableVolumetricLighting bool    `json:"enableVolumetricLighting"`
	VolumetricIntensity      float32 `json:"volumetricIntensity"`
	VolumetricSteps          int     `json:"volumetricSteps"`
	VolumetricScattering     float32 `json:"volumetricScattering"`

	// Subsurface Scattering (Enhanced)
	EnableSubsurfaceScattering bool       `json:"enableSubsurfaceScattering"`
	ScatteringIntensity        float32    `json:"scatteringIntensity"`
	ScatteringDepth            float32    `json:"scatteringDepth"`
	ScatteringColor            mgl32.Vec3 `json:"scatteringColor"`

	// Screen Space Ambient Occlusion (SSAO)
	EnableSSAO      bool    `json:"enableSSAO"`
	SSAOIntensity   float32 `json:"ssaoIntensity"`
	SSAORadius      float32 `json:"ssaoRadius"`
	SSAOBias        float32 `json:"ssaoBias"`
	SSAOSampleCount int     `json:"ssaoSampleCount"`

	// Real-Time Global Illumination
	EnableGlobalIllumination bool    `json:"enableGlobalIllumination"`
	GIIntensity              float32 `json:"giIntensity"`
	GIBounces                int     `json:"giBounces"`

	// Bloom and HDR Effects
	EnableBloom    bool    `json:"enableBloom"`
	BloomThreshold float32 `json:"bloomThreshold"`
	BloomIntensity float32 `json:"bloomIntensity"`
	BloomRadius    float32 `json:"bloomRadius"`

	// High-Quality Filtering
	EnableHighQualityFiltering bool `json:"enableHighQualityFiltering"`
	FilteringQuality           int  `json:"filteringQuality"`
	
	// Anti-Aliasing (AA) - Not part of custom uniforms, controlled at renderer level
	// These are stored here for persistence but applied via engine/renderer, not shader uniforms
	MSAASamples int  `json:"msaaSamples"` // 0, 2, 4, 8, 16 (hardware MSAA, requires restart)
	EnableFXAA  bool `json:"enableFXAA"`  // Software FXAA post-processing
}

// DefaultAdvancedRenderingConfig returns sensible defaults for all advanced rendering features
func DefaultAdvancedRenderingConfig() AdvancedRenderingConfig {
	return AdvancedRenderingConfig{
		// Core Lighting
		EnableAdvancedLighting: true,

		// Modern PBR Extensions - disabled by default for performance
		EnableClearcoat:    false,
		ClearcoatRoughness: 0.1,
		ClearcoatIntensity: 0.5,
		EnableSheen:        false,
		SheenColor:         mgl32.Vec3{1.0, 1.0, 1.0},
		SheenRoughness:     0.5,
		EnableTransmission: false,
		TransmissionFactor: 0.0,

		// Advanced Lighting Models
		EnableMultipleScattering: true,
		EnableEnergyConservation: true,
		EnableImageBasedLighting: false, // Requires IBL setup
		IBLIntensity:             1.0,

		// Perlin Noise - enabled for surface detail
		EnablePerlinNoise: true,
		NoiseScale:        0.0002,
		NoiseOctaves:      3,
		NoiseIntensity:    0.05,

		// Advanced Shadows - enabled for realism
		EnableAdvancedShadows: true,
		ShadowIntensity:       0.3,
		ShadowSoftness:        0.2,

		// Volumetric Lighting - disabled by default for performance
		EnableVolumetricLighting: false,
		VolumetricIntensity:      0.5,
		VolumetricSteps:          16,
		VolumetricScattering:     0.1,

		// Enhanced Subsurface Scattering
		EnableSubsurfaceScattering: true,
		ScatteringIntensity:        0.15,
		ScatteringDepth:            0.005,
		ScatteringColor:            mgl32.Vec3{1.0, 0.2, 0.1}, // Warm subsurface color

		// SSAO - enabled with hemisphere sampling and distance-based LOD for performance
		EnableSSAO:      true,
		SSAOIntensity:   0.35, // Reduced from 0.5 - was too dark
		SSAORadius:      200.0,
		SSAOBias:        0.015,
		SSAOSampleCount: 12,

		// Global Illumination - disabled by default for performance
		EnableGlobalIllumination: false,
		GIIntensity:              0.5,
		GIBounces:                2,

		// Bloom and HDR - enabled for realism
		EnableBloom:    true,
		BloomThreshold: 1.0,
		BloomIntensity: 0.3,
		BloomRadius:    0.4,

		// High-Quality Filtering - enabled for smooth results
		EnableHighQualityFiltering: true,
		FilteringQuality:           2,
		
		// Anti-Aliasing - good defaults
		MSAASamples: 4,    // 4x MSAA (hardware)
		EnableFXAA:  false, // FXAA disabled (use MSAA instead)
	}
}

// HighQualityRenderingConfig returns settings optimized for maximum visual quality
func HighQualityRenderingConfig() AdvancedRenderingConfig {
	config := DefaultAdvancedRenderingConfig()

	// Enable all advanced features for maximum quality
	config.EnableClearcoat = true
	config.EnableSheen = true
	config.EnableTransmission = true
	config.EnableImageBasedLighting = true
	config.EnableVolumetricLighting = true
	config.EnableGlobalIllumination = true
	config.EnableBloom = true

	// Higher quality settings
	config.ClearcoatIntensity = 0.8
	config.SheenRoughness = 0.3
	config.TransmissionFactor = 0.5
	config.VolumetricIntensity = 0.8
	config.VolumetricSteps = 32
	config.GIIntensity = 0.7
	config.GIBounces = 3
	config.BloomIntensity = 0.5
	config.FilteringQuality = 3
	config.SSAOSampleCount = 32
	
	// High quality AA
	config.MSAASamples = 8 // 8x MSAA for high quality
	config.EnableFXAA = false

	return config
}

// PerformanceRenderingConfig returns settings optimized for performance
func PerformanceRenderingConfig() AdvancedRenderingConfig {
	config := DefaultAdvancedRenderingConfig()

	// Disable expensive features
	config.EnableClearcoat = false
	config.EnableSheen = false
	config.EnableTransmission = false
	config.EnableImageBasedLighting = false
	config.EnableVolumetricLighting = false
	config.EnableGlobalIllumination = false
	config.EnableBloom = false
	config.EnableSubsurfaceScattering = false
	config.EnableSSAO = false

	// Lower quality settings for performance
	config.FilteringQuality = 1
	
	// Performance AA
	config.MSAASamples = 2 // 2x MSAA for performance
	config.EnableFXAA = false

	return config
}

// VoxelAdvancedRenderingConfig returns optimized settings for voxel rendering
func VoxelAdvancedRenderingConfig() AdvancedRenderingConfig {
	config := DefaultAdvancedRenderingConfig()

	// Enable voxel-specific features
	config.EnablePerlinNoise = true
	config.EnableAdvancedShadows = true
	config.EnableSSAO = true

	// Optimize for voxel scenes
	config.NoiseIntensity = 0.1
	config.ShadowIntensity = 0.4
	config.SSAOIntensity = 0.3
	
	// Balanced AA for voxels
	config.MSAASamples = 4
	config.EnableFXAA = false

	return config
}

// ApplyAdvancedRenderingConfig applies advanced rendering configuration to a model
func ApplyAdvancedRenderingConfig(model *Model, config AdvancedRenderingConfig) {
	if model.CustomUniforms == nil {
		model.CustomUniforms = make(map[string]interface{})
	}

	// Core lighting features
	model.CustomUniforms["enableAdvancedLighting"] = config.EnableAdvancedLighting

	// Modern PBR Extensions
	model.CustomUniforms["enableClearcoat"] = config.EnableClearcoat
	model.CustomUniforms["clearcoatRoughness"] = config.ClearcoatRoughness
	model.CustomUniforms["clearcoatIntensity"] = config.ClearcoatIntensity
	model.CustomUniforms["enableSheen"] = config.EnableSheen
	model.CustomUniforms["sheenColor"] = config.SheenColor
	model.CustomUniforms["sheenRoughness"] = config.SheenRoughness
	model.CustomUniforms["enableTransmission"] = config.EnableTransmission
	model.CustomUniforms["transmissionFactor"] = config.TransmissionFactor

	// Advanced Lighting Models
	model.CustomUniforms["enableMultipleScattering"] = config.EnableMultipleScattering
	model.CustomUniforms["enableEnergyConservation"] = config.EnableEnergyConservation
	model.CustomUniforms["enableImageBasedLighting"] = config.EnableImageBasedLighting
	model.CustomUniforms["iblIntensity"] = config.IBLIntensity

	// Apply noise settings
	model.CustomUniforms["enablePerlinNoise"] = config.EnablePerlinNoise
	model.CustomUniforms["noiseScale"] = config.NoiseScale
	model.CustomUniforms["noiseOctaves"] = int32(config.NoiseOctaves)
	model.CustomUniforms["noiseIntensity"] = config.NoiseIntensity

	// Apply shadow settings
	model.CustomUniforms["enableShadows"] = config.EnableAdvancedShadows
	model.CustomUniforms["shadowIntensity"] = config.ShadowIntensity
	model.CustomUniforms["shadowSoftness"] = config.ShadowSoftness

	// Volumetric Lighting
	model.CustomUniforms["enableVolumetricLighting"] = config.EnableVolumetricLighting
	model.CustomUniforms["volumetricIntensity"] = config.VolumetricIntensity
	model.CustomUniforms["volumetricSteps"] = int32(config.VolumetricSteps)
	model.CustomUniforms["volumetricScattering"] = config.VolumetricScattering

	// Enhanced subsurface scattering
	model.CustomUniforms["enableSubsurfaceScattering"] = config.EnableSubsurfaceScattering
	model.CustomUniforms["scatteringIntensity"] = config.ScatteringIntensity
	model.CustomUniforms["scatteringDepth"] = config.ScatteringDepth
	model.CustomUniforms["scatteringColor"] = config.ScatteringColor

	// SSAO settings
	model.CustomUniforms["enableSSAO"] = config.EnableSSAO
	model.CustomUniforms["ssaoIntensity"] = config.SSAOIntensity
	model.CustomUniforms["ssaoRadius"] = config.SSAORadius
	model.CustomUniforms["ssaoBias"] = config.SSAOBias
	model.CustomUniforms["ssaoSampleCount"] = int32(config.SSAOSampleCount)

	// Global Illumination
	model.CustomUniforms["enableGlobalIllumination"] = config.EnableGlobalIllumination
	model.CustomUniforms["giIntensity"] = config.GIIntensity
	model.CustomUniforms["giBounces"] = int32(config.GIBounces)

	// Bloom and HDR
	model.CustomUniforms["enableBloom"] = config.EnableBloom
	model.CustomUniforms["bloomThreshold"] = config.BloomThreshold
	model.CustomUniforms["bloomIntensity"] = config.BloomIntensity
	model.CustomUniforms["bloomRadius"] = config.BloomRadius

	// Apply filtering settings
	model.CustomUniforms["enableHighQualityFiltering"] = config.EnableHighQualityFiltering
	model.CustomUniforms["filteringQuality"] = int32(config.FilteringQuality)
}
