package editor

import (
	"Gopher3D/internal/renderer"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type StyleColors struct {
	BorderR       float32 `json:"border_r"`
	BorderG       float32 `json:"border_g"`
	BorderB       float32 `json:"border_b"`
	TitleActiveR  float32 `json:"title_active_r"`
	TitleActiveG  float32 `json:"title_active_g"`
	TitleActiveB  float32 `json:"title_active_b"`
	HeaderR       float32 `json:"header_r"`
	HeaderG       float32 `json:"header_g"`
	HeaderB       float32 `json:"header_b"`
	ButtonHoverR  float32 `json:"button_hover_r"`
	ButtonHoverG  float32 `json:"button_hover_g"`
	ButtonHoverB  float32 `json:"button_hover_b"`
	WindowBorderR float32 `json:"window_border_r"`
	WindowBorderG float32 `json:"window_border_g"`
	WindowBorderB float32 `json:"window_border_b"`
}

type EditorConfig struct {
	ShowHierarchy      bool `json:"show_hierarchy"`
	ShowInspector      bool `json:"show_inspector"`
	ShowFileExplorer   bool `json:"show_file_explorer"`
	ShowConsole        bool `json:"show_console"`
	ShowAdvancedRender bool `json:"show_advanced_render"`
	ShowSceneSettings  bool `json:"show_scene_settings"`

	HierarchyLayout      PanelLayout `json:"hierarchy_layout"`
	InspectorLayout      PanelLayout `json:"inspector_layout"`
	FileExplorerLayout   PanelLayout `json:"file_explorer_layout"`
	ConsoleLayout        PanelLayout `json:"console_layout"`
	SceneSettingsLayout  PanelLayout `json:"scene_settings_layout"`
	AdvancedRenderLayout PanelLayout `json:"advanced_render_layout"`

	ClearColorR    float32 `json:"clear_color_r"`
	ClearColorG    float32 `json:"clear_color_g"`
	ClearColorB    float32 `json:"clear_color_b"`
	WireframeMode  bool    `json:"wireframe_mode"`
	FrustumCulling bool    `json:"frustum_culling"`
	FaceCulling    bool    `json:"face_culling"`
	DepthTesting   bool    `json:"depth_testing"`

	SkyboxColorMode bool    `json:"skybox_color_mode"`
	SkyboxSolidR    float32 `json:"skybox_solid_r"`
	SkyboxSolidG    float32 `json:"skybox_solid_g"`
	SkyboxSolidB    float32 `json:"skybox_solid_b"`
	SkyboxPath      string  `json:"skybox_path"`

	InstanceOnAdd        bool `json:"instance_on_add"`
	DefaultInstanceCount int  `json:"default_instance_count"`

	RecentProjects []Project   `json:"recent_projects,omitempty"`
	StyleColors    StyleColors `json:"style_colors,omitempty"`

	WindowWidth     int32 `json:"window_width"`
	WindowHeight    int32 `json:"window_height"`
	WindowMaximized bool  `json:"window_maximized"`
}

// LoadConfig loads editor configuration from file
func LoadConfig() {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Println("No config file found, using defaults")
		return
	}

	var config EditorConfig
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Printf("Error parsing config: %v\n", err)
		return
	}

	ShowHierarchy = config.ShowHierarchy
	ShowInspector = config.ShowInspector
	ShowFileExplorer = config.ShowFileExplorer
	ShowConsole = config.ShowConsole
	ShowAdvancedRender = config.ShowAdvancedRender
	ShowSceneSettings = config.ShowSceneSettings

	if Eng != nil && Eng.GetRenderer() != nil {
		if openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer); ok {
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

	if len(config.RecentProjects) > 0 {
		recentProjects = config.RecentProjects
	}

	// Only apply layout if it has non-zero size (valid saved layout)
	if config.HierarchyLayout.SizeX > 0 && config.HierarchyLayout.SizeY > 0 {
		hierarchyLayout = config.HierarchyLayout
	}
	if config.InspectorLayout.SizeX > 0 && config.InspectorLayout.SizeY > 0 {
		inspectorLayout = config.InspectorLayout
	}
	if config.FileExplorerLayout.SizeX > 0 && config.FileExplorerLayout.SizeY > 0 {
		fileExplorerLayout = config.FileExplorerLayout
	}
	if config.ConsoleLayout.SizeX > 0 && config.ConsoleLayout.SizeY > 0 {
		consoleLayout = config.ConsoleLayout
	}
	if config.SceneSettingsLayout.SizeX > 0 && config.SceneSettingsLayout.SizeY > 0 {
		sceneSettingsLayout = config.SceneSettingsLayout
	}
	if config.AdvancedRenderLayout.SizeX > 0 && config.AdvancedRenderLayout.SizeY > 0 {
		advancedRenderLayout = config.AdvancedRenderLayout
	}

	SavedStyleColors = config.StyleColors

	windowBorderR = config.StyleColors.WindowBorderR
	windowBorderG = config.StyleColors.WindowBorderG
	windowBorderB = config.StyleColors.WindowBorderB

	fmt.Println("✓ Editor config loaded")
}

// SaveConfig saves editor configuration to file
func SaveConfig() {
	config := EditorConfig{
		ShowHierarchy:      ShowHierarchy,
		ShowInspector:      ShowInspector,
		ShowFileExplorer:   ShowFileExplorer,
		ShowConsole:        ShowConsole,
		ShowAdvancedRender: ShowAdvancedRender,
		ShowSceneSettings:  ShowSceneSettings,

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
		RecentProjects:       recentProjects,
		StyleColors:          getCurrentStyleColors(),
	}

	if Eng != nil && Eng.GetRenderer() != nil {
		if openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer); ok {
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
