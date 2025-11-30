package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	rebuildRequired   = false
	rebuildInProgress = false
	rebuildMutex      sync.Mutex
	lastScriptChange  time.Time
	editorExePath     string
)

// InitHotReload initializes the hot reload system
func InitHotReload() {
	// Get current executable path for restart
	exe, err := os.Executable()
	if err == nil {
		editorExePath = exe
	}
}

// IsRebuildRequired returns true if new scripts were added
func IsRebuildRequired() bool {
	rebuildMutex.Lock()
	defer rebuildMutex.Unlock()
	return rebuildRequired
}

// IsRebuilding returns true if rebuild is in progress
func IsRebuilding() bool {
	rebuildMutex.Lock()
	defer rebuildMutex.Unlock()
	return rebuildInProgress
}

// MarkRebuildRequired marks that a rebuild is needed
func MarkRebuildRequired() {
	rebuildMutex.Lock()
	defer rebuildMutex.Unlock()
	rebuildRequired = true
	lastScriptChange = time.Now()
}

// CopyScriptToEngine copies a script to the engine's scripts folder and triggers rebuild
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

	logToConsole(fmt.Sprintf("Script copied to engine: %s", filepath.Base(scriptPath)), "info")
	MarkRebuildRequired()
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

	logToConsole(fmt.Sprintf("Created script: %s", fileName), "info")
	MarkRebuildRequired()
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
	rebuildMutex.Unlock()

	go func() {
		defer func() {
			rebuildMutex.Lock()
			rebuildInProgress = false
			rebuildMutex.Unlock()
		}()

		logToConsole("Rebuilding editor with new scripts...", "info")

		// Get module path
		modulePath := getModulePath()
		if modulePath == "" {
			logToConsole("Could not find module path", "error")
			return
		}

		// Build the editor
		outputPath := filepath.Join(modulePath, "editor_new.exe")
		cmd := exec.Command("go", "build", "-o", outputPath, "./editor/cmd")
		cmd.Dir = modulePath
		cmd.Env = os.Environ()

		output, err := cmd.CombinedOutput()
		if err != nil {
			logToConsole(fmt.Sprintf("Build failed: %s", string(output)), "error")
			return
		}

		logToConsole("Build successful! Restarting editor...", "info")

		// Create a batch script to replace and restart
		restartScript := filepath.Join(modulePath, "restart_editor.bat")
		batchContent := fmt.Sprintf(`@echo off
timeout /t 1 /nobreak >nul
move /y "%s" "%s"
start "" "%s"
del "%%~f0"
`, outputPath, editorExePath, editorExePath)

		if err := os.WriteFile(restartScript, []byte(batchContent), 0755); err != nil {
			logToConsole(fmt.Sprintf("Failed to create restart script: %v", err), "error")
			return
		}

		// Execute restart script and exit
		restartCmd := exec.Command("cmd", "/c", "start", "/b", restartScript)
		restartCmd.Start()

		// Exit current process
		os.Exit(0)
	}()
}

// AutoRebuildIfNeeded checks if rebuild is needed and triggers it automatically
func AutoRebuildIfNeeded() {
	if !IsRebuildRequired() || IsRebuilding() {
		return
	}

	// Wait a bit after last change to batch multiple additions
	rebuildMutex.Lock()
	timeSinceChange := time.Since(lastScriptChange)
	rebuildMutex.Unlock()

	if timeSinceChange > 2*time.Second {
		TriggerEditorRebuild()
	}
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

