package behaviour

import "sort"

type ScriptConstructor func() Component

var scriptRegistry = make(map[string]ScriptConstructor)

func RegisterScript(name string, constructor ScriptConstructor) {
	scriptRegistry[name] = constructor
}

func GetAvailableScripts() []string {
	names := make([]string, 0, len(scriptRegistry))
	for name := range scriptRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func CreateScript(name string) Component {
	if constructor, exists := scriptRegistry[name]; exists {
		return constructor()
	}
	return nil
}
