package behaviour

import (
	"testing"
)

func TestComponentManagerRegister(t *testing.T) {
	cm := NewComponentManager()
	obj := NewGameObject("Test")

	cm.RegisterGameObject(obj)

	all := cm.GetAllGameObjects()
	if len(all) != 1 {
		t.Errorf("Expected 1 registered object, got %d", len(all))
	}
}

func TestComponentManagerUnregister(t *testing.T) {
	cm := NewComponentManager()
	obj := NewGameObject("Test")

	cm.RegisterGameObject(obj)
	cm.UnregisterGameObject(obj)

	all := cm.GetAllGameObjects()
	if len(all) != 0 {
		t.Errorf("Expected 0 objects after unregister, got %d", len(all))
	}
}

func TestComponentManagerUpdateAll(t *testing.T) {
	cm := NewComponentManager()
	obj := NewGameObject("Test")
	comp := &MockComponent{}
	obj.AddComponent(comp)
	cm.RegisterGameObject(obj)

	cm.UpdateAll()

	if !comp.updateCalled {
		t.Error("Update() was not called on component")
	}
}

func TestComponentManagerFixedUpdateAll(t *testing.T) {
	cm := NewComponentManager()
	obj := NewGameObject("Test")
	comp := &MockComponent{}
	obj.AddComponent(comp)
	cm.RegisterGameObject(obj)

	cm.FixedUpdateAll()

	if !comp.fixedCalled {
		t.Error("FixedUpdate() was not called on component")
	}
}

func TestComponentManagerInactiveObject(t *testing.T) {
	cm := NewComponentManager()
	obj := NewGameObject("Test")
	obj.Active = false
	comp := &MockComponent{}
	obj.AddComponent(comp)
	cm.RegisterGameObject(obj)

	cm.UpdateAll()

	if comp.updateCalled {
		t.Error("Update() should not be called on inactive object")
	}
}

func TestComponentManagerFindGameObject(t *testing.T) {
	cm := NewComponentManager()
	obj := NewGameObject("FindMe")
	cm.RegisterGameObject(obj)

	found := cm.FindGameObject("FindMe")

	if found == nil {
		t.Error("FindGameObject should find registered object")
	}
	if found != obj {
		t.Error("FindGameObject returned wrong object")
	}
}

func TestComponentManagerFindGameObjectNotFound(t *testing.T) {
	cm := NewComponentManager()

	found := cm.FindGameObject("NotHere")

	if found != nil {
		t.Error("FindGameObject should return nil for non-existent object")
	}
}

func TestComponentManagerClear(t *testing.T) {
	cm := NewComponentManager()
	cm.RegisterGameObject(NewGameObject("A"))
	cm.RegisterGameObject(NewGameObject("B"))

	cm.Clear()

	all := cm.GetAllGameObjects()
	if len(all) != 0 {
		t.Errorf("Clear should remove all objects, got %d", len(all))
	}
}
