package behaviour

import (
	"testing"

	"github.com/go-gl/mathgl/mgl32"
)

func TestNewGameObject(t *testing.T) {
	obj := NewGameObject("TestObject")

	if obj == nil {
		t.Fatal("NewGameObject returned nil")
	}

	if obj.Name != "TestObject" {
		t.Errorf("Expected name 'TestObject', got '%s'", obj.Name)
	}

	if !obj.Active {
		t.Error("New GameObject should be active by default")
	}

	if obj.Transform == nil {
		t.Fatal("Transform should not be nil")
	}

	if obj.Transform.Position != (mgl32.Vec3{0, 0, 0}) {
		t.Errorf("Expected position (0,0,0), got %v", obj.Transform.Position)
	}

	if obj.Transform.Scale != (mgl32.Vec3{1, 1, 1}) {
		t.Errorf("Expected scale (1,1,1), got %v", obj.Transform.Scale)
	}
}

func TestTransformSetPosition(t *testing.T) {
	transform := &Transform{
		Position: mgl32.Vec3{0, 0, 0},
		Scale:    mgl32.Vec3{1, 1, 1},
	}

	transform.SetPosition(mgl32.Vec3{10, 20, 30})

	if transform.Position != (mgl32.Vec3{10, 20, 30}) {
		t.Errorf("Expected position (10,20,30), got %v", transform.Position)
	}
}

func TestTransformTranslate(t *testing.T) {
	transform := &Transform{
		Position: mgl32.Vec3{5, 5, 5},
		Scale:    mgl32.Vec3{1, 1, 1},
	}

	transform.Translate(mgl32.Vec3{1, 2, 3})

	expected := mgl32.Vec3{6, 7, 8}
	if transform.Position != expected {
		t.Errorf("Expected position %v, got %v", expected, transform.Position)
	}
}

func TestTransformSetScale(t *testing.T) {
	transform := &Transform{
		Position: mgl32.Vec3{0, 0, 0},
		Scale:    mgl32.Vec3{1, 1, 1},
	}

	transform.SetScale(mgl32.Vec3{2, 3, 4})

	if transform.Scale != (mgl32.Vec3{2, 3, 4}) {
		t.Errorf("Expected scale (2,3,4), got %v", transform.Scale)
	}
}

type MockComponent struct {
	BaseComponent
	startCalled  bool
	updateCalled bool
	fixedCalled  bool
}

func (m *MockComponent) Start() {
	m.startCalled = true
}

func (m *MockComponent) Update() {
	m.updateCalled = true
}

func (m *MockComponent) FixedUpdate() {
	m.fixedCalled = true
}

func TestGameObjectAddComponent(t *testing.T) {
	obj := NewGameObject("Test")
	comp := &MockComponent{}

	obj.AddComponent(comp)

	if len(obj.Components) != 1 {
		t.Errorf("Expected 1 component, got %d", len(obj.Components))
	}

	if comp.GetGameObject() != obj {
		t.Error("Component's GameObject reference not set correctly")
	}
}

func TestGameObjectRemoveComponent(t *testing.T) {
	obj := NewGameObject("Test")
	comp := &MockComponent{}

	obj.AddComponent(comp)
	obj.RemoveComponent(comp)

	if len(obj.Components) != 0 {
		t.Errorf("Expected 0 components after removal, got %d", len(obj.Components))
	}
}
