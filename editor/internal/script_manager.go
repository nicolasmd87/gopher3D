package editor

import (
	"os"
	"path/filepath"
	"strings"
)

// ProjectScript represents a script file found in the project
type ProjectScript struct {
	Name     string // Script name (without extension)
	Path     string // Full path to the script file
	FileName string // File name with extension
}

var (
	projectScripts     []ProjectScript
	lastScriptScanPath = ""
)

// ScanProjectScripts scans the project's scripts folder for .go files
func ScanProjectScripts() {
	if CurrentProject == nil {
		projectScripts = nil
		return
	}

	scriptsPath := filepath.Join(CurrentProject.Path, "resources", "scripts")

	// Don't rescan if path hasn't changed
	if scriptsPath == lastScriptScanPath && len(projectScripts) > 0 {
		return
	}
	lastScriptScanPath = scriptsPath

	projectScripts = nil

	// Check if scripts folder exists
	if _, err := os.Stat(scriptsPath); os.IsNotExist(err) {
		return
	}

	// Scan for .go files
	entries, err := os.ReadDir(scriptsPath)
	if err != nil {
		logToConsole("Error scanning scripts folder: "+err.Error(), "error")
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".go") {
			continue
		}

		// Extract script name (remove .go extension)
		scriptName := strings.TrimSuffix(name, ".go")
		scriptName = strings.TrimSuffix(scriptName, ".Go")
		scriptName = strings.TrimSuffix(scriptName, ".GO")

		projectScripts = append(projectScripts, ProjectScript{
			Name:     scriptName,
			Path:     filepath.Join(scriptsPath, name),
			FileName: name,
		})
	}
}

// GetProjectScripts returns all scripts found in the project
func GetProjectScripts() []ProjectScript {
	ScanProjectScripts()
	return projectScripts
}

// GetFilteredProjectScripts returns scripts matching the search text
func GetFilteredProjectScripts(searchText string) []ProjectScript {
	ScanProjectScripts()

	if searchText == "" {
		return projectScripts
	}

	searchLower := strings.ToLower(searchText)
	var filtered []ProjectScript

	for _, script := range projectScripts {
		if strings.Contains(strings.ToLower(script.Name), searchLower) ||
			strings.Contains(strings.ToLower(script.FileName), searchLower) {
			filtered = append(filtered, script)
		}
	}

	return filtered
}

// RefreshProjectScripts forces a rescan of the scripts folder
func RefreshProjectScripts() {
	lastScriptScanPath = ""
	ScanProjectScripts()
}

// compileProjectScripts validates all scripts in the project
func compileProjectScripts() {
	if CurrentProject == nil {
		logToConsole("No project open", "error")
		return
	}

	// Initialize compiler if needed
	if GlobalScriptCompiler == nil {
		InitScriptCompiler(CurrentProject.Path)
	}

	// Validate scripts
	err := GlobalScriptCompiler.CompileScripts()
	if err != nil {
		for _, e := range GlobalScriptCompiler.CompileErrors {
			logToConsole(e, "error")
		}
	}

	// Refresh the script list
	RefreshProjectScripts()
}

// createNewScript creates a new script from template
func createNewScript(name string) {
	if CurrentProject == nil {
		logToConsole("No project open - cannot create script", "error")
		return
	}

	// Initialize compiler if needed
	if GlobalScriptCompiler == nil {
		InitScriptCompiler(CurrentProject.Path)
	}

	// Create the script
	path, err := GlobalScriptCompiler.CreateScriptTemplate(name)
	if err != nil {
		logToConsole("Failed to create script: "+err.Error(), "error")
		return
	}

	logToConsole("Created script: "+filepath.Base(path), "info")
	logToConsole("Edit the script file, then click 'Compile All'", "info")

	// Refresh the script list
	RefreshProjectScripts()
}
