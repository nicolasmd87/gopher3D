package behaviour

import (
	"github.com/go-gl/mathgl/mgl32"
)

// Component is the base interface for all components
// Components can be attached to models/game objects
type Component interface {
	// Lifecycle methods
	Awake()       // Called when component is first created
	Start()       // Called before first Update (after all Awakes)
	Update()      // Called every frame
	FixedUpdate() // Called at fixed time intervals
	OnDestroy()   // Called when component/object is destroyed

	// Component info
	GetEnabled() bool
	SetEnabled(bool)
	GetGameObject() *GameObject
	SetGameObject(*GameObject)
}

// BaseComponent provides default implementations for all Component methods
// User scripts can embed this to only override methods they need
type BaseComponent struct {
	enabled    bool
	gameObject *GameObject
	started    bool
}

func (c *BaseComponent) Awake()       {}
func (c *BaseComponent) Start()       {}
func (c *BaseComponent) Update()      {}
func (c *BaseComponent) FixedUpdate() {}
func (c *BaseComponent) OnDestroy()   {}

func (c *BaseComponent) GetEnabled() bool {
	return c.enabled
}

func (c *BaseComponent) SetEnabled(enabled bool) {
	c.enabled = enabled
}

func (c *BaseComponent) GetGameObject() *GameObject {
	return c.gameObject
}

func (c *BaseComponent) SetGameObject(obj *GameObject) {
	c.gameObject = obj
}

// GameObject represents an object in the scene
// This wraps around the renderer.Model to provide Unity-like functionality
type GameObject struct {
	Name       string
	Tag        string
	Active     bool
	Transform  *Transform
	Components []Component
	model      interface{} // Reference to renderer.Model (using interface to avoid circular import)
}

// Transform component
type Transform struct {
	BaseComponent
	Position mgl32.Vec3
	Rotation mgl32.Quat
	Scale    mgl32.Vec3
	Parent   *Transform
	Children []*Transform
}

// Transform methods
func (t *Transform) Translate(delta mgl32.Vec3) {
	t.Position = t.Position.Add(delta)
}

func (t *Transform) Rotate(axis mgl32.Vec3, angle float32) {
	rotation := mgl32.QuatRotate(angle, axis)
	t.Rotation = t.Rotation.Mul(rotation)
}

func (t *Transform) SetPosition(pos mgl32.Vec3) {
	t.Position = pos
}

func (t *Transform) SetRotation(rot mgl32.Quat) {
	t.Rotation = rot
}

func (t *Transform) SetScale(scale mgl32.Vec3) {
	t.Scale = scale
}

func (t *Transform) Forward() mgl32.Vec3 {
	return t.Rotation.Rotate(mgl32.Vec3{0, 0, -1})
}

func (t *Transform) Up() mgl32.Vec3 {
	return t.Rotation.Rotate(mgl32.Vec3{0, 1, 0})
}

func (t *Transform) Right() mgl32.Vec3 {
	return t.Rotation.Rotate(mgl32.Vec3{1, 0, 0})
}

// GameObject methods
func NewGameObject(name string) *GameObject {
	obj := &GameObject{
		Name:       name,
		Active:     true,
		Components: make([]Component, 0),
		Transform: &Transform{
			Position: mgl32.Vec3{0, 0, 0},
			Rotation: mgl32.QuatIdent(),
			Scale:    mgl32.Vec3{1, 1, 1},
		},
	}
	obj.Transform.SetGameObject(obj)
	return obj
}

func (obj *GameObject) AddComponent(component Component) {
	component.SetGameObject(obj)
	component.SetEnabled(true)
	obj.Components = append(obj.Components, component)
	component.Awake()
}

func (obj *GameObject) GetComponent(componentType string) Component {
	for _, comp := range obj.Components {
		if comp != nil {
			return comp
		}
	}
	return nil
}

func (obj *GameObject) GetComponents(componentType string) []Component {
	var result []Component
	for _, comp := range obj.Components {
		if comp != nil {
			result = append(result, comp)
		}
	}
	return result
}

func (obj *GameObject) RemoveComponent(component Component) {
	for i, comp := range obj.Components {
		if comp == component {
			comp.OnDestroy()
			obj.Components = append(obj.Components[:i], obj.Components[i+1:]...)
			return
		}
	}
}

func (obj *GameObject) SetModel(model interface{}) {
	obj.model = model
}

func (obj *GameObject) GetModel() interface{} {
	return obj.model
}

type ModelInterface interface {
	GetPosition() mgl32.Vec3
	GetRotation() mgl32.Quat
	GetScale() mgl32.Vec3
	SetPositionVec(mgl32.Vec3)
	SetRotationQuat(mgl32.Quat)
	SetScaleVec(mgl32.Vec3)
	MarkDirty()
}

func (obj *GameObject) internalUpdate() {
	if !obj.Active {
		return
	}

	for _, comp := range obj.Components {
		if comp.GetEnabled() {
			comp.Update()
		}
	}
}

func (obj *GameObject) internalFixedUpdate() {
	if !obj.Active {
		return
	}

	for _, comp := range obj.Components {
		if comp.GetEnabled() {
			comp.FixedUpdate()
		}
	}
}

func (obj *GameObject) internalStart() {
	if !obj.Active {
		return
	}

	for _, comp := range obj.Components {
		if comp.GetEnabled() {
			comp.Start()
		}
	}
}

func (obj *GameObject) Destroy() {
	for _, comp := range obj.Components {
		comp.OnDestroy()
	}
	obj.Active = false
}
