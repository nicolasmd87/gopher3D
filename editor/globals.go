package main

import (
	"Gopher3D/internal/engine"
	"Gopher3D/editor/platforms"
	"Gopher3D/editor/renderers"
	"time"
)

var (
	eng           *engine.Gopher
	platform      *platforms.GLFW
	imguiRenderer *renderers.OpenGL3

	selectedModelIndex = -1
	selectedLightIndex = -1
	selectedType       = "" // "model" or "light"
	showHierarchy      = true
	showInspector      = true
	showDemoWindow     = false
	showAddModel       = false
	showAddLight       = false
	showFileExplorer   = true
	showConsole        = true
	showStyleEditor    = false
	showAdvancedRender = false
	showSceneSettings  = false
	showGizmos         = true // Default to visible
	
	// New Feature Flags
	showAddWater       = false
	showAddVoxel       = false

	// FPS tracking
	lastFrameTime = time.Now()
	frameCount    = 0
	fps           = 0.0
	fpsUpdateTime = time.Now()

	imguiInitialized = false
	sceneSetup       = false
	firstFrameComplete = false

	// File explorer state
	currentDirectory        = "." // Will be updated to absolute path
	selectedFilePath        = ""
	fileExplorerSearch      = ""         // Search/filter text
	fileExplorerPathHistory = []string{} // Breadcrumb history

	// Console state
	consoleLines      = []ConsoleEntry{}
	consoleInput      = ""
	consoleAutoScroll = true
	maxConsoleLines   = 500

	// Scene management
	currentScenePath = ""
	sceneModified    = false

	// Skybox management
	currentSkyboxPath = ""
	skyboxColorMode   = true
	skyboxSolidColor  = [3]float32{0.4, 0.6, 0.9} // Default sky blue

	// Model instancing
	instanceModelOnAdd = false
	instanceCount      = 1

	// Available models to load (paths relative to editor/ directory)
	availableModels = []struct {
		Name string
		Path string
	}{
	}

	// Editor config
	configPath = "editor_config.json"

	// Panel layouts (current state)
	hierarchyLayout      PanelLayout
	inspectorLayout      PanelLayout
	fileExplorerLayout   PanelLayout
	consoleLayout        PanelLayout
	sceneSettingsLayout  PanelLayout
	advancedRenderLayout PanelLayout

	// Track if layouts have been initialized
	layoutsInitialized = false
	
	// Feature Instances
	activeWaterSim *WaterSimulation
)

type ConsoleEntry struct {
	Message string
	Type    string // "info", "warning", "error", "command"
}

// PanelLayout stores panel position, size, and state
type PanelLayout struct {
	PosX      float32 `json:"pos_x"`
	PosY      float32 `json:"pos_y"`
	SizeX     float32 `json:"size_x"`
	SizeY     float32 `json:"size_y"`
	Collapsed bool    `json:"collapsed"`
}
