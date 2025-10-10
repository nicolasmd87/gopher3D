package main

import (
	behaviour "Gopher3D/internal/behaviour"
	"Gopher3D/internal/engine"
	loader "Gopher3D/internal/loader"
	"Gopher3D/internal/renderer"
	"fmt"

	mgl "github.com/go-gl/mathgl/mgl32"
)

type MaterialsShowcase struct {
	engine *engine.Gopher
	f104   *renderer.Model
}

func NewMaterialsShowcase(engine *engine.Gopher) {
	ms := &MaterialsShowcase{engine: engine}
	behaviour.GlobalBehaviourManager.Add(ms)
}

func main() {

	engine := engine.NewGopher(engine.OPENGL)
	NewMaterialsShowcase(engine)

	engine.Width = 1600
	engine.Height = 900

	engine.Render(100, 100)
}

func (ms *MaterialsShowcase) Start() {
	// Setup camera for optimal viewing
	ms.engine.Camera.InvertMouse = false
	ms.engine.Camera.Position = mgl.Vec3{0, 200, 800} // Far back to see the giant plane
	ms.engine.Camera.Speed = 500



	// Add VERY BRIGHT directional light with high ambient
	ms.engine.Light = renderer.CreateSunlight(mgl.Vec3{0.4, 0.8, 0.3})
	ms.engine.Light.Intensity = 5.0       // Much brighter
	ms.engine.Light.AmbientStrength = 0.8 // Very high ambient so we can see it
	ms.engine.Light.Color = mgl.Vec3{1.0, 0.95, 0.9}

	fmt.Println("Loading F-104 Starfighter model...")

	// Load F-104 Starfighter with multi-materials
	f104, err := loader.LoadObjectWithPath("examples/resources/obj/f104starfighter.obj", false)
	if err != nil {
		fmt.Printf("Error loading F-104: %v\n", err)
		fmt.Println("Make sure f104starfighter.obj exists in examples/resources/obj/")
		return
	}

	// Position and scale the aircraft - MUCH BIGGER
	f104.SetPosition(0, 0, 0)
	f104.SetScale(100, 100, 100) // 10x bigger scale

	ms.f104 = f104
	ms.engine.AddModel(f104)


}

func (ms *MaterialsShowcase) Update() {

}

func (ms *MaterialsShowcase) UpdateFixed() {
}
