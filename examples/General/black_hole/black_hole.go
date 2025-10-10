package main

import (
	behaviour "Gopher3D/internal/behaviour"
	"Gopher3D/internal/engine"
	loader "Gopher3D/internal/loader"
	"Gopher3D/internal/renderer"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime/pprof"
	"time"

	mgl "github.com/go-gl/mathgl/mgl32"
)

func startCPUProfile() {
	f, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal("could not create CPU profile: ", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
}

func stopCPUProfile() {
	pprof.StopCPUProfile()
}

type Particle struct {
	position    mgl.Vec3
	previousPos mgl.Vec3
	color       string
	active      bool // To check if the particle is still active
}

type BlackHole struct {
	position mgl.Vec3
	mass     float32
	radius   float32 // Radius of the black hole's event horizon
}

type BlackHoleBehaviour struct {
	blackHoles    []*BlackHole
	redParticles  []*Particle
	blueParticles []*Particle
	redModel      *renderer.Model
	blueModel     *renderer.Model
	engine        *engine.Gopher
}

func NewBlackHoleBehaviour(engine *engine.Gopher) {
	bhb := &BlackHoleBehaviour{engine: engine}
	behaviour.GlobalBehaviourManager.Add(bhb)
}

func main() {
	engine := engine.NewGopher(engine.OPENGL)
	NewBlackHoleBehaviour(engine)

	engine.Width = 1980
	engine.Height = 1080

	engine.Render(0, 0)
}

func (bhb *BlackHoleBehaviour) Start() {

	bhb.engine.Camera.InvertMouse = false
	bhb.engine.Camera.Position = mgl.Vec3{200, 150, 1000}
	bhb.engine.Camera.Speed = 900

	// Fixed star lighting - no more beige
	bhb.engine.Light = renderer.CreatePointLight(
		mgl.Vec3{-1200, 600, 500}, // Same position as sun
		mgl.Vec3{1.0, 0.95, 0.8},  // Warm star color
		4.0, 2000.0,               // Lower intensity to prevent beige
	)
	bhb.engine.Light.AmbientStrength = 0.2 // Lower ambient
	bhb.engine.Light.Temperature = 5800    // Sun-like star temperature
	bhb.engine.Light.Type = renderer.STATIC_LIGHT

	// Create and add a black hole to the scene
	bhPosition := mgl.Vec3{0, 0, 0}
	bhMass := float32(10000)
	bhRadius := float32(50)
	blackHole := &BlackHole{position: bhPosition, mass: bhMass, radius: bhRadius}
	bhb.blackHoles = append(bhb.blackHoles, blackHole)

	// Num of instances for each color
	instances := 100000

	// Load the red particle model with instancing enabled
	redModel, err := loader.LoadObjectInstance("../../resources/obj/Sphere_Low.obj", true, instances)
	if err != nil {
		panic(fmt.Sprintf("Failed to load red sphere: %v", err))
	}
	redModel.Scale = mgl.Vec3{0.3, 0.3, 0.3} // Much smaller particles

	// Balanced plasma particles - no more beige
	redModel.SetPolishedMetal(1.0, 0.2, 0.0) // Metallic red
	redModel.SetExposure(1.2)                // Lower exposure to prevent beige
	bhb.redModel = redModel
	bhb.engine.AddModel(redModel)

	// Load the blue particle model with instancing enabled
	blueModel, err := loader.LoadObjectInstance("../../resources/obj/Sphere_Low.obj", true, instances)
	if err != nil {
		panic(fmt.Sprintf("Failed to load blue sphere: %v", err))
	}
	blueModel.Scale = mgl.Vec3{0.3, 0.3, 0.3} // Much smaller particles

	// Balanced plasma particles - no more beige
	blueModel.SetPolishedMetal(0.0, 0.4, 1.0) // Metallic blue
	blueModel.SetExposure(1.2)                // Lower exposure to prevent beige
	bhb.blueModel = blueModel
	bhb.engine.AddModel(blueModel)

	// NO SUN - just particles and lighting
	fmt.Printf("Black hole scene initialized - particles only!\n")

	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Initialize red particles
	for i := 0; i < instances; i++ {
		position := mgl.Vec3{
			rand.Float32()*500 - 100,
			rand.Float32()*500 - 100,
			rand.Float32()*500 - 100,
		}

		velocity := bhb.calculateTangentialVelocity(position, blackHole)

		particle := &Particle{
			position:    position,
			previousPos: position.Sub(velocity), // Initialize previous position for Verlet integration
			color:       "red",
			active:      true,
		}

		bhb.redParticles = append(bhb.redParticles, particle)
		bhb.redModel.SetInstancePosition(i, position)
	}

	// Initialize blue particles
	for i := 0; i < instances; i++ {
		position := mgl.Vec3{
			rand.Float32()*500 - 100,
			rand.Float32()*500 - 100,
			rand.Float32()*500 - 100,
		}

		velocity := bhb.calculateTangentialVelocity(position, blackHole)

		particle := &Particle{
			position:    position,
			previousPos: position.Sub(velocity), // Initialize previous position for Verlet integration
			color:       "blue",
			active:      true,
		}

		bhb.blueParticles = append(bhb.blueParticles, particle)
		bhb.blueModel.SetInstancePosition(i, position)
	}
}

func (bhb *BlackHoleBehaviour) calculateTangentialVelocity(position mgl.Vec3, blackHole *BlackHole) mgl.Vec3 {
	// Calculate the direction vector from the black hole to the particle
	direction := position.Sub(blackHole.position).Normalize()

	// Calculate a perpendicular vector for tangential velocity
	tangential := mgl.Vec3{-direction.Y(), direction.X(), 0}.Normalize()

	// Set the magnitude of the tangential velocity based on the distance
	distance := position.Sub(blackHole.position).Len()
	speed := float32(math.Sqrt(float64(blackHole.mass) / float64(distance)))

	// Reduce the speed to prevent particles from escaping
	return tangential.Mul(speed * 0.01)
}

func (bhb *BlackHoleBehaviour) Update() {
	for i := len(bhb.redParticles) - 1; i >= 0; i-- {
		p := bhb.redParticles[i]
		if !p.active {
			continue
		}

		for _, bh := range bhb.blackHoles {
			if bh.isWithinEventHorizon(p) {
				p.active = false
				bhb.redModel.RemoveModelInstance(i)
				bhb.redParticles = append(bhb.redParticles[:i], bhb.redParticles[i+1:]...)
				continue
			}
			bh.ApplyGravity(p)
		}

		newPosition := p.position.Mul(2).Sub(p.previousPos)
		p.previousPos = p.position
		p.position = newPosition
		bhb.redModel.SetInstancePosition(i, p.position)
	}

	for i := len(bhb.blueParticles) - 1; i >= 0; i-- {
		p := bhb.blueParticles[i]
		if !p.active {
			continue
		}

		for _, bh := range bhb.blackHoles {
			if bh.isWithinEventHorizon(p) {
				p.active = false
				bhb.blueModel.RemoveModelInstance(i)
				bhb.blueParticles = append(bhb.blueParticles[:i], bhb.blueParticles[i+1:]...)
				continue
			}
			bh.ApplyGravity(p)
		}

		newPosition := p.position.Mul(2).Sub(p.previousPos)
		p.previousPos = p.position
		p.position = newPosition
		bhb.blueModel.SetInstancePosition(i, p.position)
	}
}

func (bhb *BlackHoleBehaviour) UpdateFixed() {

}

func (bh *BlackHole) isWithinEventHorizon(p *Particle) bool {
	distance := p.position.Sub(bh.position).Len()
	return distance < bh.radius
}

func (bh *BlackHole) ApplyGravity(p *Particle) {
	direction := bh.position.Sub(p.position)
	distance := direction.Len()

	if distance == 0 {
		return
	}

	direction = direction.Normalize()

	gravity := (bh.mass * 0.0005) / (distance * distance)
	force := direction.Mul(gravity)

	maxForce := float32(2.0)
	if force.Len() > maxForce {
		force = force.Normalize().Mul(maxForce)
	}

	p.position = p.position.Add(force)
}
