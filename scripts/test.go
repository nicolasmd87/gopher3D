package scripts

import (
	"Gopher3D/internal/behaviour"
	mgl "github.com/go-gl/mathgl/mgl32"
)

// TestScript is a custom script component
type TestScript struct {
	behaviour.BaseComponent
	Speed float32
}

func init() {
	behaviour.RegisterScript("TestScript", func() behaviour.Component {
		return &TestScript{Speed: 1.0}
	})
}

func (s *TestScript) Start() {
	// Called once when script starts
}

func (s *TestScript) Update() {
	// Called every frame
	// Example: rotate the object
	if obj := s.GetGameObject(); obj != nil {
		obj.Transform.Rotate(mgl.Vec3{0, 1, 0}, mgl.DegToRad(s.Speed))
	}
}

func (s *TestScript) FixedUpdate() {
	// Called at fixed intervals for physics
}
