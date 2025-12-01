package scripts

import (
	"Gopher3D/internal/behaviour"
	mgl "github.com/go-gl/mathgl/mgl32"
)

// CheckScript is a custom script component
type CheckScript struct {
	behaviour.BaseComponent
	Speed float32
}

func init() {
	behaviour.RegisterScript("CheckScript", func() behaviour.Component {
		return &CheckScript{Speed: 1.0}
	})
}

func (s *CheckScript) Start() {
	// Called once when script starts
}

func (s *CheckScript) Update() {
	// Called every frame
	// Example: rotate the object
	if obj := s.GetGameObject(); obj != nil {
		obj.Transform.Rotate(mgl.Vec3{0, 1, 0}, mgl.DegToRad(s.Speed))
	}
}

func (s *CheckScript) FixedUpdate() {
	// Called at fixed intervals for physics
}
