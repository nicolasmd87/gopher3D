package editor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var (
	rebuildInProgress = false
	rebuildMutex      sync.Mutex
	rebuildOutput     string
	rebuildError      string
	rebuildSuccess    bool
	showRebuildModal  = false
	editorExePath     string
	tempSceneFile     = ""
	pendingRestart    = false
)

// InitHotReload initializes the hot reload system
func InitHotReload() {
	// Get current executable path for restart
	exe, err := os.Executable()
	if err == nil {
		editorExePath = exe
	}

	// Set up temp scene file path
	tempDir := os.TempDir()
	tempSceneFile = filepath.Join(tempDir, "gopher3d_restore_scene.json")

	// Check if we should restore a scene (launched with --restore flag)
	for _, arg := range os.Args {
		if arg == "--restore" {
			RestoreSceneOnStartup()
			break
		}
	}
}

// IsRebuilding returns true if rebuild is in progress
func IsRebuilding() bool {
	rebuildMutex.Lock()
	defer rebuildMutex.Unlock()
	return rebuildInProgress
}

// GetRebuildOutput returns the current build output
func GetRebuildOutput() string {
	rebuildMutex.Lock()
	defer rebuildMutex.Unlock()
	return rebuildOutput
}

// GetRebuildError returns the build error if any
func GetRebuildError() string {
	rebuildMutex.Lock()
	defer rebuildMutex.Unlock()
	return rebuildError
}

// WasRebuildSuccessful returns true if last rebuild succeeded
func WasRebuildSuccessful() bool {
	rebuildMutex.Lock()
	defer rebuildMutex.Unlock()
	return rebuildSuccess
}

// IsPendingRestart returns true if editor should restart
func IsPendingRestart() bool {
	rebuildMutex.Lock()
	defer rebuildMutex.Unlock()
	return pendingRestart
}

// ShowRebuildModal returns whether to show the rebuild modal
func ShowRebuildModal() bool {
	return showRebuildModal
}

// SetShowRebuildModal sets the rebuild modal visibility
func SetShowRebuildModal(show bool) {
	showRebuildModal = show
}

// CopyScriptToEngine copies a script to the engine's scripts folder
func CopyScriptToEngine(scriptPath string) error {
	// Get the engine scripts directory
	engineScriptsDir := getEngineScriptsDir()
	if engineScriptsDir == "" {
		return fmt.Errorf("could not find engine scripts directory")
	}

	// Read the script
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read script: %v", err)
	}

	// Ensure package is "scripts"
	contentStr := string(content)
	if strings.Contains(contentStr, "package main") {
		contentStr = strings.Replace(contentStr, "package main", "package scripts", 1)
	}

	// Write to engine scripts folder
	destPath := filepath.Join(engineScriptsDir, filepath.Base(scriptPath))
	if err := os.WriteFile(destPath, []byte(contentStr), 0644); err != nil {
		return fmt.Errorf("failed to write script: %v", err)
	}

	logToConsole(fmt.Sprintf("Script imported: %s", filepath.Base(scriptPath)), "success")
	logToConsole("Click 'Rebuild Editor' to use the new script", "info")
	return nil
}

// CreateAndAddScript creates a new script and adds it to the engine
func CreateAndAddScript(name string) error {
	engineScriptsDir := getEngineScriptsDir()
	if engineScriptsDir == "" {
		return fmt.Errorf("could not find engine scripts directory")
	}

	// Normalize script name
	scriptName := strings.Title(strings.TrimSuffix(name, "Script"))
	if !strings.HasSuffix(scriptName, "Script") {
		scriptName += "Script"
	}

	fileName := strings.ToLower(name) + ".go"
	if !strings.HasSuffix(fileName, ".go") {
		fileName = strings.ToLower(scriptName) + ".go"
	}
	filePath := filepath.Join(engineScriptsDir, fileName)

	// Check if exists
	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("script already exists: %s", fileName)
	}

	// Create script content
	content := fmt.Sprintf(`package scripts

import (
	"Gopher3D/internal/behaviour"
	mgl "github.com/go-gl/mathgl/mgl32"
)

// %s is a custom script component
type %s struct {
	behaviour.BaseComponent
	Speed float32
}

func init() {
	behaviour.RegisterScript("%s", func() behaviour.Component {
		return &%s{Speed: 1.0}
	})
}

func (s *%s) Start() {
	// Called once when script starts
}

func (s *%s) Update() {
	// Called every frame
	// Example: rotate the object
	if obj := s.GetGameObject(); obj != nil {
		obj.Transform.Rotate(mgl.Vec3{0, 1, 0}, mgl.DegToRad(s.Speed))
	}
}

func (s *%s) FixedUpdate() {
	// Called at fixed intervals for physics
}
`, scriptName, scriptName, scriptName, scriptName, scriptName, scriptName, scriptName)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create script: %v", err)
	}

	logToConsole(fmt.Sprintf("Created script: %s", fileName), "success")
	logToConsole("Click 'Rebuild Editor' to use the new script", "info")
	return nil
}

// AddScriptToEngine imports an existing script file to the engine
func AddScriptToEngine(scriptPath string) error {
	return CopyScriptToEngine(scriptPath)
}

// TriggerEditorRebuild rebuilds the editor with new scripts
func TriggerEditorRebuild() {
	rebuildMutex.Lock()
	if rebuildInProgress {
		rebuildMutex.Unlock()
		return
	}
	rebuildInProgress = true
	rebuildOutput = "Starting build...\n"
	rebuildError = ""
	rebuildSuccess = false
	pendingRestart = false
	rebuildMutex.Unlock()

	showRebuildModal = true

	go func() {
		defer func() {
			rebuildMutex.Lock()
			rebuildInProgress = false
			rebuildMutex.Unlock()
		}()

		// Get module path
		modulePath := getModulePath()
		if modulePath == "" {
			rebuildMutex.Lock()
			rebuildError = "Could not find module path (go.mod)"
			rebuildOutput += "ERROR: " + rebuildError + "\n"
			rebuildMutex.Unlock()
			return
		}

		rebuildMutex.Lock()
		rebuildOutput += fmt.Sprintf("Module path: %s\n", modulePath)
		rebuildMutex.Unlock()

		// Determine output executable name
		exeName := "editor_new"
		if runtime.GOOS == "windows" {
			exeName += ".exe"
		}
		outputPath := filepath.Join(modulePath, exeName)

		rebuildMutex.Lock()
		rebuildOutput += fmt.Sprintf("Building: go build -o %s ./editor/cmd\n", exeName)
		rebuildMutex.Unlock()

		// Build the editor
		cmd := exec.Command("go", "build", "-o", outputPath, "./editor/cmd")
		cmd.Dir = modulePath
		cmd.Env = os.Environ()

		output, err := cmd.CombinedOutput()

		rebuildMutex.Lock()
		rebuildOutput += string(output)

		if err != nil {
			rebuildError = fmt.Sprintf("Build failed: %v", err)
			rebuildOutput += "\n" + rebuildError + "\n"
			rebuildMutex.Unlock()
			return
		}

		rebuildOutput += "\nBuild successful!\n"
		rebuildSuccess = true
		pendingRestart = true
		rebuildMutex.Unlock()

		logToConsole("Build successful! Click 'Restart' to apply changes.", "success")
	}()
}

// SaveSceneAndRestart saves the current scene and restarts the editor
func SaveSceneAndRestart() {
	// Save current scene state to temp file
	if err := saveSceneForRestore(); err != nil {
		logToConsole(fmt.Sprintf("Warning: Could not save scene state: %v", err), "warning")
	}

	modulePath := getModulePath()
	if modulePath == "" {
		logToConsole("Could not find module path", "error")
		return
	}

	// Determine executable names
	exeName := "editor_new"
	currentExeName := "editor"
	if runtime.GOOS == "windows" {
		exeName += ".exe"
		currentExeName += ".exe"
	}

	newExePath := filepath.Join(modulePath, exeName)
	targetExePath := filepath.Join(modulePath, currentExeName)

	// Check if new executable exists
	if _, err := os.Stat(newExePath); os.IsNotExist(err) {
		logToConsole("New editor executable not found. Build first.", "error")
		return
	}

	logToConsole("Restarting editor...", "info")

	if runtime.GOOS == "windows" {
		// Windows: use a batch script to replace and restart
		restartScript := filepath.Join(modulePath, "restart_editor.bat")
		batchContent := fmt.Sprintf(`@echo off
timeout /t 1 /nobreak >nul
del /f "%s" 2>nul
move /y "%s" "%s"
start "" "%s" --restore
del "%%~f0"
`, targetExePath, newExePath, targetExePath, targetExePath)

		if err := os.WriteFile(restartScript, []byte(batchContent), 0755); err != nil {
			logToConsole(fmt.Sprintf("Failed to create restart script: %v", err), "error")
			return
		}

		// Execute restart script
		restartCmd := exec.Command("cmd", "/c", "start", "/b", restartScript)
		restartCmd.Start()
	} else {
		// Unix: use a shell script
		restartScript := filepath.Join(modulePath, "restart_editor.sh")
		shellContent := fmt.Sprintf(`#!/bin/bash
sleep 1
rm -f "%s"
mv "%s" "%s"
"%s" --restore &
rm -f "$0"
`, targetExePath, newExePath, targetExePath, targetExePath)

		if err := os.WriteFile(restartScript, []byte(shellContent), 0755); err != nil {
			logToConsole(fmt.Sprintf("Failed to create restart script: %v", err), "error")
			return
		}

		// Execute restart script
		restartCmd := exec.Command("bash", restartScript)
		restartCmd.Start()
	}

	// Exit current process
	os.Exit(0)
}

// saveSceneForRestore saves minimal scene state for restoration after restart
func saveSceneForRestore() error {
	// Create a simple state object with current scene path
	state := map[string]interface{}{
		"scenePath":   currentScenePath,
		"projectPath": "",
	}

	if CurrentProject != nil {
		state["projectPath"] = CurrentProject.Path
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(tempSceneFile, data, 0644)
}

// RestoreSceneOnStartup restores scene state after editor restart
func RestoreSceneOnStartup() {
	if tempSceneFile == "" {
		return
	}

	data, err := os.ReadFile(tempSceneFile)
	if err != nil {
		return // No restore file, that's fine
	}

	// Clean up temp file
	defer os.Remove(tempSceneFile)

	var state map[string]interface{}
	if err := json.Unmarshal(data, &state); err != nil {
		return
	}

	// Restore project path if available
	if projectPath, ok := state["projectPath"].(string); ok && projectPath != "" {
		// Queue project loading for after initialization
		pendingProjectPath = projectPath
	}

	// Restore scene path if available
	if scenePath, ok := state["scenePath"].(string); ok && scenePath != "" {
		// Queue scene loading for after initialization
		pendingScenePath = scenePath
	}

	logToConsole("Editor restarted with new scripts", "success")
}

// Pending paths for restoration after init
var (
	pendingProjectPath = ""
	pendingScenePath   = ""
)

// GetPendingProjectPath returns and clears the pending project path
func GetPendingProjectPath() string {
	path := pendingProjectPath
	pendingProjectPath = ""
	return path
}

// GetPendingScenePath returns and clears the pending scene path
func GetPendingScenePath() string {
	path := pendingScenePath
	pendingScenePath = ""
	return path
}

// getEngineScriptsDir returns the path to the engine's scripts folder
func getEngineScriptsDir() string {
	modulePath := getModulePath()
	if modulePath == "" {
		return ""
	}

	scriptsDir := filepath.Join(modulePath, "scripts")

	// Create if doesn't exist
	os.MkdirAll(scriptsDir, 0755)

	return scriptsDir
}

// ClearRebuildState resets the rebuild state for a new build
func ClearRebuildState() {
	rebuildMutex.Lock()
	defer rebuildMutex.Unlock()
	rebuildOutput = ""
	rebuildError = ""
	rebuildSuccess = false
	pendingRestart = false
}
