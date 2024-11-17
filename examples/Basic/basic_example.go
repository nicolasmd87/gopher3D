package main

// This is a basic example of how to setup the engine and behaviour packages
import (
	behaviour "Gopher3D/internal/Behaviour"
	loader "Gopher3D/internal/Loader"
	"Gopher3D/internal/engine"
	"Gopher3D/internal/renderer"

	mgl "github.com/go-gl/mathgl/mgl32"
)

type TestBehaviour struct {
	engine *engine.Gopher
	name   string
	model  *renderer.Model
}

func NewTestBehaviour(engine *engine.Gopher) {
	mb := &TestBehaviour{engine: engine, name: "Test"}
	behaviour.GlobalBehaviourManager.Add(mb)
}
func main() {
	engine := engine.NewGopher(engine.OPENGL) // or engine.VULKAN
	NewTestBehaviour(engine)
	engine.Width = 720
	engine.Height = 480

	// WINDOW POS IN X,Y AND MODEL
	engine.Render(400, 400)
}
func (mb *TestBehaviour) Start() {
	mb.engine.Light = renderer.CreateLight()
	//Static light for some extra FPS
	mb.engine.Light.Type = renderer.STATIC_LIGHT
	mb.engine.Light.Position = mgl.Vec3{0, 1000, 0}
	mb.engine.SetFrustumCulling(false)
	mb.engine.SetFaceCulling(false)
	// Invert mouse for OpenGL
	mb.engine.Camera.InvertMouse = false
	m, err := loader.LoadObjectWithPath("../resources/obj/Cube.obj", true)
	if err != nil {
		panic(err)
	}
	// I want to increase the scale of the model
	m.Scale = mgl.Vec3{10.0, 10.0, 10.0}
	mb.model = m
	mb.engine.AddModel(m)
}

func (mb *TestBehaviour) Update() {
	mb.model.SetPosition(mb.model.X(), mb.model.Y()+0.005, mb.model.Z())
	mb.model.Rotate(0.5, 0, 0)
}

func (mb *TestBehaviour) UpdateFixed() {
	mb.model.SetPosition(mb.model.X(), mb.model.Y()+0.005, mb.model.Z())
	mb.model.Rotate(0.5, 0, 0)
}
