package renderer

import (
	"image"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

type LightType int

var FrustumCullingEnabled bool = false
var FaceCullingEnabled bool = false
var Debug bool = false
var DepthTestEnabled bool = true // New flag for depth testing
var ClearColorR float32 = 0.0    // Background clear color red
var ClearColorG float32 = 0.0    // Background clear color green
var ClearColorB float32 = 0.0    // Background clear color blue

const (
	STATIC_LIGHT LightType = iota
	DYNAMIC_LIGHT
)

type Light struct {
	// HOT DATA - Accessed every render call for lighting calculations
	Position        mgl32.Vec3 // Light position in world space
	Color           mgl32.Vec3 // Light color (RGB)
	Direction       mgl32.Vec3 // Direction for directional lights (normalized)
	Intensity       float32    // Light intensity multiplier
	AmbientStrength float32    // Configurable ambient lighting (0.0-1.0)
	Temperature     float32    // Color temperature in Kelvin (2000-10000, ~5500 is daylight)
	ConstantAtten   float32    // Constant attenuation factor (usually 1.0)
	LinearAtten     float32    // Linear attenuation factor
	QuadraticAtten  float32    // Quadratic attenuation factor
	
	// COLD DATA - Configuration, rarely changes during runtime
	Name       string    // Light name for editor identification
	Type       LightType // "static", "dynamic"
	Mode       string    // "directional", "point", "spot"
	Calculated bool      // Pre-calculation flag
}

type Render interface {
	Init(width, height int32, window *glfw.Window)
	Render(camera Camera, light *Light)
	AddModel(model *Model)
	RemoveModel(model *Model)
	LoadTexture(path string) (uint32, error)
	CreateTextureFromImage(img image.Image) (uint32, error)
	SetSkybox(skybox *Skybox)
	Cleanup()
	UpdateViewport(width, height int32)
}
