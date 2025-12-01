package renderer

import (
	"testing"
)

func TestDefaultAdvancedRenderingConfig(t *testing.T) {
	config := DefaultAdvancedRenderingConfig()

	if config.SSAORadius <= 0 {
		t.Error("Default SSAO radius should be positive")
	}
}

func TestAdvancedRenderingConfigFields(t *testing.T) {
	config := AdvancedRenderingConfig{
		EnableSSAO:                 true,
		EnableGlobalIllumination:   true,
		EnableSubsurfaceScattering: true,
		SSAORadius:                 1.0,
		SSAOIntensity:              1.0,
	}

	if !config.EnableSSAO {
		t.Error("SSAO should be enabled")
	}

	if !config.EnableGlobalIllumination {
		t.Error("GI should be enabled")
	}

	if !config.EnableSubsurfaceScattering {
		t.Error("SSS should be enabled")
	}
}

func TestWaterPhotorealisticConfig(t *testing.T) {
	config := WaterPhotorealisticConfig()

	if config.CausticsIntensity <= 0 {
		t.Error("Water config should have positive caustics intensity")
	}

	if config.CausticsScale <= 0 {
		t.Error("Water config should have positive caustics scale")
	}
}
