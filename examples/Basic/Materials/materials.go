package main

// This is a basic example of how to setup the engine and behaviour packages
import (
	behaviour "Gopher3D/internal/behaviour"
	"Gopher3D/internal/engine"
	loader "Gopher3D/internal/loader"
	"Gopher3D/internal/renderer"

	mgl "github.com/go-gl/mathgl/mgl32"
)

type GoCraftBehaviour struct {
	engine     *engine.Gopher
	name       string
	SceneModel *renderer.Model
}

func NewMaterialExampleBehaviour(engine *engine.Gopher) {
	gocraftBehaviour := &GoCraftBehaviour{engine: engine, name: "GoCraft"}
	behaviour.GlobalBehaviourManager.Add(gocraftBehaviour)
}

func main() {
	engine := engine.NewGopher(engine.OPENGL)

	NewMaterialExampleBehaviour(engine)
	engine.Render(768, 50)
}

func (mb *GoCraftBehaviour) Start() {
	// Professional studio lighting setup for material showcase
	mb.engine.Light = renderer.CreatePointLight(
		mgl.Vec3{400, 1000, 600},  // Elevated studio light position
		mgl.Vec3{1.0, 0.98, 0.95}, // Pure white with slight warm tint
		4.5, 1500.0,               // High intensity and range for dramatic lighting
	)
	mb.engine.Light.AmbientStrength = 0.1 // Low ambient for dramatic shadows on metallic surfaces
	mb.engine.Light.Temperature = 5400    // Professional studio lighting temperature

	// OpenGL and Vulkan use different coordinate systems
	mb.engine.Camera.InvertMouse = false
	mb.engine.Camera.Position = mgl.Vec3{500, 5000, 1000}
	mb.engine.Camera.Speed = 1000
	mb.engine.Light.Type = renderer.STATIC_LIGHT

	// For point lights, you can still set position (ignored for directional lights)
	if mb.engine.Light.Mode == "point" {
		mb.engine.Light.Position = mgl.Vec3{500, 5000, 1000}
	}

	mb.engine.SetDebugMode(true)
	createWorld(mb)
}

func (mb *GoCraftBehaviour) Update() {

}

func (mb *GoCraftBehaviour) UpdateFixed() {

}

func createWorld(mb *GoCraftBehaviour) {
	model, _ := loader.LoadObjectWithPath("../resources/obj/IronMan.obj", true)

	// Showcase premium Iron Man suit material - high-tech metallic finish
	model.SetPolishedMetal(0.8, 0.15, 0.1) // Enhanced red metal with slight gold tint
	model.SetExposure(1.5)                 // Higher exposure for dramatic metallic reflections

	// Add material name for identification
	model.Material.Name = "IronMan_Suit_Mark_VII"

	mb.SceneModel = model
	spawnBlock(mb.engine, mb.SceneModel, 0, 0)
}

func spawnBlock(engine *engine.Gopher, model *renderer.Model, x, z int) {
	model.SetPosition(0, 0, 0)
	model.Scale = mgl.Vec3{20.0, 20.0, 20.0}
	engine.AddModel(model)
}
