package behaviour

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

	for i := 0; i < len(names)-1; i++ {
		for j := i + 1; j < len(names); j++ {
			if names[i] > names[j] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}

	return names
}

func CreateScript(name string) Component {
	if constructor, exists := scriptRegistry[name]; exists {
		return constructor()
	}
	return nil
}
