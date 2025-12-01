package behaviour

// ComponentManager manages all GameObjects and their components
// Similar to Unity's scene management system
type ComponentManager struct {
	gameObjects []*GameObject
	toDestroy   []*GameObject
}

var GlobalComponentManager = NewComponentManager()

func NewComponentManager() *ComponentManager {
	return &ComponentManager{
		gameObjects: make([]*GameObject, 0),
		toDestroy:   make([]*GameObject, 0),
	}
}

// RegisterGameObject adds a GameObject to the manager
func (cm *ComponentManager) RegisterGameObject(obj *GameObject) {
	cm.gameObjects = append(cm.gameObjects, obj)
	obj.internalStart()
}

func (cm *ComponentManager) UnregisterGameObject(obj *GameObject) {
	for i, o := range cm.gameObjects {
		if o == obj {
			cm.gameObjects = append(cm.gameObjects[:i], cm.gameObjects[i+1:]...)
			obj.Destroy()
			return
		}
	}
}

// FindGameObject finds a GameObject by name
func (cm *ComponentManager) FindGameObject(name string) *GameObject {
	for _, obj := range cm.gameObjects {
		if obj.Name == name {
			return obj
		}
	}
	return nil
}

// FindGameObjectsWithTag finds all GameObjects with a specific tag
func (cm *ComponentManager) FindGameObjectsWithTag(tag string) []*GameObject {
	var result []*GameObject
	for _, obj := range cm.gameObjects {
		if obj.Tag == tag {
			result = append(result, obj)
		}
	}
	return result
}

// UpdateAll calls Update on all active GameObjects
func (cm *ComponentManager) UpdateAll() {
	// Process destroyed objects
	if len(cm.toDestroy) > 0 {
		for _, obj := range cm.toDestroy {
			cm.UnregisterGameObject(obj)
		}
		cm.toDestroy = cm.toDestroy[:0]
	}

	for _, obj := range cm.gameObjects {
		if obj.Active {
			if obj.GetModel() != nil {
				if model, ok := obj.GetModel().(ModelInterface); ok {
					modelPos := model.GetPosition()
					modelRot := model.GetRotation()
					modelScale := model.GetScale()

					if !obj.Transform.Position.ApproxEqual(modelPos) ||
						!obj.Transform.Rotation.ApproxEqual(modelRot) ||
						!obj.Transform.Scale.ApproxEqual(modelScale) {
						obj.Transform.Position = modelPos
						obj.Transform.Rotation = modelRot
						obj.Transform.Scale = modelScale
					}
				}
			}

			obj.internalUpdate()

			if obj.GetModel() != nil {
				if model, ok := obj.GetModel().(ModelInterface); ok {
					modelPos := model.GetPosition()

					if !obj.Transform.Position.ApproxEqual(modelPos) ||
						!obj.Transform.Rotation.ApproxEqual(model.GetRotation()) ||
						!obj.Transform.Scale.ApproxEqual(model.GetScale()) {
						model.SetPositionVec(obj.Transform.Position)
						model.SetRotationQuat(obj.Transform.Rotation)
						model.SetScaleVec(obj.Transform.Scale)
					}
				}
			}
		}
	}
}

// FixedUpdateAll calls FixedUpdate on all active GameObjects
func (cm *ComponentManager) FixedUpdateAll() {
	for _, obj := range cm.gameObjects {
		if obj.Active {
			obj.internalFixedUpdate()
		}
	}
}

// DestroyGameObject marks a GameObject for destruction (will be removed next frame)
func (cm *ComponentManager) DestroyGameObject(obj *GameObject) {
	cm.toDestroy = append(cm.toDestroy, obj)
}

// GetAllGameObjects returns all registered GameObjects
func (cm *ComponentManager) GetAllGameObjects() []*GameObject {
	return cm.gameObjects
}

// Clear removes all GameObjects
func (cm *ComponentManager) Clear() {
	for _, obj := range cm.gameObjects {
		obj.Destroy()
	}
	cm.gameObjects = cm.gameObjects[:0]
	cm.toDestroy = cm.toDestroy[:0]
}
