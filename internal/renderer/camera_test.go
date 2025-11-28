package renderer

import (
	"math"
	"testing"

	"github.com/go-gl/mathgl/mgl32"
)

func TestNewDefaultCamera(t *testing.T) {
	cam := NewDefaultCamera(800, 600)

	if cam == nil {
		t.Fatal("NewDefaultCamera returned nil")
	}

	if cam.Position == (mgl32.Vec3{0, 0, 0}) {
		t.Error("Camera position should not be at origin")
	}

	if cam.Speed <= 0 {
		t.Error("Camera speed should be positive")
	}

	if cam.Sensitivity <= 0 {
		t.Error("Camera sensitivity should be positive")
	}
}

func TestCameraGetViewMatrix(t *testing.T) {
	cam := NewDefaultCamera(800, 600)
	cam.Position = mgl32.Vec3{0, 0, 5}
	cam.Front = mgl32.Vec3{0, 0, -1}
	cam.Up = mgl32.Vec3{0, 1, 0}

	view := cam.GetViewMatrix()

	if view.At(3, 3) != 1.0 {
		t.Error("View matrix should be valid (w component = 1)")
	}
}

func TestCameraGetProjectionMatrix(t *testing.T) {
	cam := NewDefaultCamera(800, 600)

	proj := cam.GetProjectionMatrix()

	if proj.At(3, 3) != 0.0 {
		t.Error("Perspective projection should have w=0 at (3,3)")
	}
}

func TestCameraGetViewProjection(t *testing.T) {
	cam := NewDefaultCamera(800, 600)

	vp := cam.GetViewProjection()

	zero := mgl32.Mat4{}
	if vp == zero {
		t.Error("ViewProjection should not be zero matrix")
	}
}

func TestCameraPositionDirect(t *testing.T) {
	cam := NewDefaultCamera(800, 600)

	cam.Position = mgl32.Vec3{10, 20, 30}

	if cam.Position.X() != 10 || cam.Position.Y() != 20 || cam.Position.Z() != 30 {
		t.Errorf("Expected position (10,20,30), got %v", cam.Position)
	}
}

func TestCameraUpdateVectors(t *testing.T) {
	cam := NewDefaultCamera(800, 600)
	cam.Yaw = -90
	cam.Pitch = 0

	cam.updateCameraVectors()

	frontLen := cam.Front.Len()
	if math.Abs(float64(frontLen)-1.0) > 0.01 {
		t.Errorf("Front vector should be normalized, length=%f", frontLen)
	}
}

func TestCameraInvertMouse(t *testing.T) {
	cam := NewDefaultCamera(800, 600)

	cam.InvertMouse = false
	if cam.InvertMouse != false {
		t.Error("InvertMouse should be false")
	}

	cam.InvertMouse = true
	if cam.InvertMouse != true {
		t.Error("InvertMouse should be true")
	}
}
