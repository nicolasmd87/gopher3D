package scripts

import (
	"Gopher3D/internal/behaviour"
	"github.com/go-gl/mathgl/mgl32"
)

type RotateScript struct {
	behaviour.BaseComponent
	Speed float32
}

func init() {
	behaviour.RegisterScript("RotateScript", func() behaviour.Component {
		return &RotateScript{Speed: 45.0}
	})
	println("Registered RotateScript")
}

func (r *RotateScript) Start() {}

func (r *RotateScript) Update() {
	deltaTime := float32(0.016)
	transform := r.GetGameObject().Transform
	transform.Rotate(mgl32.Vec3{0, 1, 0}, mgl32.DegToRad(r.Speed*deltaTime))
}

func (r *RotateScript) FixedUpdate() {}
