package main

import (
	"Gopher3D/internal/renderer"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// EditorConfig stores all editor settings
type EditorConfig struct {
	// Panel visibility
	ShowHierarchy      bool `json:"show_hierarchy"`
	ShowInspector      bool `json:"show_inspector"`
	ShowFileExplorer   bool `json:"show_file_explorer"`
	ShowConsole        bool `json:"show_console"`
	ShowAdvancedRender bool `json:"show_advanced_render"`
	ShowSceneSettings  bool `json:"show_scene_settings"`

	// Panel layouts
	HierarchyLayout      PanelLayout `json:"hierarchy_layout"`
	InspectorLayout      PanelLayout `json:"inspector_layout"`
	FileExplorerLayout   PanelLayout `json:"file_explorer_layout"`
	ConsoleLayout        PanelLayout `json:"console_layout"`
	SceneSettingsLayout  PanelLayout `json:"scene_settings_layout"`
	AdvancedRenderLayout PanelLayout `json:"advanced_render_layout"`

	// Rendering settings
	ClearColorR    float32 `json:"clear_color_r"`
	ClearColorG    float32 `json:"clear_color_g"`
	ClearColorB    float32 `json:"clear_color_b"`
	WireframeMode  bool    `json:"wireframe_mode"`
	FrustumCulling bool    `json:"frustum_culling"`
	FaceCulling    bool    `json:"face_culling"`
	DepthTesting   bool    `json:"depth_testing"`

	// Skybox
	SkyboxColorMode bool    `json:"skybox_color_mode"`
	SkyboxSolidR    float32 `json:"skybox_solid_r"`
	SkyboxSolidG    float32 `json:"skybox_solid_g"`
	SkyboxSolidB    float32 `json:"skybox_solid_b"`
	SkyboxPath      string  `json:"skybox_path"`

	// Model loading
	InstanceOnAdd        bool `json:"instance_on_add"`
	DefaultInstanceCount int  `json:"default_instance_count"`
}

// Load editor configuration from file
func loadConfig() {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Println("No config file found, using defaults")
		saveConfig() // Create default config file
		return
	}

	var config EditorConfig
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Printf("Error parsing config: %v\n", err)
		return
	}

	// Apply config to editor state
	showHierarchy = config.ShowHierarchy
	showInspector = config.ShowInspector
	showFileExplorer = config.ShowFileExplorer
	showConsole = config.ShowConsole
	showAdvancedRender = config.ShowAdvancedRender
	showSceneSettings = config.ShowSceneSettings

	if eng != nil && eng.GetRenderer() != nil {
		if openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer); ok {
			openglRenderer.ClearColorR = config.ClearColorR
			openglRenderer.ClearColorG = config.ClearColorG
			openglRenderer.ClearColorB = config.ClearColorB
		}
	}

	renderer.Debug = config.WireframeMode
	renderer.FrustumCullingEnabled = config.FrustumCulling
	renderer.FaceCullingEnabled = config.FaceCulling
	renderer.DepthTestEnabled = config.DepthTesting

	skyboxColorMode = config.SkyboxColorMode
	skyboxSolidColor[0] = config.SkyboxSolidR
	skyboxSolidColor[1] = config.SkyboxSolidG
	skyboxSolidColor[2] = config.SkyboxSolidB
	currentSkyboxPath = config.SkyboxPath

	instanceModelOnAdd = config.InstanceOnAdd
	if config.DefaultInstanceCount > 0 {
		instanceCount = config.DefaultInstanceCount
	}

	// Load panel layouts
	hierarchyLayout = config.HierarchyLayout
	inspectorLayout = config.InspectorLayout
	fileExplorerLayout = config.FileExplorerLayout
	consoleLayout = config.ConsoleLayout
	sceneSettingsLayout = config.SceneSettingsLayout
	advancedRenderLayout = config.AdvancedRenderLayout

	fmt.Println("✓ Editor config loaded")
}

// Save editor configuration to file
func saveConfig() {
	config := EditorConfig{
		ShowHierarchy:      showHierarchy,
		ShowInspector:      showInspector,
		ShowFileExplorer:   showFileExplorer,
		ShowConsole:        showConsole,
		ShowAdvancedRender: showAdvancedRender,
		ShowSceneSettings:  showSceneSettings,

		HierarchyLayout:      hierarchyLayout,
		InspectorLayout:      inspectorLayout,
		FileExplorerLayout:   fileExplorerLayout,
		ConsoleLayout:        consoleLayout,
		SceneSettingsLayout:  sceneSettingsLayout,
		AdvancedRenderLayout: advancedRenderLayout,

		WireframeMode:  renderer.Debug,
		FrustumCulling: renderer.FrustumCullingEnabled,
		FaceCulling:    renderer.FaceCullingEnabled,
		DepthTesting:   renderer.DepthTestEnabled,

		SkyboxColorMode: skyboxColorMode,
		SkyboxSolidR:    skyboxSolidColor[0],
		SkyboxSolidG:    skyboxSolidColor[1],
		SkyboxSolidB:    skyboxSolidColor[2],
		SkyboxPath:      currentSkyboxPath,

		InstanceOnAdd:        instanceModelOnAdd,
		DefaultInstanceCount: instanceCount,
	}

	if eng != nil && eng.GetRenderer() != nil {
		if openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer); ok {
			config.ClearColorR = openglRenderer.ClearColorR
			config.ClearColorG = openglRenderer.ClearColorG
			config.ClearColorB = openglRenderer.ClearColorB
		}
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling config: %v\n", err)
		return
	}

	if err := ioutil.WriteFile(configPath, data, 0644); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		return
	}
}


