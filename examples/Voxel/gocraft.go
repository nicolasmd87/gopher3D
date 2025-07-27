package main

import (
	behaviour "Gopher3D/internal/behaviour"
	loader "Gopher3D/internal/loader"
	"Gopher3D/internal/engine"
	"Gopher3D/internal/renderer"
	"math/rand"
	"time"

	perlin "github.com/aquilax/go-perlin" // Example Perlin noise library
	"github.com/go-gl/mathgl/mgl32"
)

var p = perlin.NewPerlin(2, 2, 3, rand.New(rand.NewSource(time.Now().UnixNano())).Int63())

type GoCraftBehaviour struct {
	engine          *engine.Gopher
	name            string
	worldHeight     int
	worldWidth      int
	noiseDistortion float64
	cubeModel       *renderer.Model // Instanced model
}

func NewGocraftBehaviour(engine *engine.Gopher) {
	gocraftBehaviour := &GoCraftBehaviour{engine: engine, name: "GoCraft"}
	behaviour.GlobalBehaviourManager.Add(gocraftBehaviour)
}

func main() {
	engine := engine.NewGopher(engine.OPENGL) // or engine.VULKAN

	NewGocraftBehaviour(engine)

	engine.Width = 1024
	engine.Height = 768

	// WINDOW POS IN X,Y AND MODEL
	engine.Render(600, 200)
}

func (mb *GoCraftBehaviour) Start() {
	mb.engine.Light = renderer.CreateLight()
	mb.engine.Light.Type = renderer.STATIC_LIGHT
	mb.engine.Light.Position = mgl32.Vec3{1000, 1000, 0}
	mb.engine.Camera.InvertMouse = false
	mb.engine.Camera.Position = mgl32.Vec3{500, 50, 500}
	mb.engine.Camera.Speed = 200

	// Adjust world size and noise
	mb.worldHeight = 1500
	mb.worldWidth = 1500
	mb.noiseDistortion = 22

	// Enable face culling for performance
	mb.engine.SetFaceCulling(true)

	// Load the cube model with instancing enabled
	model, err := loader.LoadObjectInstance("../resources/obj/Cube.obj", false, mb.worldHeight*mb.worldWidth)
	if err != nil {
		panic(err)
	}
	model.SetTexture("../resources/textures/Grass.png")
	model.Scale = mgl32.Vec3{1, 1, 1}
	mb.cubeModel = model

	// Add the model to the engine
	mb.engine.AddModel(model)

	// Create the world using instancing
	createWorld(mb)
}

func (mb *GoCraftBehaviour) Update() {
	// Update logic for the world (if needed)
}

func (mb *GoCraftBehaviour) UpdateFixed() {
	// No fixed update required for this example
}

// Create the world using instanced rendering with spacing to prevent overlap
func createWorld(mb *GoCraftBehaviour) {
	var index int
	spacing := 2.0 // Adjust this value to add more spacing between cubes
	for x := 0; x < mb.worldHeight; x++ {
		for z := 0; z < mb.worldWidth; z++ {
			// Generate noise for the Y-axis (height)
			y := p.Noise2D(float64(x)*0.1, float64(z)*0.1)
			y = scaleNoise(mb, y)

			// Set the instance position for each cube with spacing applied
			mb.cubeModel.SetInstancePosition(index, mgl32.Vec3{float32(x) * float32(spacing), float32(y), float32(z) * float32(spacing)})
			index++
		}
	}

	// Update the instance count in the model
	mb.cubeModel.InstanceCount = mb.worldHeight * mb.worldWidth
}

func scaleNoise(mb *GoCraftBehaviour, noiseVal float64) float64 {
	// Scale and adjust the noise value to suit the height range of your terrain
	return (noiseVal / 2) * mb.noiseDistortion
}
