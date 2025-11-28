package behaviour

import (
	"testing"
)

func TestRegisterScript(t *testing.T) {
	scriptRegistry = make(map[string]ScriptConstructor)

	RegisterScript("TestScript", func() Component {
		return &MockComponent{}
	})

	scripts := GetAvailableScripts()

	if len(scripts) != 1 {
		t.Errorf("Expected 1 script, got %d", len(scripts))
	}

	if scripts[0] != "TestScript" {
		t.Errorf("Expected 'TestScript', got '%s'", scripts[0])
	}
}

func TestCreateScript(t *testing.T) {
	scriptRegistry = make(map[string]ScriptConstructor)

	RegisterScript("TestScript", func() Component {
		return &MockComponent{}
	})

	comp := CreateScript("TestScript")

	if comp == nil {
		t.Error("CreateScript returned nil")
	}
}

func TestCreateScriptNotFound(t *testing.T) {
	scriptRegistry = make(map[string]ScriptConstructor)

	comp := CreateScript("NonExistent")

	if comp != nil {
		t.Error("CreateScript should return nil for non-existent script")
	}
}

func TestGetAvailableScriptsSorted(t *testing.T) {
	scriptRegistry = make(map[string]ScriptConstructor)

	RegisterScript("Zebra", func() Component { return &MockComponent{} })
	RegisterScript("Alpha", func() Component { return &MockComponent{} })
	RegisterScript("Middle", func() Component { return &MockComponent{} })

	scripts := GetAvailableScripts()

	if len(scripts) != 3 {
		t.Fatalf("Expected 3 scripts, got %d", len(scripts))
	}

	if scripts[0] != "Alpha" {
		t.Errorf("Expected first script 'Alpha', got '%s'", scripts[0])
	}
	if scripts[1] != "Middle" {
		t.Errorf("Expected second script 'Middle', got '%s'", scripts[1])
	}
	if scripts[2] != "Zebra" {
		t.Errorf("Expected third script 'Zebra', got '%s'", scripts[2])
	}
}
