package scripts

import (
	"Gopher3D/internal/behaviour"
	"math"
)

type BounceScript struct {
	behaviour.BaseComponent
	Height float32
	Speed  float32
	startY float32
	time   float32
}

func init() {
	behaviour.RegisterScript("BounceScript", func() behaviour.Component {
		return &BounceScript{Height: 5.0, Speed: 2.0}
	})
	println("Registered BounceScript")
}

func (b *BounceScript) Start() {
	b.startY = b.GetGameObject().Transform.Position.Y()
}

func (b *BounceScript) Update() {
	deltaTime := float32(0.016)
	b.time += deltaTime * b.Speed
	offset := float32(math.Sin(float64(b.time))) * b.Height
	b.GetGameObject().Transform.Position[1] = b.startY + offset
}

func (b *BounceScript) FixedUpdate() {}

