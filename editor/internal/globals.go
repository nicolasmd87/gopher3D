package editor

import (
	"Gopher3D/editor/platforms"
	"Gopher3D/editor/renderers"
	"Gopher3D/internal/behaviour"
	"Gopher3D/internal/engine"
	"Gopher3D/internal/renderer"
	"time"
)

var (
	Eng           *engine.Gopher
	Platform      *platforms.GLFW
	ImguiRenderer *renderers.OpenGL3

	selectedModelIndex      = -1
	selectedLightIndex      = -1
	selectedGameObjectIndex = -1
	selectedType            = ""
	ShowHierarchy           = true
	ShowInspector           = true
	ShowDemoWindow          = false
	ShowAddModel            = false
	ShowAddLight            = false
	ShowFileExplorer        = true
	ShowConsole             = true
	ShowStyleEditor         = false
	ShowAdvancedRender      = false
	ShowSceneSettings       = false
	ShowGizmos              = true

	ShowAddWater      = false
	ShowAddVoxel      = false
	ShowScriptBrowser = false

	// Script browser state
	scriptBrowserTarget      *behaviour.GameObject
	scriptBrowserModelTarget *renderer.Model

	lastFrameTime = time.Now()
	frameCount    = 0
	fps           = 0.0
	fpsUpdateTime = time.Now()

	ImguiInitialized   = false
	SceneSetup         = false
	firstFrameComplete = false

	currentDirectory        = "."
	selectedFilePath        = ""
	fileExplorerSearch      = ""
	fileExplorerPathHistory = []string{}

	consoleLines      = []ConsoleEntry{}
	consoleInput      = ""
	consoleAutoScroll = true
	maxConsoleLines   = 500

	currentScenePath = ""
	sceneModified    = false

	modelNameEditBuffer = make(map[int]string)

	currentSkyboxPath = ""
	skyboxTexturePath = ""
	skyboxColorMode   = true
	skyboxSolidColor  = [3]float32{0.4, 0.6, 0.9}

	instanceModelOnAdd = false
	instanceCount      = 1

	availableModels = []struct {
		Name string
		Path string
	}{}

	configPath = "editor_config.json"

	hierarchyLayout      PanelLayout
	inspectorLayout      PanelLayout
	fileExplorerLayout   PanelLayout
	consoleLayout        PanelLayout
	sceneSettingsLayout  PanelLayout
	advancedRenderLayout PanelLayout

	layoutsInitialized = false

	activeWaterSim *WaterSimulation

	globalAdvancedRenderingEnabled = true

	scriptSearchText = ""
	newScriptName    = ""

	modelToGameObject = make(map[*renderer.Model]*behaviour.GameObject)

	SavedStyleColors   StyleColors
	StyleColorsApplied = false

	windowBorderR = float32(0.0)
	windowBorderG = float32(0.0)
	windowBorderB = float32(0.0)

	// Camera management
	SceneCameras        []*renderer.Camera
	selectedCameraIndex = -1
	ShowAddCamera       = false
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
