package scripts

import (
	"Gopher3D/internal/behaviour"
	"math"
)

type OrbitScript struct {
	behaviour.BaseComponent
	Radius float32
	Speed  float32
	time   float32
}

func init() {
	behaviour.RegisterScript("OrbitScript", func() behaviour.Component {
		return &OrbitScript{Radius: 10.0, Speed: 1.0}
	})
	println("Registered OrbitScript")
}

func (o *OrbitScript) Start() {}

func (o *OrbitScript) Update() {
	deltaTime := float32(0.016)
	o.time += deltaTime * o.Speed
	
	x := float32(math.Cos(float64(o.time))) * o.Radius
	z := float32(math.Sin(float64(o.time))) * o.Radius
	
	o.GetGameObject().Transform.Position[0] = x
	o.GetGameObject().Transform.Position[2] = z
}

func (o *OrbitScript) FixedUpdate() {}

