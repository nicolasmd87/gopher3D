package main

import (
	behaviour "Gopher3D/internal/behaviour"
	"Gopher3D/internal/engine"
	loader "Gopher3D/internal/loader"
	"Gopher3D/internal/renderer"
	"fmt"
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
	// FINAL FIX: Ultra-high dramatic lighting with fixed normals!
	mb.engine.Light = renderer.CreatePointLight(
		mgl32.Vec3{200, 5000, 200},
		mgl32.Vec3{1.0, 0.95, 0.8},
		3.5,
		8000.0,
	)
	mb.engine.Light.AmbientStrength = 0.15 // Lower ambient for more dramatic shadows
	mb.engine.Light.Type = renderer.STATIC_LIGHT

	mb.engine.Camera.InvertMouse = false
	mb.engine.Camera.Position = mgl32.Vec3{100, 120, 100}
	mb.engine.Camera.Speed = 200

	// Enhanced world generation for beautiful varied terrain
	mb.worldHeight = 400
	mb.worldWidth = 400
	mb.noiseDistortion = 20

	// Enable face culling and depth testing for better performance and z-fighting reduction
	mb.engine.SetFaceCulling(true)

	// Set skybox color for background
	renderer.SetSkyboxColor(0.5, 0.7, 1.0) // Bright day sky blue
	err := mb.engine.SetSkybox("dark_sky") // Create skybox with that color
	if err != nil {
		fmt.Printf("Could not set skybox: %v\n", err)
	} else {
		fmt.Println("Voxel skybox created with bright day sky color")
	}

	// Load the cube model with instancing enabled - FORCE normal recalculation to fix lighting
	model, err := loader.LoadObjectInstance("../resources/obj/Cube.obj", true, mb.worldHeight*mb.worldWidth)
	if err != nil {
		panic(err)
	}
	// Apply texture - SetTexture handles errors internally with logging
	model.SetTexture("../resources/textures/Grass.png")
	model.Scale = mgl32.Vec3{0.5, 0.5, 0.5}

	// Beautiful enhanced grass material properties
	model.SetMatte(0.3, 0.6, 0.15) // Richer, more vibrant grass green
	model.SetExposure(1.1)         // Slightly higher exposure for prettier lighting
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

// Create the beautiful world using instanced rendering with perfect anti-z-fighting spacing
func createWorld(mb *GoCraftBehaviour) {
	var index int
	spacing := 0.5

	for x := 0; x < mb.worldHeight; x++ {
		for z := 0; z < mb.worldWidth; z++ {
			// Enhanced multi-octave noise for realistic terrain
			baseY := p.Noise2D(float64(x)*0.05, float64(z)*0.05)   // Large features
			detailY := p.Noise2D(float64(x)*0.15, float64(z)*0.15) // Medium details
			fineY := p.Noise2D(float64(x)*0.3, float64(z)*0.3)     // Fine details

			// Combine different noise scales for realistic terrain
			combinedY := baseY*0.6 + detailY*0.3 + fineY*0.1
			y := scaleNoise(mb, combinedY)

			mb.cubeModel.SetInstancePosition(index, mgl32.Vec3{
				float32(x) * float32(spacing),
				float32(y),
				float32(z) * float32(spacing),
			})
			index++
		}
	}

	// Update the instance count in the model
	mb.cubeModel.InstanceCount = mb.worldHeight * mb.worldWidth
}

func scaleNoise(mb *GoCraftBehaviour, noiseVal float64) float64 {
	scaledNoise := noiseVal * mb.noiseDistortion

	if scaledNoise < -5 {
		scaledNoise = -5
	}
	if scaledNoise > 25 {
		scaledNoise = 25
	}

	return scaledNoise
}
