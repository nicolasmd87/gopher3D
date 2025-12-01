package main

import (
	"Gopher3D/internal/behaviour"
	"github.com/go-gl/mathgl/mgl32"
)

// Rotate2Script is a custom script component
type Rotate2Script struct {
	behaviour.BaseComponent

	// Add your custom fields here
	Speed float32
}

func init() {
	behaviour.RegisterScript("Rotate2Script", func() behaviour.Component {
		return &Rotate2Script{Speed: 1.0}
	})
}

// Start is called once when the script is first activated
func (s *Rotate2Script) Start() {
	// Initialize your script here
}

// Update is called every frame
func (s *Rotate2Script) Update() {
	// Add your update logic here
	// Example: rotate the object
	// transform := s.GetGameObject().Transform
	// transform.Rotate(mgl32.Vec3{0, 1, 0}, mgl32.DegToRad(s.Speed * 0.016))
	_ = mgl32.Vec3{} // Placeholder to avoid unused import
}

// FixedUpdate is called at fixed time intervals (physics)
func (s *Rotate2Script) FixedUpdate() {
	// Add your physics logic here
}
