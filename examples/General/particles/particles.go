package main

import (
	behaviour "Gopher3D/internal/behaviour"
	"Gopher3D/internal/engine"
	loader "Gopher3D/internal/loader"
	"Gopher3D/internal/renderer"
	"math"
	"math/rand"
	"time"

	mgl "github.com/go-gl/mathgl/mgl32"
)

type Particle struct {
	position mgl.Vec3
	velocity mgl.Vec3
	model    *renderer.Model
	index    int // Store the instance index for updating the correct instance position
}

type ParticleBehaviour struct {
	engine      *engine.Gopher
	particles   []*Particle
	redModel    *renderer.Model
	blueModel   *renderer.Model
	greenModel  *renderer.Model
	yellowModel *renderer.Model
	purpleModel *renderer.Model
}

func NewParticleBehaviour(engine *engine.Gopher) {
	mb := &ParticleBehaviour{engine: engine}
	behaviour.GlobalBehaviourManager.Add(mb)
}

func main() {
	engine := engine.NewGopher(engine.OPENGL) // or engine.VULKAN
	NewParticleBehaviour(engine)
	engine.Width = 1024
	engine.Height = 768

	// WINDOW POS IN X,Y AND MODEL
	engine.Render(600, 200)
}

func (pb *ParticleBehaviour) Start() {
	pb.engine.Camera.InvertMouse = false
	pb.engine.Camera.Position = mgl.Vec3{0, 50, 1000}
	pb.engine.Camera.Speed = 400

	// Dynamic studio lighting for vibrant particle showcase
	pb.engine.Light = renderer.CreatePointLight(
		mgl.Vec3{0, 200, 300},    // Elevated position for dramatic lighting
		mgl.Vec3{1.0, 0.95, 0.9}, // Warm white with slight golden tint
		3.5, 800.0,               // Higher intensity and range for particle field
	)
	pb.engine.Light.AmbientStrength = 0.15 // Moderate ambient to see all particles
	pb.engine.Light.Type = renderer.STATIC_LIGHT

	pb.engine.SetFrustumCulling(false)
	pb.engine.SetFaceCulling(true)

	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Create instanced models for each color
	numInstances := 300
	pb.redModel = createModelInstance(pb, "red", numInstances)
	pb.blueModel = createModelInstance(pb, "blue", numInstances)
	pb.greenModel = createModelInstance(pb, "green", numInstances)
	pb.yellowModel = createModelInstance(pb, "yellow", numInstances)
	pb.purpleModel = createModelInstance(pb, "purple", numInstances)

	// Create and initialize particles
	initializeParticles(pb, pb.redModel, numInstances)
	initializeParticles(pb, pb.blueModel, numInstances)
	initializeParticles(pb, pb.greenModel, numInstances)
	initializeParticles(pb, pb.yellowModel, numInstances)
	initializeParticles(pb, pb.purpleModel, numInstances)
}

func createModelInstance(pb *ParticleBehaviour, color string, numInstances int) *renderer.Model {
	// Load the model for the given color with instancing enabled
	model, err := loader.LoadObjectInstance("../../resources/obj/Sphere_Low.obj", true, numInstances)
	if err != nil {
		panic(err)
	}
	model.Scale = mgl.Vec3{5, 5, 5}

	// Set vibrant PBR materials for each particle type with appropriate surface properties
	switch color {
	case "red":
		model.SetGlossy(0.9, 0.1, 0.1) // Glossy red plastic-like finish
		model.SetExposure(1.3)         // Higher exposure for vibrant red
	case "yellow":
		model.SetPolishedMetal(1.0, 0.8, 0.1) // Metallic gold appearance
		model.SetExposure(1.4)                // High exposure for golden shine
	case "green":
		model.SetMatte(0.1, 0.8, 0.2) // Matte green like jade
		model.SetExposure(1.1)        // Natural exposure
	case "blue":
		model.SetGlossy(0.1, 0.3, 0.9) // Glossy blue ceramic
		model.SetExposure(1.2)         // Enhanced blue vibrancy
	case "purple":
		model.SetRoughMetal(0.6, 0.2, 0.8) // Rough metallic purple
		model.SetExposure(1.25)            // Balanced exposure
	}
	model.Material.Name = "Particle_" + color
	pb.engine.AddModel(model)
	return model
}

func initializeParticles(pb *ParticleBehaviour, model *renderer.Model, numInstances int) {
	// Initialize particles for the given model
	for i := 0; i < numInstances; i++ {
		position := mgl.Vec3{
			rand.Float32()*100 - 50, // Random X between -50 and 50
			rand.Float32()*100 - 50, // Random Y between -50 and 50
			rand.Float32()*100 - 50, // Random Z between -50 and 50
		}

		particle := &Particle{
			position: position,
			velocity: mgl.Vec3{0, 0, 0}, // Set initial velocity to zero
			model:    model,
			index:    i, // Store the instance index for this particle
		}

		pb.particles = append(pb.particles, particle)
		model.SetInstancePosition(i, position) // Set the instance position for instanced rendering
	}
}

func (pb *ParticleBehaviour) Update() {
	UpdateParticles(pb)
}

func (pb *ParticleBehaviour) UpdateFixed() {
}

func UpdateParticles(pb *ParticleBehaviour) {
	for _, p := range pb.particles {
		pb.applyForces(p)
		// Apply damping to the velocity to prevent particles from flying apart
		p.velocity = p.velocity.Mul(0.99)
		p.position = p.position.Add(p.velocity)

		// Update the instance position in the renderer using the correct model and index
		p.model.SetInstancePosition(p.index, p.position)
	}
}

func (pb *ParticleBehaviour) applyForces(p *Particle) {
	for _, other := range pb.particles {
		if p == other {
			continue
		}
		force := pb.calculateForce(p, other)
		p.velocity = p.velocity.Add(force)
	}
}

func fastInverseSqrt(x float32) float32 {
	i := math.Float32bits(x)
	i = 0x5f3759df - (i >> 1)
	y := math.Float32frombits(i)
	y = y * (1.5 - (0.5 * x * y * y)) // Perform one iteration of Newton's method
	return y
}

func (pb *ParticleBehaviour) calculateForce(p1, p2 *Particle) mgl.Vec3 {
	direction := p2.position.Sub(p1.position)
	distance := direction.Len()

	if distance == 0 {
		return mgl.Vec3{0, 0, 0}
	}

	direction = direction.Normalize()
	magnitude := float32(0.01)

	// Introduce a softening parameter to avoid infinite forces
	softening := float32(0.1)
	invSqrtDistance := fastInverseSqrt(distance*distance + softening*softening)
	forceMagnitude := magnitude * invSqrtDistance * invSqrtDistance

	// Clamp the force to a maximum value to prevent excessive acceleration
	maxForce := float32(10.0)
	if forceMagnitude > maxForce {
		forceMagnitude = maxForce
	} else if forceMagnitude < -maxForce {
		forceMagnitude = -maxForce
	}

	force := direction.Mul(forceMagnitude)
	return force
}
