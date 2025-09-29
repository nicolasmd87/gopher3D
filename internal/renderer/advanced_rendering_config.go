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
	EnableAdvancedShadows    bool    `json:"enableAdvancedShadows"`
	ShadowIntensity          float32 `json:"shadowIntensity"`
	ShadowSoftness           float32 `json:"shadowSoftness"`
	EnableCascadedShadowMaps bool    `json:"enableCascadedShadowMaps"`
	EnablePCFShadows         bool    `json:"enablePCFShadows"`
	PCFKernelSize            int     `json:"pcfKernelSize"`

	// Volumetric Lighting
	EnableVolumetricLighting bool    `json:"enableVolumetricLighting"`
	VolumetricIntensity      float32 `json:"volumetricIntensity"`
	VolumetricSteps          int     `json:"volumetricSteps"`
	VolumetricScattering     float32 `json:"volumetricScattering"`

	// Level of Detail and Performance
	EnableLOD             bool    `json:"enableLOD"`
	LODTransitionDistance float32 `json:"lodTransitionDistance"`
	PerformanceScaling    float32 `json:"performanceScaling"`

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

	// Temporal Anti-Aliasing
	EnableTAA      bool    `json:"enableTAA"`
	TAABlendFactor float32 `json:"taaBlendFactor"`

	// High-Quality Filtering
	EnableHighQualityFiltering bool `json:"enableHighQualityFiltering"`
	FilteringQuality           int  `json:"filteringQuality"`
	AntiAliasing               bool `json:"antiAliasing"`

	// Mesh Quality and Tessellation Controls
	EnableMeshSmoothing    bool    `json:"enableMeshSmoothing"`
	MeshSmoothingIntensity float32 `json:"meshSmoothingIntensity"`
	TessellationQuality    int     `json:"tessellationQuality"`
	NormalSmoothingRadius  float32 `json:"normalSmoothingRadius"`
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
		EnableAdvancedShadows:    true,
		ShadowIntensity:          0.3,
		ShadowSoftness:           0.2,
		EnableCascadedShadowMaps: false, // Requires shadow map setup
		EnablePCFShadows:         true,
		PCFKernelSize:            3,

		// Volumetric Lighting - disabled by default for performance
		EnableVolumetricLighting: false,
		VolumetricIntensity:      0.5,
		VolumetricSteps:          16,
		VolumetricScattering:     0.1,

		// LOD/Visibility - enabled for performance
		EnableLOD:             true,
		LODTransitionDistance: 50000.0,
		PerformanceScaling:    0.3,

		// Enhanced Subsurface Scattering
		EnableSubsurfaceScattering: true,
		ScatteringIntensity:        0.15,
		ScatteringDepth:            0.005,
		ScatteringColor:            mgl32.Vec3{1.0, 0.2, 0.1}, // Warm subsurface color

		// SSAO - enabled for depth perception
		EnableSSAO:      true,
		SSAOIntensity:   0.25,
		SSAORadius:      150.0,
		SSAOBias:        0.025,
		SSAOSampleCount: 16,

		// Global Illumination - disabled by default for performance
		EnableGlobalIllumination: false,
		GIIntensity:              0.5,
		GIBounces:                2,

		// Bloom and HDR - enabled for realism
		EnableBloom:    true,
		BloomThreshold: 1.0,
		BloomIntensity: 0.3,
		BloomRadius:    0.4,

		// TAA - disabled by default (requires temporal data)
		EnableTAA:      false,
		TAABlendFactor: 0.1,

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
	config.EnableTAA = true

	// Higher quality settings
	config.ClearcoatIntensity = 0.8
	config.SheenRoughness = 0.3
	config.TransmissionFactor = 0.5
	config.VolumetricIntensity = 0.8
	config.VolumetricSteps = 32
	config.GIIntensity = 0.7
	config.GIBounces = 3
	config.BloomIntensity = 0.5
	config.FilteringQuality = 3         // Maximum quality
	config.MeshSmoothingIntensity = 0.9 // Maximum smoothing
	config.TessellationQuality = 4      // Maximum quality
	config.SSAOSampleCount = 32         // Higher quality SSAO

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
	config.EnableTAA = false
	config.EnableSubsurfaceScattering = false
	config.EnableSSAO = false

	// Lower quality settings for performance
	config.FilteringQuality = 1
	config.MeshSmoothingIntensity = 0.3
	config.TessellationQuality = 1
	config.PerformanceScaling = 0.5
	config.PCFKernelSize = 1

	return config
}

// VoxelAdvancedRenderingConfig returns optimized settings for voxel rendering
func VoxelAdvancedRenderingConfig() AdvancedRenderingConfig {
	config := DefaultAdvancedRenderingConfig()

	// Enable voxel-specific features
	config.EnablePerlinNoise = true
	config.EnableAdvancedShadows = true
	config.EnableSSAO = true
	config.EnableLOD = true
	config.EnableMeshSmoothing = false // Voxels should stay crisp

	// Optimize for voxel scenes
	config.NoiseIntensity = 0.1
	config.ShadowIntensity = 0.4
	config.SSAOIntensity = 0.3
	config.PerformanceScaling = 0.5
	config.TessellationQuality = 1 // Low for voxels

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
	model.CustomUniforms["enablePCFShadows"] = config.EnablePCFShadows
	model.CustomUniforms["pcfKernelSize"] = int32(config.PCFKernelSize)

	// Volumetric Lighting
	model.CustomUniforms["enableVolumetricLighting"] = config.EnableVolumetricLighting
	model.CustomUniforms["volumetricIntensity"] = config.VolumetricIntensity
	model.CustomUniforms["volumetricSteps"] = int32(config.VolumetricSteps)
	model.CustomUniforms["volumetricScattering"] = config.VolumetricScattering

	// Apply LOD settings
	model.CustomUniforms["enableLOD"] = config.EnableLOD
	model.CustomUniforms["lodTransitionDistance"] = config.LODTransitionDistance
	model.CustomUniforms["performanceScaling"] = config.PerformanceScaling

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

	// TAA
	model.CustomUniforms["enableTAA"] = config.EnableTAA
	model.CustomUniforms["taaBlendFactor"] = config.TAABlendFactor

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
