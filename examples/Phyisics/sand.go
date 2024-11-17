package main

import (
	behaviour "Gopher3D/internal/Behaviour"
	loader "Gopher3D/internal/Loader"
	"Gopher3D/internal/engine"
	"Gopher3D/internal/renderer"
	"math/rand"
	"sync"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"
	mgl "github.com/go-gl/mathgl/mgl32"
)

const (
	Gravity           float32 = -9.81
	Friction          float32 = 0.95
	MaxExplosionForce float32 = 15.0
	RadiusOfEffect    float32 = 80.0
	AttractionForce   float32 = 60.0
	RadiusOfInfluence float32 = 80.0
	ParticleBatchSize int     = 16
	TimeStep          float32 = 0.016
)

type Particle struct {
	position mgl.Vec3
	velocity mgl.Vec3
	active   bool
	grabbed  bool
}

type SandSimulation struct {
	sandParticles   []*Particle
	sandModel       *renderer.Model
	engine          *engine.Gopher
	explosionOrigin mgl.Vec3
	mousePressed    bool
}

func NewSandSimulation(engine *engine.Gopher) {
	ss := &SandSimulation{engine: engine}
	behaviour.GlobalBehaviourManager.Add(ss)
}

func main() {
	engine := engine.NewGopher(engine.OPENGL)
	NewSandSimulation(engine)

	engine.Width = 1980
	engine.Height = 1080

	engine.Render(0, 0)
}

func (ss *SandSimulation) Start() {
	ss.engine.Camera.InvertMouse = false
	ss.engine.Camera.Position = mgl.Vec3{0, 100, 300}
	ss.engine.Camera.Speed = 200
	ss.engine.Light = renderer.CreateLight()
	ss.engine.Light.Type = renderer.STATIC_LIGHT
	ss.engine.Light.Intensity = 0.03
	ss.engine.Light.Position = mgl.Vec3{0, 2000, -1200}

	instances := 1200000
	sandModel, err := loader.LoadObjectInstance("../resources/obj/Sphere_Low.obj", true, instances)
	if err != nil {
		panic(err)
	}
	sandModel.Scale = mgl.Vec3{0.3, 0.3, 0.3}
	sandModel.SetDiffuseColor(139, 69, 19)
	ss.sandModel = sandModel
	ss.engine.AddModel(sandModel)

	rand.Seed(time.Now().UnixNano())

	ss.sandParticles = make([]*Particle, instances)
	for i := 0; i < instances; i++ {
		position := mgl.Vec3{
			rand.Float32()*100 - 50, // Spread particles along X-axis
			0,                       // Y-coordinate set to ground level
			rand.Float32()*100 - 50, // Spread particles along Z-axis
		}

		particle := &Particle{
			position: position,
			velocity: mgl.Vec3{0, 0, 0},
			active:   true,
		}

		ss.sandParticles[i] = particle
		ss.sandModel.SetInstancePosition(i, position)
	}
}

func (ss *SandSimulation) Update() {
	dt := TimeStep

	mousePos := ss.engine.GetMousePosition()
	mousePressed := ss.engine.IsMouseButtonPressed(glfw.MouseButtonLeft)
	mouseWorldPos := ss.engine.Camera.ScreenToWorld(mousePos, int(ss.engine.Width), int(ss.engine.Height))

	if ss.mousePressed && !mousePressed {
		ss.explosionOrigin = mouseWorldPos
		ss.StopParticlesAfterRelease()
	}

	ss.mousePressed = mousePressed

	if mousePressed {
		ss.ApplyForcesToParticles(mouseWorldPos, AttractionForce, dt)
	}

	ss.UpdateParticles(dt, Friction)
}

func (ss *SandSimulation) ApplyForcesToParticles(mouseWorldPos mgl.Vec3, attractionForce, dt float32) {
	numWorkers := ParticleBatchSize
	batchSize := (len(ss.sandParticles) + numWorkers - 1) / numWorkers

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		start := w * batchSize
		end := start + batchSize
		if end > len(ss.sandParticles) {
			end = len(ss.sandParticles)
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				p := ss.sandParticles[i]
				if !p.active {
					continue
				}

				distanceToMouse := p.position.Sub(mouseWorldPos).Len()
				if distanceToMouse < RadiusOfInfluence {
					p.grabbed = true

					forceMultiplier := 1 - (distanceToMouse / RadiusOfInfluence)
					direction := mouseWorldPos.Sub(p.position).Normalize()
					forceToMouse := direction.Mul(attractionForce * dt * forceMultiplier)

					p.velocity = p.velocity.Mul(0.9).Add(forceToMouse)
				}
			}
		}(start, end)
	}
	wg.Wait()
}

func (ss *SandSimulation) StopParticlesAfterRelease() {
	for i := 0; i < len(ss.sandParticles); i++ {
		p := ss.sandParticles[i]
		if p.grabbed {
			direction := p.position.Sub(ss.explosionOrigin)
			distance := direction.Len()

			if distance > 0 && distance < RadiusOfEffect {
				direction = direction.Normalize()
				forceMultiplier := 1 - (distance / RadiusOfEffect)
				explosionForce := MaxExplosionForce * forceMultiplier

				p.velocity = p.velocity.Add(direction.Mul(explosionForce))
			}

			p.grabbed = false
		}
	}
}

func (ss *SandSimulation) UpdateParticles(dt, friction float32) {
	numWorkers := ParticleBatchSize
	batchSize := (len(ss.sandParticles) + numWorkers - 1) / numWorkers
	gravityEffect := Gravity * dt

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		start := w * batchSize
		end := start + batchSize
		if end > len(ss.sandParticles) {
			end = len(ss.sandParticles)
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				p := ss.sandParticles[i]
				if !p.active {
					continue
				}

				// Apply gravity
				p.velocity[1] += gravityEffect
				// Update position
				p.position = p.position.Add(p.velocity.Mul(dt))

				// Collision with floor at y = 0
				if p.position.Y() <= 0 {
					p.position[1] = 0
					p.velocity[1] = 0         // Prevent bouncing
					p.velocity[0] *= friction // Apply friction to X axis
					p.velocity[2] *= friction // Apply friction to Z axis
				}

				// Update the instance position in the renderer
				ss.sandModel.SetInstancePosition(i, p.position)
			}
		}(start, end)
	}
	wg.Wait()
}

func (ss *SandSimulation) UpdateFixed() {}
