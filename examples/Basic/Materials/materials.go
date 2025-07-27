package main

// This is a basic example of how to setup the engine and behaviour packages
import (
	behaviour "Gopher3D/internal/behaviour"
	loader "Gopher3D/internal/loader"
	"Gopher3D/internal/engine"
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
	mb.engine.Light = renderer.CreateLight()
	// OpenGL and Vulkan use different coordinate systems
	mb.engine.Camera.InvertMouse = false
	mb.engine.Camera.Position = mgl.Vec3{500, 5000, 1000}
	mb.engine.Camera.Speed = 1000
	mb.engine.Light.Type = renderer.STATIC_LIGHT
	mb.engine.Light.Position = mgl.Vec3{500, 5000, 1000}
	mb.engine.SetDebugMode(true)
	createWorld(mb)
}

func (mb *GoCraftBehaviour) Update() {

}

func (mb *GoCraftBehaviour) UpdateFixed() {

}

func createWorld(mb *GoCraftBehaviour) {
	model, _ := loader.LoadObjectWithPath("../resources/obj/IronMan.obj", true)
	mb.SceneModel = model
	spawnBlock(mb.engine, mb.SceneModel, 0, 0)
}

func spawnBlock(engine *engine.Gopher, model *renderer.Model, x, z int) {
	model.SetPosition(0, 0, 0)
	model.Scale = mgl.Vec3{20.0, 20.0, 20.0}
	engine.AddModel(model)
}
