package main

import (
	"Gopher3D/internal/engine"
	"Gopher3D/internal/loader"
	"Gopher3D/internal/renderer"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"Gopher3D/editor/platforms"
	"Gopher3D/editor/renderers"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/sqweek/dialog"
	mgl "github.com/go-gl/mathgl/mgl32"
)

var (
	eng                *engine.Gopher
	platform           *platforms.GLFW
	imguiRenderer      *renderers.OpenGL3

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
	
	// FPS tracking
	lastFrameTime      = time.Now()
	frameCount         = 0
	fps                = 0.0
	fpsUpdateTime      = time.Now()

	imguiInitialized = false
	sceneSetup       = false

	// File explorer state
	currentDirectory  = "../examples/resources"
	selectedFilePath  = ""

	// Console state
	consoleLines      = []ConsoleEntry{}
	consoleInput      = ""
	consoleAutoScroll = true
	maxConsoleLines   = 500
	
	// Scene management
	currentScenePath = ""
	sceneModified    = false
	
	// Skybox management
	currentSkyboxPath  = ""
	skyboxColorMode    = true
	skyboxSolidColor   = [3]float32{0.4, 0.6, 0.9} // Default sky blue
	
	// Model instancing
	instanceModelOnAdd = false
	instanceCount      = 1

	// Available models to load (paths relative to editor/ directory)
	availableModels = []struct {
		Name string
		Path string
	}{
		{"Cube", "../examples/resources/obj/Cube.obj"},
		{"Sphere", "../examples/resources/obj/Sphere.obj"},
		{"Low-Poly Sphere", "../examples/resources/obj/Sphere_Low.obj"},
		{"Triangle", "../examples/resources/obj/Triangle.obj"},
		{"Earth (Textured)", "../examples/resources/obj/Earth/Earth.obj"},
	}
	
	// Editor config
	configPath = "editor_config.json"
)

type ConsoleEntry struct {
	Message string
	Type    string // "info", "warning", "error", "command"
}

// EditorConfig stores all editor settings
type EditorConfig struct {
	// Panel visibility
	ShowHierarchy      bool    `json:"show_hierarchy"`
	ShowInspector      bool    `json:"show_inspector"`
	ShowFileExplorer   bool    `json:"show_file_explorer"`
	ShowConsole        bool    `json:"show_console"`
	ShowAdvancedRender bool    `json:"show_advanced_render"`
	ShowSceneSettings  bool    `json:"show_scene_settings"`
	
	// Rendering settings
	ClearColorR        float32 `json:"clear_color_r"`
	ClearColorG        float32 `json:"clear_color_g"`
	ClearColorB        float32 `json:"clear_color_b"`
	WireframeMode      bool    `json:"wireframe_mode"`
	FrustumCulling     bool    `json:"frustum_culling"`
	FaceCulling        bool    `json:"face_culling"`
	DepthTesting       bool    `json:"depth_testing"`
	
	// Skybox
	SkyboxColorMode    bool    `json:"skybox_color_mode"`
	SkyboxSolidR       float32 `json:"skybox_solid_r"`
	SkyboxSolidG       float32 `json:"skybox_solid_g"`
	SkyboxSolidB       float32 `json:"skybox_solid_b"`
	SkyboxPath         string  `json:"skybox_path"`
	
	// Model loading
	InstanceOnAdd      bool    `json:"instance_on_add"`
	DefaultInstanceCount int   `json:"default_instance_count"`
}

// Load editor configuration from file
func loadConfig() {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		// File doesn't exist or can't be read - use defaults
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
		
		WireframeMode:    renderer.Debug,
		FrustumCulling:   renderer.FrustumCullingEnabled,
		FaceCulling:      renderer.FaceCullingEnabled,
		DepthTesting:     renderer.DepthTestEnabled,
		
		SkyboxColorMode:  skyboxColorMode,
		SkyboxSolidR:     skyboxSolidColor[0],
		SkyboxSolidG:     skyboxSolidColor[1],
		SkyboxSolidB:     skyboxSolidColor[2],
		SkyboxPath:       currentSkyboxPath,
		
		InstanceOnAdd:         instanceModelOnAdd,
		DefaultInstanceCount:  instanceCount,
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

func main() {
	runtime.LockOSThread()

	fmt.Println("===========================================")
	fmt.Println("   Gopher3D Editor with ImGui")
	fmt.Println("===========================================")

	// Create ImGui context
	context := imgui.CreateContext(nil)
	defer context.Destroy()
	defer saveConfig() // Save editor settings on exit

	// Create engine
	eng = engine.NewGopher(engine.OPENGL)
	eng.Width = 1920
	eng.Height = 1080

	// Set render callback to handle ImGui initialization and rendering on main thread
	eng.SetOnRenderCallback(func(deltaTime float64) {
		// Initialize ImGui on first render (when window exists and we're on main thread)
		if !imguiInitialized && eng.GetWindow() != nil {
			initializeImGui()
		}

		// Setup scene once ImGui is ready
		if imguiInitialized && !sceneSetup {
			setupEditorScene()
			sceneSetup = true
		}

		// Control camera input based on ImGui state
		if imguiInitialized {
			io := imgui.CurrentIO()
			// Disable camera input when ImGui wants keyboard or mouse
			eng.EnableCameraInput = !io.WantCaptureKeyboard() && !io.WantCaptureMouse()
		}

		// Render ImGui UI
		if imguiInitialized {
			renderImGuiFrame()
		}
	})

	fmt.Println("Starting engine...")
	// Start engine (creates window inside Render())
	eng.Render(50, 50)
}

func initializeImGui() {
	fmt.Println("Initializing ImGui on main thread...")

	window := eng.GetWindow()
	io := imgui.CurrentIO()

	// Create GLFW platform
	var err error
	platform, err = platforms.NewGLFWFromExistingWindow(window, io)
	if err != nil {
		fmt.Printf("ERROR: Failed to create GLFW platform: %v\n", err)
		return
	}

	// Create OpenGL3 renderer (this creates OpenGL objects, must be on main thread!)
	imguiRenderer, err = renderers.NewOpenGL3(io)
	if err != nil {
		fmt.Printf("ERROR: Failed to create OpenGL3 renderer: %v\n", err)
		return
	}

	// Apply dark theme
	applyDarkTheme()

	imguiInitialized = true
	fmt.Println("✓ ImGui initialized successfully!")
}

func renderImGuiFrame() {
	if platform == nil || imguiRenderer == nil {
		return
	}

	// New frame
	platform.NewFrame()
	imgui.NewFrame()

	// Render UI
	renderEditorUI()

	// Render
	imgui.Render()
	displaySize := platform.DisplaySize()
	framebufferSize := platform.FramebufferSize()
	imguiRenderer.Render(displaySize, framebufferSize, imgui.RenderedDrawData())
}

func renderEditorUI() {
	openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
	if !ok {
		return
	}

	models := openglRenderer.GetModels()

	// Main menu bar
	if imgui.BeginMainMenuBar() {
		if imgui.BeginMenu("File") {
			if imgui.MenuItem("New Scene") {
				newScene()
			}
			if imgui.MenuItem("Save Scene...") {
				saveScene()
			}
			if imgui.MenuItem("Load Scene...") {
				loadScene()
			}
			imgui.Separator()
			if imgui.MenuItem("Import Model...") {
				// Open native file browser
				filename, err := dialog.File().
					SetStartDir("../examples/resources/obj").
					Filter("3D Models", "obj").
					Title("Import Model").
					Load()
				if err == nil && filename != "" {
					// Extract name from path
					name := getFileNameFromPath(filename)
					addModelToScene(filename, name)
				}
			}
			if imgui.MenuItem("Import Texture...") {
				// Check if model is selected
				openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
				if ok {
					models := openglRenderer.GetModels()
					if selectedModelIndex >= 0 && selectedModelIndex < len(models) {
						filename, err := dialog.File().
							SetStartDir("../examples/resources/textures").
							Filter("Images", "png", "jpg", "jpeg").
							Title("Import Texture").
							Load()
						if err == nil && filename != "" {
							loadTextureToSelected(filename)
						}
					} else {
						fmt.Println("Please select a model first before loading a texture")
					}
				}
			}
			imgui.EndMenu()
		}
		if imgui.BeginMenu("Add") {
			if imgui.MenuItem("Model...") {
				showAddModel = true
			}
			if imgui.MenuItem("Light...") {
				showAddLight = true
			}
			imgui.EndMenu()
		}
		if imgui.BeginMenu("View") {
			imgui.Text("Core Panels:")
			if imgui.MenuItemV("Scene Hierarchy", "", showHierarchy, true) {
				showHierarchy = !showHierarchy
				saveConfig()
			}
			if imgui.MenuItemV("Inspector", "", showInspector, true) {
				showInspector = !showInspector
				saveConfig()
			}
			imgui.Separator()
			imgui.Text("Utility Panels:")
			if imgui.MenuItemV("File Explorer", "", showFileExplorer, true) {
				showFileExplorer = !showFileExplorer
				saveConfig()
			}
			if imgui.MenuItemV("Console", "", showConsole, true) {
				showConsole = !showConsole
				saveConfig()
			}
			imgui.Separator()
			imgui.Text("Settings Panels:")
			if imgui.MenuItemV("Scene Settings", "", showSceneSettings, true) {
				showSceneSettings = !showSceneSettings
				saveConfig()
			}
			if imgui.MenuItemV("Style Editor", "", showStyleEditor, true) {
				showStyleEditor = !showStyleEditor
			}
			if imgui.MenuItemV("Advanced Rendering", "", showAdvancedRender, true) {
				showAdvancedRender = !showAdvancedRender
				saveConfig()
			}
			if imgui.MenuItemV("ImGui Demo", "", showDemoWindow, true) {
				showDemoWindow = !showDemoWindow
			}
			imgui.EndMenu()
		}
		
		// FPS Display in menu bar (right side)
		updateFPS()
		menuBarSize := imgui.WindowSize()
		fpsText := fmt.Sprintf("FPS: %.0f", fps)
		fpsTextSize := imgui.CalcTextSize(fpsText, false, 0)
		imgui.SetCursorPos(imgui.Vec2{X: menuBarSize.X - fpsTextSize.X - 10, Y: imgui.CursorPosY()})
		imgui.Text(fpsText)
		
		imgui.EndMainMenuBar()
	}

	// Add Model Dialog
	if showAddModel {
		renderAddModelDialog()
	}

	// Add Light Dialog
	if showAddLight {
		renderAddLightDialog()
	}

	// File Explorer (Bottom Left)
	if showFileExplorer {
		imgui.SetNextWindowPosV(imgui.Vec2{X: 10, Y: float32(eng.Height) - 360}, imgui.ConditionFirstUseEver, imgui.Vec2{})
		imgui.SetNextWindowSizeV(imgui.Vec2{X: 300, Y: 150}, imgui.ConditionFirstUseEver)
		renderFileExplorer()
	}

	// Console (Bottom Middle)
	if showConsole {
		imgui.SetNextWindowPosV(imgui.Vec2{X: 320, Y: float32(eng.Height) - 360}, imgui.ConditionFirstUseEver, imgui.Vec2{})
		imgui.SetNextWindowSizeV(imgui.Vec2{X: float32(eng.Width) - 680, Y: 350}, imgui.ConditionFirstUseEver)
		renderConsole()
	}
	
	// Style Editor for easy color customization
	if showStyleEditor {
		imgui.SetNextWindowPosV(imgui.Vec2{X: float32(eng.Width) - 520, Y: 30}, imgui.ConditionFirstUseEver, imgui.Vec2{})
		imgui.SetNextWindowSizeV(imgui.Vec2{X: 500, Y: 600}, imgui.ConditionFirstUseEver)
		imgui.Begin("Style Editor")
		imgui.Text("Customize Editor Colors")
		imgui.Separator()
		
		style := imgui.CurrentStyle()
		
		imgui.Text("Key Colors:")
		imgui.Separator()
		
		// Border color
		borderColor := style.Color(imgui.StyleColorBorder)
		borderColorVec := [3]float32{borderColor.X, borderColor.Y, borderColor.Z}
		if imgui.ColorEdit3V("Borders & Separators", &borderColorVec, 0) {
			newColor := imgui.Vec4{X: borderColorVec[0], Y: borderColorVec[1], Z: borderColorVec[2], W: 1.0}
			style.SetColor(imgui.StyleColorBorder, newColor)
			style.SetColor(imgui.StyleColorSeparator, newColor)
		}
		
		// Active title color
		titleBgActive := style.Color(imgui.StyleColorTitleBgActive)
		titleColorVec := [3]float32{titleBgActive.X, titleBgActive.Y, titleBgActive.Z}
		if imgui.ColorEdit3V("Active Window Title", &titleColorVec, 0) {
			newColor := imgui.Vec4{X: titleColorVec[0], Y: titleColorVec[1], Z: titleColorVec[2], W: 1.0}
			style.SetColor(imgui.StyleColorTitleBgActive, newColor)
		}
		
		// Header/Selection color
		headerColor := style.Color(imgui.StyleColorHeader)
		headerColorVec := [3]float32{headerColor.X, headerColor.Y, headerColor.Z}
		if imgui.ColorEdit3V("Selected Items", &headerColorVec, 0) {
			newColor := imgui.Vec4{X: headerColorVec[0], Y: headerColorVec[1], Z: headerColorVec[2], W: 0.4}
			style.SetColor(imgui.StyleColorHeader, newColor)
			style.SetColor(imgui.StyleColorHeaderActive, imgui.Vec4{X: headerColorVec[0], Y: headerColorVec[1], Z: headerColorVec[2], W: 1.0})
		}
		
		// Button hover color
		buttonHover := style.Color(imgui.StyleColorButtonHovered)
		buttonColorVec := [3]float32{buttonHover.X, buttonHover.Y, buttonHover.Z}
		if imgui.ColorEdit3V("Button Hover", &buttonColorVec, 0) {
			style.SetColor(imgui.StyleColorButtonHovered, imgui.Vec4{X: buttonColorVec[0], Y: buttonColorVec[1], Z: buttonColorVec[2], W: 0.6})
		}
		
		// Tab active color
		tabActive := style.Color(imgui.StyleColorTabActive)
		tabColorVec := [3]float32{tabActive.X, tabActive.Y, tabActive.Z}
		if imgui.ColorEdit3V("Active Tab", &tabColorVec, 0) {
			style.SetColor(imgui.StyleColorTabActive, imgui.Vec4{X: tabColorVec[0], Y: tabColorVec[1], Z: tabColorVec[2], W: 1.0})
		}
		
		imgui.Separator()
		imgui.Text("Quick Presets:")
		if imgui.Button("Go Cyan") {
			applyDarkTheme() // Reapply our Go cyan theme
		}
		imgui.SameLine()
		if imgui.Button("Reset to Dark") {
			imgui.StyleColorsDark()
		}
		
		imgui.End()
	}
	
	// Advanced Rendering Options
	if showAdvancedRender {
		imgui.SetNextWindowPosV(imgui.Vec2{X: float32(eng.Width)/2 - 200, Y: 100}, imgui.ConditionFirstUseEver, imgui.Vec2{})
		imgui.SetNextWindowSizeV(imgui.Vec2{X: 400, Y: 450}, imgui.ConditionFirstUseEver)
		imgui.Begin("Advanced Rendering")
		imgui.Text("Rendering Options & Techniques")
		imgui.Separator()
		
		// Wireframe Mode
		if imgui.Checkbox("Wireframe Mode", &renderer.Debug) {
			logToConsole(fmt.Sprintf("Wireframe: %v", renderer.Debug), "info")
		}
		
		// Culling Options
		imgui.Separator()
		imgui.Text("Culling:")
		if imgui.Checkbox("Frustum Culling", &renderer.FrustumCullingEnabled) {
			logToConsole(fmt.Sprintf("Frustum Culling: %v", renderer.FrustumCullingEnabled), "info")
		}
		if imgui.Checkbox("Face Culling", &renderer.FaceCullingEnabled) {
			logToConsole(fmt.Sprintf("Face Culling: %v", renderer.FaceCullingEnabled), "info")
		}
		
		// Depth Test
		imgui.Separator()
		if imgui.Checkbox("Depth Testing", &renderer.DepthTestEnabled) {
			logToConsole(fmt.Sprintf("Depth Test: %v", renderer.DepthTestEnabled), "info")
		}
		
		// Background Color
		imgui.Separator()
		imgui.Text("Background Clear Color:")
		if openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer); ok {
			bgColor := [3]float32{openglRenderer.ClearColorR, openglRenderer.ClearColorG, openglRenderer.ClearColorB}
			if imgui.ColorEdit3V("Clear Color", &bgColor, 0) {
				openglRenderer.ClearColorR = bgColor[0]
				openglRenderer.ClearColorG = bgColor[1]
				openglRenderer.ClearColorB = bgColor[2]
				saveConfig()
			}
		}
		
		// Transparency Rendering Info
		imgui.Separator()
		imgui.Text("Transparency Rendering:")
		imgui.PushTextWrapPos()
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.6, Y: 0.6, Z: 0.6, W: 1.0})
		imgui.Text("Materials with Alpha < 0.99 automatically disable face culling and depth writes for proper transparency. Both front and back faces are rendered.")
		imgui.PopStyleColor()
		imgui.PopTextWrapPos()
		
		// Performance Stats
		imgui.Separator()
		imgui.Text("Performance Statistics:")
		imgui.Text(fmt.Sprintf("FPS: %.0f", fps))
		if openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer); ok {
			models := openglRenderer.GetModels()
			lights := openglRenderer.GetLights()
			imgui.Text(fmt.Sprintf("Models: %d", len(models)))
			imgui.Text(fmt.Sprintf("Lights: %d", len(lights)))
			imgui.Text(fmt.Sprintf("Draw Calls: %d", openglRenderer.GetDrawCalls()))
			
			totalInstances := 0
			for _, model := range models {
				if model.IsInstanced {
					totalInstances += model.InstanceCount
				}
			}
			imgui.Text(fmt.Sprintf("Instances: %d", totalInstances))
		}
		
		imgui.Separator()
		imgui.PushTextWrapPos()
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.6, Y: 0.6, Z: 0.6, W: 1.0})
		imgui.Text("Note: Advanced rendering techniques like instancing, PBR shading, and shadow mapping are already integrated into the engine.")
		imgui.PopStyleColor()
		imgui.PopTextWrapPos()
		
		imgui.End()
	}
	
	// Scene Settings (Skybox, Environment)
	if showSceneSettings {
		imgui.SetNextWindowPosV(imgui.Vec2{X: float32(eng.Width) - 380, Y: float32(eng.Height) - 430}, imgui.ConditionFirstUseEver, imgui.Vec2{})
		imgui.SetNextWindowSizeV(imgui.Vec2{X: 370, Y: 420}, imgui.ConditionFirstUseEver)
		imgui.Begin("Scene Settings")
		
		// Skybox Section
		if imgui.CollapsingHeaderV("Skybox / Background", imgui.TreeNodeFlagsDefaultOpen) {
			imgui.Text("Background Mode:")
			if imgui.RadioButton("Solid Color", skyboxColorMode) {
				skyboxColorMode = true
				saveConfig()
			}
			imgui.SameLine()
			if imgui.RadioButton("Skybox Image", !skyboxColorMode) {
				skyboxColorMode = false
				saveConfig()
			}
			
			imgui.Separator()
			
			if skyboxColorMode {
				// Solid color mode
				imgui.Text("Sky Color:")
				if imgui.ColorEdit3V("##skycolor", &skyboxSolidColor, 0) {
					// Apply solid color skybox
					colorStr := fmt.Sprintf("solid:%.2f,%.2f,%.2f", 
						skyboxSolidColor[0], skyboxSolidColor[1], skyboxSolidColor[2])
					if err := eng.SetSkybox(colorStr); err == nil {
						eng.UpdateSkyboxColor(skyboxSolidColor[0], skyboxSolidColor[1], skyboxSolidColor[2])
						currentSkyboxPath = colorStr
						logToConsole(fmt.Sprintf("Skybox color set: RGB(%.2f, %.2f, %.2f)", 
							skyboxSolidColor[0], skyboxSolidColor[1], skyboxSolidColor[2]), "info")
						saveConfig()
					} else {
						logToConsole(fmt.Sprintf("Failed to set skybox: %v", err), "error")
					}
				}
				
				// Quick color presets
				imgui.Separator()
				imgui.Text("Presets:")
				if imgui.Button("Sky Blue") {
					skyboxSolidColor = [3]float32{0.4, 0.6, 0.9}
					colorStr := "solid:0.4,0.6,0.9"
					eng.SetSkybox(colorStr)
					eng.UpdateSkyboxColor(0.4, 0.6, 0.9)
					currentSkyboxPath = colorStr
					saveConfig()
				}
				imgui.SameLine()
				if imgui.Button("Sunset") {
					skyboxSolidColor = [3]float32{0.9, 0.5, 0.3}
					colorStr := "solid:0.9,0.5,0.3"
					eng.SetSkybox(colorStr)
					eng.UpdateSkyboxColor(0.9, 0.5, 0.3)
					currentSkyboxPath = colorStr
					saveConfig()
				}
				imgui.SameLine()
				if imgui.Button("Night") {
					skyboxSolidColor = [3]float32{0.05, 0.05, 0.15}
					colorStr := "solid:0.05,0.05,0.15"
					eng.SetSkybox(colorStr)
					eng.UpdateSkyboxColor(0.05, 0.05, 0.15)
					currentSkyboxPath = colorStr
					saveConfig()
				}
			} else {
				// Skybox image mode
				if currentSkyboxPath != "" && !strings.HasPrefix(currentSkyboxPath, "solid:") {
					imgui.Text("Current:")
					imgui.SameLine()
					imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.0, Y: 0.678, Z: 0.847, W: 1.0})
					imgui.Text(filepath.Base(currentSkyboxPath))
					imgui.PopStyleColor()
				} else {
					imgui.Text("No skybox loaded")
				}
				
				if imgui.Button("Load Skybox Image...") {
					filename, err := dialog.File().
						SetStartDir("../examples/resources/textures").
						Filter("Images", "png", "jpg", "jpeg", "hdr").
						Title("Load Skybox").
						Load()
					if err == nil && filename != "" {
						if err := eng.SetSkybox(filename); err == nil {
							currentSkyboxPath = filename
							logToConsole(fmt.Sprintf("Skybox loaded: %s", filepath.Base(filename)), "info")
							saveConfig()
						} else {
							logToConsole(fmt.Sprintf("Failed to load skybox: %v", err), "error")
						}
					}
				}
				
				imgui.PushTextWrapPos()
				imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.6, Y: 0.6, Z: 0.6, W: 1.0})
				imgui.Text("Supported: Single equirectangular image or cubemap textures")
				imgui.PopStyleColor()
				imgui.PopTextWrapPos()
			}
			
			imgui.Separator()
			if imgui.Button("Clear Skybox") {
				currentSkyboxPath = ""
				// Set to black background
				eng.SetSkybox("solid:0.0,0.0,0.0")
				logToConsole("Skybox cleared", "info")
				saveConfig()
			}
		}
		
		// Scene Info Section
		imgui.Separator()
		if imgui.CollapsingHeaderV("Scene Info", imgui.TreeNodeFlagsDefaultOpen) {
			if openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer); ok {
				models := openglRenderer.GetModels()
				lights := openglRenderer.GetLights()
				imgui.Text(fmt.Sprintf("Models: %d", len(models)))
				imgui.Text(fmt.Sprintf("Lights: %d", len(lights)))
				if currentScenePath != "" {
					imgui.Text("Scene: " + filepath.Base(currentScenePath))
				} else {
					imgui.Text("Scene: Unsaved")
				}
			}
		}
		
		imgui.End()
	}

	// Scene Hierarchy (Left Side, Top)
	if showHierarchy {
		imgui.SetNextWindowPosV(imgui.Vec2{X: 10, Y: 30}, imgui.ConditionFirstUseEver, imgui.Vec2{})
		imgui.SetNextWindowSizeV(imgui.Vec2{X: 300, Y: float32(eng.Height) - 420}, imgui.ConditionFirstUseEver)
		imgui.Begin("Scene Hierarchy")
		imgui.Text("Scene Objects:")
		imgui.Separator()

		// Models section
		if imgui.CollapsingHeaderV("[M] Models", imgui.TreeNodeFlagsDefaultOpen) {
			for i, model := range models {
				// CRITICAL: Push unique ID for each Selectable to ensure clicks work correctly
				imgui.PushID(fmt.Sprintf("model_%d", i))
				isSelected := selectedType == "model" && selectedModelIndex == i
				if imgui.SelectableV("  "+model.Name, isSelected, 0, imgui.Vec2{}) {
					selectedModelIndex = i
					selectedLightIndex = -1
					selectedType = "model"
				}
				// Double-click to focus camera on model
				if imgui.IsItemHovered() && imgui.IsMouseDoubleClicked(0) {
					focusCameraOnModel(model)
				}
				imgui.PopID()
			}
		}

		// Lights section
		lights := openglRenderer.GetLights()
		if imgui.CollapsingHeaderV("[L] Lights", imgui.TreeNodeFlagsDefaultOpen) {
			for i, light := range lights {
				// CRITICAL: Push unique ID for each Selectable to ensure clicks work correctly
				imgui.PushID(fmt.Sprintf("light_%d", i))
				icon := "[Dir]"
				if light.Mode == "point" {
					icon = "[Pnt]"
				}
				displayName := light.Name
				if displayName == "" {
					displayName = fmt.Sprintf("Light %d", i)
				}
				isSelected := selectedType == "light" && selectedLightIndex == i
				if imgui.SelectableV("  "+icon+" "+displayName, isSelected, 0, imgui.Vec2{}) {
					selectedLightIndex = i
					selectedModelIndex = -1
					selectedType = "light"
				}
				imgui.PopID()
			}
		}
		imgui.End()
	}

	// Inspector (Right Side, Top)
	if showInspector {
		imgui.SetNextWindowPosV(imgui.Vec2{X: float32(eng.Width) - 350, Y: 30}, imgui.ConditionFirstUseEver, imgui.Vec2{})
		imgui.SetNextWindowSizeV(imgui.Vec2{X: 340, Y: float32(eng.Height) - 420}, imgui.ConditionFirstUseEver)
		imgui.Begin("Inspector")

		if selectedType == "model" && selectedModelIndex >= 0 && selectedModelIndex < len(models) {
			model := models[selectedModelIndex]
			imgui.Text("Selected: " + model.Name)
			imgui.Separator()
			
			// Position (keyboard editable)
			imgui.Text("Position")
			posX, posY, posZ := model.Position.X(), model.Position.Y(), model.Position.Z()
			w := imgui.ContentRegionAvail().X
			imgui.PushItemWidth(w / 3.3)
			changed := false
			if imgui.DragFloatV("##posX", &posX, 0.5, 0, 0, "X: %.1f", 0) {
				changed = true
			}
			imgui.SameLine()
			if imgui.DragFloatV("##posY", &posY, 0.5, 0, 0, "Y: %.1f", 0) {
				changed = true
			}
			imgui.SameLine()
			if imgui.DragFloatV("##posZ", &posZ, 0.5, 0, 0, "Z: %.1f", 0) {
				changed = true
			}
			imgui.PopItemWidth()
			if changed {
				model.SetPosition(posX, posY, posZ)
				model.IsDirty = true
			}

			// Scale (keyboard editable)
			imgui.Text("Scale")
			scaleX, scaleY, scaleZ := model.Scale.X(), model.Scale.Y(), model.Scale.Z()
			imgui.PushItemWidth(w / 3.3)
			changed = false
			if imgui.DragFloatV("##scaleX", &scaleX, 0.05, 0, 0, "X: %.2f", 0) {
				changed = true
			}
			imgui.SameLine()
			if imgui.DragFloatV("##scaleY", &scaleY, 0.05, 0, 0, "Y: %.2f", 0) {
				changed = true
			}
			imgui.SameLine()
			if imgui.DragFloatV("##scaleZ", &scaleZ, 0.05, 0, 0, "Z: %.2f", 0) {
				changed = true
			}
			imgui.PopItemWidth()
			if changed {
				model.SetScale(scaleX, scaleY, scaleZ)
				model.IsDirty = true
			}

			// Material - Complete Properties
			if model.Material != nil {
				imgui.Separator()
				if imgui.CollapsingHeaderV("Material Properties", imgui.TreeNodeFlagsDefaultOpen) {
            // Diffuse Color
            diffuse := [3]float32{model.Material.DiffuseColor[0], model.Material.DiffuseColor[1], model.Material.DiffuseColor[2]}
            if imgui.ColorEdit3V("Diffuse Color", &diffuse, 0) {
                // Apply to main material
                model.SetDiffuseColor(diffuse[0], diffuse[1], diffuse[2])
                // Propagate to all material groups (multi-material models)
                if len(model.MaterialGroups) > 0 {
                    for i := range model.MaterialGroups {
                        if model.MaterialGroups[i].Material != nil {
                            model.MaterialGroups[i].Material.DiffuseColor = [3]float32{diffuse[0], diffuse[1], diffuse[2]}
                        }
                    }
                }
                model.IsDirty = true
            }
					
            // Specular Color
            specular := model.Material.SpecularColor
            if imgui.ColorEdit3V("Specular Color", &specular, 0) {
                model.Material.SpecularColor = specular
                if len(model.MaterialGroups) > 0 {
                    for i := range model.MaterialGroups {
                        if model.MaterialGroups[i].Material != nil {
                            model.MaterialGroups[i].Material.SpecularColor = specular
                        }
                    }
                }
                model.IsDirty = true
            }
					
            // Shininess
            shininess := model.Material.Shininess
            if imgui.SliderFloatV("Shininess", &shininess, 0.0, 128.0, "%.1f", 0) {
                model.Material.Shininess = shininess
                if len(model.MaterialGroups) > 0 {
                    for i := range model.MaterialGroups {
                        if model.MaterialGroups[i].Material != nil {
                            model.MaterialGroups[i].Material.Shininess = shininess
                        }
                    }
                }
                model.IsDirty = true
            }

					// PBR Properties
            metallic := model.Material.Metallic
            if imgui.SliderFloatV("Metallic", &metallic, 0.0, 1.0, "%.2f", 0) {
                model.SetMaterialPBR(metallic, model.Material.Roughness)
                if len(model.MaterialGroups) > 0 {
                    for i := range model.MaterialGroups {
                        if model.MaterialGroups[i].Material != nil {
                            model.MaterialGroups[i].Material.Metallic = metallic
                        }
                    }
                }
                model.IsDirty = true
            }

            roughness := model.Material.Roughness
            if imgui.SliderFloatV("Roughness", &roughness, 0.0, 1.0, "%.2f", 0) {
                model.SetMaterialPBR(model.Material.Metallic, roughness)
                if len(model.MaterialGroups) > 0 {
                    for i := range model.MaterialGroups {
                        if model.MaterialGroups[i].Material != nil {
                            model.MaterialGroups[i].Material.Roughness = roughness
                        }
                    }
                }
                model.IsDirty = true
            }
					
					// Exposure
            exposure := model.Material.Exposure
            if imgui.SliderFloatV("Exposure", &exposure, 0.1, 5.0, "%.2f", 0) {
                model.Material.Exposure = exposure
                if len(model.MaterialGroups) > 0 {
                    for i := range model.MaterialGroups {
                        if model.MaterialGroups[i].Material != nil {
                            model.MaterialGroups[i].Material.Exposure = exposure
                        }
                    }
                }
                model.IsDirty = true
            }
					
					// Alpha (Transparency)
            alpha := model.Material.Alpha
            if imgui.SliderFloatV("Alpha", &alpha, 0.0, 1.0, "%.2f", 0) {
                model.Material.Alpha = alpha
                if len(model.MaterialGroups) > 0 {
                    for i := range model.MaterialGroups {
                        if model.MaterialGroups[i].Material != nil {
                            model.MaterialGroups[i].Material.Alpha = alpha
                        }
                    }
                }
                model.IsDirty = true
            }
					
					// Transparency note
					if alpha < 0.99 {
						imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.9, Y: 0.7, Z: 0.3, W: 1.0})
						imgui.Text("⚠ Transparency active")
						imgui.PopStyleColor()
						imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.6, Y: 0.6, Z: 0.6, W: 1.0})
						imgui.PushTextWrapPos()
						imgui.Text("Note: Some artifacts may appear due to render order. For best results, keep Alpha at 1.0 or 0.0.")
						imgui.PopTextWrapPos()
						imgui.PopStyleColor()
					}
				}

				// Texture Management
				imgui.Separator()
				if imgui.CollapsingHeaderV("Texture", imgui.TreeNodeFlagsDefaultOpen) {
                    if model.Material.TexturePath != "" {
						imgui.Text("Current:")
						imgui.SameLine()
						imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.0, Y: 0.678, Z: 0.847, W: 1.0})
						imgui.Text(filepath.Base(model.Material.TexturePath))
						imgui.PopStyleColor()
						
						imgui.Text(fmt.Sprintf("Texture ID: %d", model.Material.TextureID))
						
						if imgui.Button("Clear Texture") {
                            // Clear on main
                            model.Material.TextureID = 0
                            model.Material.TexturePath = ""
                            // Clear on groups
                            if len(model.MaterialGroups) > 0 {
                                for i := range model.MaterialGroups {
                                    if model.MaterialGroups[i].Material != nil {
                                        model.MaterialGroups[i].Material.TextureID = 0
                                        model.MaterialGroups[i].Material.TexturePath = ""
                                    }
                                }
                            }
							logToConsole(fmt.Sprintf("Texture cleared from '%s'", model.Name), "info")
						}
					} else {
						imgui.Text("No texture loaded")
					}
					if imgui.Button("Load Texture...") {
						// Open native file browser
						filename, err := dialog.File().
							SetStartDir("../examples/resources/textures").
							Filter("Images", "png", "jpg", "jpeg").
							Title("Load Texture").
							Load()
						if err == nil && filename != "" {
							loadTextureToSelected(filename)
						}
					}
				}
			}

			// Quick Actions
			imgui.Separator()
			imgui.Text("Quick Actions:")
			imgui.Spacing()
			
			if imgui.Button("Duplicate Model") {
				// Create a duplicate
				newModel, err := loader.LoadObjectWithPath(model.SourcePath, true)
				if err == nil {
					newModel.Name = model.Name + " (Copy)"
					newModel.SetPosition(model.Position.X() + 15.0, model.Position.Y(), model.Position.Z())
					newModel.SetScale(model.Scale.X(), model.Scale.Y(), model.Scale.Z())
					// Copy material
					if newModel.Material != nil && model.Material != nil {
						*newModel.Material = *model.Material
					}
					eng.AddModel(newModel)
					logToConsole(fmt.Sprintf("✓ Duplicated '%s'", model.Name), "info")
				}
			}
			imgui.SameLine()
			
			if imgui.Button("Focus Camera") {
				targetPos := model.Position
				eng.Camera.Position = targetPos.Add(mgl.Vec3{0, 50, 100})
				logToConsole(fmt.Sprintf("Camera focused on '%s'", model.Name), "info")
			}
			
			if imgui.Button("Reset Material") {
				if model.Material != nil {
					model.Material.DiffuseColor = [3]float32{1.0, 1.0, 1.0}
					model.Material.SpecularColor = [3]float32{1.0, 1.0, 1.0}
					model.Material.Shininess = 32.0
					model.Material.Metallic = 0.0
					model.Material.Roughness = 0.5
					model.Material.Exposure = 1.0
					model.Material.Alpha = 1.0
					model.IsDirty = true
					logToConsole("Material reset to defaults", "info")
				}
			}
			imgui.SameLine()
			
			if imgui.Button("Delete Model") {
				openglRenderer.RemoveModel(model)
				selectedModelIndex = -1
				selectedType = ""
				logToConsole(fmt.Sprintf("Deleted '%s'", model.Name), "info")
			}
		} else if selectedType == "light" && selectedLightIndex >= 0 && selectedLightIndex < len(openglRenderer.GetLights()) {
			lights := openglRenderer.GetLights()
			light := lights[selectedLightIndex]
			
			displayName := light.Name
			if displayName == "" {
				displayName = fmt.Sprintf("Light %d", selectedLightIndex)
			}
			imgui.Text("Selected: " + displayName)
			imgui.Separator()

			// Name
			imgui.InputText("Name", &light.Name)
			
			// Light type (read-only)
			imgui.Text(fmt.Sprintf("Type: %s", light.Mode))
			imgui.Separator()

			// Position or Direction
			if light.Mode == "point" {
				pos := [3]float32{light.Position.X(), light.Position.Y(), light.Position.Z()}
				if imgui.DragFloat3V("Position", &pos, 1.0, -10000, 10000, "%.2f", 0) {
					light.Position = mgl.Vec3{pos[0], pos[1], pos[2]}
				}
				
				// Range (derived from attenuation)
				// Approximate range where light intensity drops to ~1%
				range_ := float32(0.0)
				if light.QuadraticAtten > 0 {
					range_ = 1.0 / float32(math.Sqrt(float64(light.QuadraticAtten)))
				}
				if imgui.SliderFloatV("Range", &range_, 1.0, 1000.0, "%.1f", 0) {
					// Update attenuation based on range
					light.ConstantAtten = 1.0
					light.LinearAtten = 2.0 / range_
					light.QuadraticAtten = 1.0 / (range_ * range_)
				}
			} else if light.Mode == "directional" {
				dir := [3]float32{light.Direction.X(), light.Direction.Y(), light.Direction.Z()}
				if imgui.DragFloat3V("Direction", &dir, 0.1, -1.0, 1.0, "%.2f", 0) {
					light.Direction = mgl.Vec3{dir[0], dir[1], dir[2]}.Normalize()
				}
			}

			// Color
			imgui.Separator()
			color := [3]float32{light.Color.X(), light.Color.Y(), light.Color.Z()}
			if imgui.ColorEdit3V("Color", &color, 0) {
				light.Color = mgl.Vec3{color[0], color[1], color[2]}
			}

			// Intensity
			intensity := light.Intensity
			if imgui.SliderFloatV("Intensity", &intensity, 0.0, 10.0, "%.2f", 0) {
				light.Intensity = intensity
			}
			
			// Ambient Strength
			imgui.Separator()
			ambientStrength := light.AmbientStrength
			if imgui.SliderFloatV("Ambient Strength", &ambientStrength, 0.0, 1.0, "%.2f", 0) {
				light.AmbientStrength = ambientStrength
			}
			
			// Temperature (in Kelvin)
			temperature := light.Temperature
			if imgui.SliderFloatV("Temperature (K)", &temperature, 1000.0, 10000.0, "%.0f", 0) {
				light.Temperature = temperature
			}
			
			// Advanced: Manual Attenuation Control (for point lights)
			if light.Mode == "point" {
				imgui.Separator()
				if imgui.CollapsingHeaderV("Advanced Attenuation", 0) {
					constant := light.ConstantAtten
					if imgui.SliderFloatV("Constant", &constant, 0.0, 2.0, "%.3f", 0) {
						light.ConstantAtten = constant
					}
					linear := light.LinearAtten
					if imgui.SliderFloatV("Linear", &linear, 0.0, 1.0, "%.4f", 0) {
						light.LinearAtten = linear
					}
					quadratic := light.QuadraticAtten
					if imgui.SliderFloatV("Quadratic", &quadratic, 0.0, 1.0, "%.5f", 0) {
						light.QuadraticAtten = quadratic
					}
				}
			}

			// Delete light button
			imgui.Separator()
			imgui.Spacing()
			if imgui.Button("Delete Light") {
				openglRenderer.RemoveLight(light)
				selectedLightIndex = -1
				selectedType = ""
			}
		} else {
			imgui.Text("No object selected")
			imgui.Spacing()
			imgui.PushTextWrapPos()
			imgui.Text("Select an object from the Scene Hierarchy to edit its properties.")
			imgui.PopTextWrapPos()
		}
		imgui.End()
	}

	if showDemoWindow {
		imgui.ShowDemoWindow(&showDemoWindow)
	}
}

func renderAddModelDialog() {
	imgui.OpenPopup("Add Model")
	
	centerX := float32(eng.Width) / 2
	centerY := float32(eng.Height) / 2
	imgui.SetNextWindowPosV(imgui.Vec2{X: centerX - 200, Y: centerY - 250}, imgui.ConditionAppearing, imgui.Vec2{})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 400, Y: 500}, imgui.ConditionAppearing)
	
	if imgui.BeginPopupModalV("Add Model", nil, imgui.WindowFlagsNoResize) {
		imgui.Text("Select a model to add to the scene:")
		imgui.Separator()
		imgui.Spacing()
		
		// Model list with preview info
		for i, modelInfo := range availableModels {
			if imgui.SelectableV(modelInfo.Name, false, 0, imgui.Vec2{X: 0, Y: 30}) {
				// Load and add model
				addModelToScene(modelInfo.Path, modelInfo.Name)
				showAddModel = false
				imgui.CloseCurrentPopup()
			}
			
			// Show path as smaller text
			if i < len(availableModels) {
				imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.5, Y: 0.5, Z: 0.5, W: 1.0})
				imgui.Text(fmt.Sprintf("  Path: %s", modelInfo.Path))
				imgui.PopStyleColor()
				imgui.Spacing()
			}
		}
		
		imgui.Separator()
		imgui.Spacing()
		
		// Instancing options
		imgui.Text("Instancing Options:")
		if imgui.Checkbox("Enable Instancing", &instanceModelOnAdd) {
			saveConfig()
		}
		imgui.SameLine()
		if imgui.IsItemHovered() {
			imgui.SetTooltip("Render multiple copies efficiently using GPU instancing")
		}
		
		if instanceModelOnAdd {
			imgui.Text("Instance Count:")
			imgui.SameLine()
			imgui.PushItemWidth(100)
			instanceCount32 := int32(instanceCount)
			if imgui.InputInt("##instancecount", &instanceCount32) {
				instanceCount = int(instanceCount32)
				if instanceCount < 1 {
					instanceCount = 1
				} else if instanceCount > 10000 {
					instanceCount = 10000
				}
				saveConfig()
			}
			imgui.PopItemWidth()
		}
		
		imgui.Separator()
		
		// Buttons at bottom
		imgui.Spacing()
		if imgui.Button("Cancel") {
			showAddModel = false
			imgui.CloseCurrentPopup()
		}
		
		imgui.EndPopup()
	}
}

func renderAddLightDialog() {
	imgui.OpenPopup("Add Light")
	
	centerX := float32(eng.Width) / 2
	centerY := float32(eng.Height) / 2
	imgui.SetNextWindowPosV(imgui.Vec2{X: centerX - 200, Y: centerY - 150}, imgui.ConditionAppearing, imgui.Vec2{})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 400, Y: 300}, imgui.ConditionAppearing)
	
	if imgui.BeginPopupModalV("Add Light", nil, imgui.WindowFlagsNoResize) {
		imgui.Text("Select a light type to add:")
		imgui.Separator()
		imgui.Spacing()
		
		// Directional light option
		if imgui.SelectableV("[Dir] Directional Light", false, 0, imgui.Vec2{X: 0, Y: 40}) {
			// Create directional light
			light := renderer.CreateDirectionalLight(
				mgl.Vec3{-0.2, -1.0, -0.3}.Normalize(),
				mgl.Vec3{1.0, 1.0, 1.0},
				1.0,
			)
			light.Name = fmt.Sprintf("Directional Light %d", len(eng.GetRenderer().(*renderer.OpenGLRenderer).GetLights())+1)
			eng.GetRenderer().(*renderer.OpenGLRenderer).AddLight(light)
			logToConsole(fmt.Sprintf("Added %s", light.Name), "info")
			showAddLight = false
			imgui.CloseCurrentPopup()
		}
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.5, Y: 0.5, Z: 0.5, W: 1.0})
		imgui.Text("  Like sunlight, illuminates from a direction")
		imgui.PopStyleColor()
		imgui.Spacing()
		
		// Point light option
		if imgui.SelectableV("[Pnt] Point Light", false, 0, imgui.Vec2{X: 0, Y: 40}) {
			// Create point light at camera position
			light := renderer.CreatePointLight(
				eng.Camera.Position,
				mgl.Vec3{1.0, 1.0, 1.0},
				1.0,
				100.0,
			)
			light.Name = fmt.Sprintf("Point Light %d", len(eng.GetRenderer().(*renderer.OpenGLRenderer).GetLights())+1)
			eng.GetRenderer().(*renderer.OpenGLRenderer).AddLight(light)
			logToConsole(fmt.Sprintf("Added %s", light.Name), "info")
			showAddLight = false
			imgui.CloseCurrentPopup()
		}
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.5, Y: 0.5, Z: 0.5, W: 1.0})
		imgui.Text("  Like a bulb, radiates light in all directions")
		imgui.PopStyleColor()
		imgui.Spacing()
		
		imgui.Separator()
		
		// Buttons at bottom
		imgui.Spacing()
		if imgui.Button("Cancel") {
			showAddLight = false
			imgui.CloseCurrentPopup()
		}
		
		imgui.EndPopup()
	}
}

func getFileNameFromPath(path string) string {
	// Extract filename from path (cross-platform)
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			name := path[i+1:]
			// Remove extension
			for j := len(name) - 1; j >= 0; j-- {
				if name[j] == '.' {
					return name[:j]
				}
			}
			return name
		}
	}
	return path
}

func loadTextureToSelected(path string) {
	openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
	if !ok {
		fmt.Println("ERROR: Cannot access renderer")
		logToConsole("ERROR: Cannot access renderer", "error")
		return
	}

	models := openglRenderer.GetModels()
	if selectedType != "model" || selectedModelIndex < 0 || selectedModelIndex >= len(models) {
		fmt.Println("ERROR: No model selected")
		logToConsole("Please select a model before loading a texture", "warning")
		return
	}

	model := models[selectedModelIndex]
	logToConsole(fmt.Sprintf("Loading texture: %s", filepath.Base(path)), "info")
	
	textureID, err := openglRenderer.LoadTexture(path)
	if err != nil {
		fmt.Printf("ERROR: Failed to load texture: %v\n", err)
		logToConsole(fmt.Sprintf("Failed to load texture: %v", err), "error")
		return
	}

	if model.Material != nil {
		model.Material.TextureID = textureID
		model.Material.TexturePath = path  // Store path for display and scene saving
		model.IsDirty = true  // Force render update
	}
	fmt.Printf("✓ Texture loaded and applied to '%s'\n", model.Name)
	logToConsole(fmt.Sprintf("✓ Texture applied: %s to '%s'", filepath.Base(path), model.Name), "info")
}

func addModelToScene(path string, name string) {
	fmt.Printf("Loading model: %s from %s\n", name, path)
	logToConsole(fmt.Sprintf("Loading model: %s", name), "info")
	
	model, err := loader.LoadObjectWithPath(path, true)
	if err != nil {
		fmt.Printf("ERROR: Failed to load model: %v\n", err)
		logToConsole(fmt.Sprintf("Failed to load model: %v", err), "error")
		return
	}
	
	model.Name = name
	
	// Position new models slightly offset so they don't overlap
	models := eng.GetRenderer().(*renderer.OpenGLRenderer).GetModels()
	offset := float32(len(models)) * 5.0
	model.SetPosition(offset, 10, 0)
	model.SetScale(10, 10, 10)
	
	// Ensure proper material defaults
	if model.Material != nil {
		if model.Material.Exposure == 0 {
			model.Material.Exposure = 1.0
		}
		if model.Material.Alpha == 0 {
			model.Material.Alpha = 1.0
		}
	}
	
	// Apply instancing if enabled
	if instanceModelOnAdd && instanceCount > 1 {
		model.IsInstanced = true
		model.InstanceCount = instanceCount
		
		// Create instance matrices in a grid pattern
		model.InstanceModelMatrices = make([]mgl.Mat4, instanceCount)
		gridSize := int(math.Sqrt(float64(instanceCount))) + 1
		for i := 0; i < instanceCount; i++ {
			row := i / gridSize
			col := i % gridSize
			x := offset + float32(col)*20.0
			z := float32(row) * 20.0
			
			translation := mgl.Translate3D(x, 10, z)
			scale := mgl.Scale3D(10, 10, 10)
			model.InstanceModelMatrices[i] = translation.Mul4(scale)
		}
		model.InstanceMatricesUpdated = true
		
		fmt.Printf("✓ Model '%s' added with %d instances\n", name, instanceCount)
		logToConsole(fmt.Sprintf("✓ Model '%s' added with %d instances", name, instanceCount), "info")
	} else {
		fmt.Printf("✓ Model '%s' added to scene at position (%.1f, 10, 0)\n", name, offset)
		logToConsole(fmt.Sprintf("✓ Model '%s' added to scene", name), "info")
	}
	
	eng.AddModel(model)
}

func applyDarkTheme() {
	style := imgui.CurrentStyle()
	
	// Go Cyan color (#00ADD8)
	goCyan := imgui.Vec4{X: 0.0, Y: 0.678, Z: 0.847, W: 1.0}
	goCyanHover := imgui.Vec4{X: 0.0, Y: 0.678, Z: 0.847, W: 0.6}
	goCyanActive := imgui.Vec4{X: 0.0, Y: 0.678, Z: 0.847, W: 0.8}
	goCyanDim := imgui.Vec4{X: 0.0, Y: 0.678, Z: 0.847, W: 0.4}

	// Base colors
	style.SetColor(imgui.StyleColorWindowBg, imgui.Vec4{X: 0.1, Y: 0.1, Z: 0.1, W: 0.95})
	style.SetColor(imgui.StyleColorTitleBg, imgui.Vec4{X: 0.08, Y: 0.08, Z: 0.08, W: 1.0})
	style.SetColor(imgui.StyleColorTitleBgActive, goCyan) // Active window title - Go cyan
	style.SetColor(imgui.StyleColorMenuBarBg, imgui.Vec4{X: 0.14, Y: 0.14, Z: 0.14, W: 1.0})
	
	// Borders and separators - Go cyan
	style.SetColor(imgui.StyleColorBorder, goCyan)
	style.SetColor(imgui.StyleColorSeparator, goCyan)
	style.SetColor(imgui.StyleColorSeparatorHovered, goCyanActive)
	style.SetColor(imgui.StyleColorSeparatorActive, goCyan)
	
	// Headers (hierarchy selection) - Go cyan
	style.SetColor(imgui.StyleColorHeader, goCyanDim)
	style.SetColor(imgui.StyleColorHeaderHovered, goCyanHover)
	style.SetColor(imgui.StyleColorHeaderActive, goCyan)
	
	// Buttons - Go cyan accents
	style.SetColor(imgui.StyleColorButton, imgui.Vec4{X: 0.2, Y: 0.2, Z: 0.2, W: 1.0})
	style.SetColor(imgui.StyleColorButtonHovered, goCyanHover)
	style.SetColor(imgui.StyleColorButtonActive, goCyanActive)
	
	// Frame backgrounds
	style.SetColor(imgui.StyleColorFrameBg, imgui.Vec4{X: 0.2, Y: 0.2, Z: 0.2, W: 0.54})
	style.SetColor(imgui.StyleColorFrameBgHovered, imgui.Vec4{X: 0.25, Y: 0.25, Z: 0.25, W: 0.78})
	style.SetColor(imgui.StyleColorFrameBgActive, imgui.Vec4{X: 0.3, Y: 0.3, Z: 0.3, W: 0.67})
	
	// Sliders and grab handles - Go cyan
	style.SetColor(imgui.StyleColorSliderGrab, goCyan)
	style.SetColor(imgui.StyleColorSliderGrabActive, goCyanActive)
	
	// Tabs - Go cyan
	style.SetColor(imgui.StyleColorTab, imgui.Vec4{X: 0.15, Y: 0.15, Z: 0.15, W: 1.0})
	style.SetColor(imgui.StyleColorTabHovered, goCyanActive)
	style.SetColor(imgui.StyleColorTabActive, goCyan)
	style.SetColor(imgui.StyleColorTabUnfocused, imgui.Vec4{X: 0.12, Y: 0.12, Z: 0.12, W: 1.0})
	style.SetColor(imgui.StyleColorTabUnfocusedActive, goCyanDim)
	
	// Checkboxes and radio buttons
	style.SetColor(imgui.StyleColorCheckMark, goCyan)
	
	// Text selection
	style.SetColor(imgui.StyleColorTextSelectedBg, goCyanDim)

	// Increase border thickness for visibility
	style.SetWindowBorderSize(1.5)
	style.SetFrameBorderSize(1.0)
	style.SetWindowRounding(4.0)
	style.SetFrameRounding(2.0)
	style.SetGrabRounding(2.0)
}

func setupEditorScene() {
	fmt.Println("Setting up editor scene...")

	// Check if camera is ready (should be by now, but be safe)
	if eng.Camera == nil {
		fmt.Println("Warning: Camera not ready yet, skipping scene setup")
		sceneSetup = false // Allow retry next frame
		return
	}

	eng.Camera.Position = mgl.Vec3{0, 50, 150}
	eng.Camera.Speed = 100
	eng.Camera.InvertMouse = false

	// Create default light
	defaultLight := renderer.CreateDirectionalLight(
		mgl.Vec3{-0.3, -1, -0.5},
		mgl.Vec3{1.0, 0.95, 0.85},
		1.5,
	)
	defaultLight.Name = "Sun"
	defaultLight.AmbientStrength = 0.3
	defaultLight.Type = renderer.STATIC_LIGHT
	eng.Light = defaultLight
	
	// Add light to renderer's lights array (so editor can manage it)
	openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
	if ok {
		openglRenderer.AddLight(defaultLight)
	}

	// Note: Grid floor removed - it was interfering with the scene
	// TODO: Implement proper debug grid lines if needed
	
	fmt.Println("✓ Editor scene ready!")
	
	// Load editor configuration
	loadConfig()
	
	// Add initial console message
	logToConsole("Editor initialized - Type 'help' for available commands", "info")
}

func renderFileExplorer() {
	imgui.Begin("File Explorer")
	
	// Current directory display
	imgui.Text("Current: " + currentDirectory)
	imgui.Separator()
	
	// Up directory button
	if imgui.Button(".. (Up)") {
		parentDir := filepath.Dir(currentDirectory)
		if parentDir != currentDirectory {
			currentDirectory = parentDir
		}
	}
	
	imgui.Separator()
	
	// Read directory contents
	entries, err := os.ReadDir(currentDirectory)
	if err != nil {
		imgui.Text("Error reading directory: " + err.Error())
		imgui.End()
		return
	}
	
	// Display directories first
	for _, entry := range entries {
		if entry.IsDir() {
			if imgui.SelectableV("📁 "+entry.Name(), false, 0, imgui.Vec2{}) {
				currentDirectory = filepath.Join(currentDirectory, entry.Name())
			}
		}
	}
	
	// Then display files
	for _, entry := range entries {
		if !entry.IsDir() {
			icon := "📄"
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext == ".obj" {
				icon = "🗿"
			} else if ext == ".png" || ext == ".jpg" || ext == ".jpeg" {
				icon = "🖼️"
			}
			
			selected := selectedFilePath == filepath.Join(currentDirectory, entry.Name())
			if imgui.SelectableV(icon+" "+entry.Name(), selected, 0, imgui.Vec2{}) {
				selectedFilePath = filepath.Join(currentDirectory, entry.Name())
				
				// Double-click to load
				if ext == ".obj" {
					logToConsole("Loading model: "+selectedFilePath, "info")
					name := getFileNameFromPath(selectedFilePath)
					addModelToScene(selectedFilePath, name)
				} else if ext == ".png" || ext == ".jpg" || ext == ".jpeg" {
					logToConsole("Loading texture: "+selectedFilePath, "info")
					loadTextureToSelected(selectedFilePath)
				}
			}
		}
	}
	
	imgui.End()
}

func renderConsole() {
	imgui.BeginV("Console", nil, imgui.WindowFlagsMenuBar)
	
	// Menu bar for console options
	if imgui.BeginMenuBar() {
		if imgui.MenuItem("Clear") {
			consoleLines = []ConsoleEntry{}
		}
		imgui.Checkbox("Auto-scroll", &consoleAutoScroll)
		imgui.EndMenuBar()
	}
	
	// Tab bar
	if imgui.BeginTabBar("ConsoleTabs") {
		// Console Tab (with command input)
		if imgui.BeginTabItem("Console") {
			// Reserve space for input bar at bottom
			footerHeight := imgui.FrameHeightWithSpacing()
			imgui.BeginChildV("ConsoleScrollRegion", imgui.Vec2{X: 0, Y: -footerHeight}, true, 0)
			
			// Display console lines
			for _, entry := range consoleLines {
				var color imgui.Vec4
				switch entry.Type {
				case "error":
					color = imgui.Vec4{X: 1.0, Y: 0.3, Z: 0.3, W: 1.0}
				case "warning":
					color = imgui.Vec4{X: 1.0, Y: 0.8, Z: 0.2, W: 1.0}
				case "command":
					color = imgui.Vec4{X: 0.5, Y: 0.8, Z: 1.0, W: 1.0}
				default:
					color = imgui.Vec4{X: 0.9, Y: 0.9, Z: 0.9, W: 1.0}
				}
				
				imgui.PushStyleColor(imgui.StyleColorText, color)
				imgui.Text(entry.Message)
				imgui.PopStyleColor()
			}
			
			// Auto-scroll to bottom
			if consoleAutoScroll && imgui.ScrollY() >= imgui.ScrollMaxY() {
				imgui.SetScrollHereY(1.0)
			}
			
			imgui.EndChild()
			
			// Command input
			imgui.Separator()
			imgui.PushItemWidth(-1)
			if imgui.InputTextV("##ConsoleInput", &consoleInput, imgui.InputTextFlagsEnterReturnsTrue, nil) {
				if consoleInput != "" {
					executeConsoleCommand(consoleInput)
					consoleInput = ""
				}
				imgui.SetKeyboardFocusHere() // Keep focus on input
			}
			imgui.PopItemWidth()
			
			imgui.EndTabItem()
		}
		
		// Logs Tab (engine logs, read-only)
		if imgui.BeginTabItem("Logs") {
			imgui.BeginChildV("LogsScrollRegion", imgui.Vec2{X: 0, Y: 0}, true, 0)
			
			imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.7, Y: 0.7, Z: 0.7, W: 1.0})
			imgui.Text("Engine logs appear here...")
			imgui.Text("(JSON logs from logger are visible in terminal)")
			imgui.PopStyleColor()
			
			// TODO: Integrate with engine logger to display logs here
			
			imgui.EndChild()
			imgui.EndTabItem()
		}
		
		// Performance Tab (stats + controls)
		if imgui.BeginTabItem("Performance") {
			openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
			if ok {
				models := openglRenderer.GetModels()
				imgui.Text(fmt.Sprintf("FPS: %.0f", fps))
				imgui.Text(fmt.Sprintf("Models: %d", len(models)))
				imgui.Text(fmt.Sprintf("Draw Calls: %d", openglRenderer.GetDrawCalls()))
				imgui.Text(fmt.Sprintf("Instances: %d", openglRenderer.GetTotalInstanceCount()))
				
				imgui.Separator()
				imgui.Text("Rendering Options:")
				
				debug := renderer.Debug
				if imgui.Checkbox("Wireframe", &debug) {
					renderer.Debug = debug
				}
				
				frustum := renderer.FrustumCullingEnabled
				if imgui.Checkbox("Frustum Culling", &frustum) {
					renderer.FrustumCullingEnabled = frustum
				}
				
				faceCull := renderer.FaceCullingEnabled
				if imgui.Checkbox("Face Culling", &faceCull) {
					renderer.FaceCullingEnabled = faceCull
				}
			}
			imgui.EndTabItem()
		}
		
		imgui.EndTabBar()
	}
	
	imgui.End()
}

func updateFPS() {
	frameCount++
	now := time.Now()
	
	// Update FPS every second
	if now.Sub(fpsUpdateTime) >= time.Second {
		fps = float64(frameCount) / now.Sub(fpsUpdateTime).Seconds()
		frameCount = 0
		fpsUpdateTime = now
	}
}

func logToConsole(message string, msgType string) {
	timestamp := time.Now().Format("15:04:05")
	consoleLines = append(consoleLines, ConsoleEntry{
		Message: fmt.Sprintf("[%s] %s", timestamp, message),
		Type:    msgType,
	})
	
	// Limit console history
	if len(consoleLines) > maxConsoleLines {
		consoleLines = consoleLines[len(consoleLines)-maxConsoleLines:]
	}
}

func executeConsoleCommand(cmd string) {
	logToConsole("> "+cmd, "command")
	
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}
	
	command := strings.ToLower(parts[0])
	
	switch command {
	case "help":
		logToConsole("Available commands:", "info")
		logToConsole("  clear - Clear console", "info")
		logToConsole("  models - List all models in scene", "info")
		logToConsole("  inspect <name> - Show detailed material info", "info")
		logToConsole("  wireframe [on/off] - Toggle wireframe mode", "info")
		logToConsole("  culling [on/off] - Toggle frustum culling", "info")
		logToConsole("  delete <name> - Delete model by name", "info")
		logToConsole("  grid [on/off] - Toggle reference grid visibility", "info")
		logToConsole("  fix-materials - Reset all materials to defaults", "info")
		
	case "clear":
		consoleLines = []ConsoleEntry{}
		
	case "grid":
		if len(parts) > 1 {
			openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
			if ok {
				models := openglRenderer.GetModels()
				for _, model := range models {
					if model.Name == "Grid Floor" {
						if parts[1] == "off" {
							model.SetScale(0, 0, 0) // Hide grid
							logToConsole("Reference grid hidden", "info")
						} else if parts[1] == "on" {
							model.SetScale(500, 0.5, 500) // Show grid
							logToConsole("Reference grid visible", "info")
						}
						break
					}
				}
			}
		} else {
			logToConsole("Usage: grid [on/off]", "warning")
		}
		
	case "models":
		openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
		if ok {
			models := openglRenderer.GetModels()
			logToConsole(fmt.Sprintf("Total models: %d", len(models)), "info")
			for i, model := range models {
				logToConsole(fmt.Sprintf("  %d: %s (pos: %.1f, %.1f, %.1f)", 
					i, model.Name, model.Position.X(), model.Position.Y(), model.Position.Z()), "info")
			}
		}
		
	case "wireframe":
		if len(parts) > 1 {
			if parts[1] == "on" {
				renderer.Debug = true
				logToConsole("Wireframe enabled", "info")
			} else if parts[1] == "off" {
				renderer.Debug = false
				logToConsole("Wireframe disabled", "info")
			}
		} else {
			logToConsole("Usage: wireframe [on/off]", "warning")
		}
		
	case "culling":
		if len(parts) > 1 {
			if parts[1] == "on" {
				renderer.FrustumCullingEnabled = true
				logToConsole("Frustum culling enabled", "info")
			} else if parts[1] == "off" {
				renderer.FrustumCullingEnabled = false
				logToConsole("Frustum culling disabled", "info")
			}
		} else {
			logToConsole("Usage: culling [on/off]", "warning")
		}
		
	case "delete":
		if len(parts) > 1 {
			modelName := strings.Join(parts[1:], " ")
			openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
			if ok {
				models := openglRenderer.GetModels()
				found := false
				for _, model := range models {
					if model.Name == modelName {
						openglRenderer.RemoveModel(model)
						logToConsole(fmt.Sprintf("Deleted model: %s", modelName), "info")
						found = true
						break
					}
				}
				if !found {
					logToConsole(fmt.Sprintf("Model not found: %s", modelName), "error")
				}
			}
		} else {
			logToConsole("Usage: delete <model_name>", "warning")
		}
		
	case "inspect":
		if len(parts) > 1 {
			modelName := strings.Join(parts[1:], " ")
			openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
			if !ok {
				logToConsole("ERROR: Cannot access renderer", "error")
				return
			}
			models := openglRenderer.GetModels()
			found := false
			for _, model := range models {
				if model.Name == modelName {
					found = true
					logToConsole(fmt.Sprintf("=== Model: %s ===", model.Name), "info")
					logToConsole(fmt.Sprintf("Path: %s", model.SourcePath), "info")
					logToConsole(fmt.Sprintf("Position: (%.2f, %.2f, %.2f)", model.Position.X(), model.Position.Y(), model.Position.Z()), "info")
					logToConsole(fmt.Sprintf("Scale: (%.2f, %.2f, %.2f)", model.Scale.X(), model.Scale.Y(), model.Scale.Z()), "info")
					if model.Material != nil {
						logToConsole("=== Material ===", "info")
						logToConsole(fmt.Sprintf("Diffuse: (%.2f, %.2f, %.2f)", model.Material.DiffuseColor[0], model.Material.DiffuseColor[1], model.Material.DiffuseColor[2]), "info")
						logToConsole(fmt.Sprintf("Specular: (%.2f, %.2f, %.2f)", model.Material.SpecularColor[0], model.Material.SpecularColor[1], model.Material.SpecularColor[2]), "info")
						logToConsole(fmt.Sprintf("Shininess: %.2f", model.Material.Shininess), "info")
						logToConsole(fmt.Sprintf("Metallic: %.2f, Roughness: %.2f", model.Material.Metallic, model.Material.Roughness), "info")
						logToConsole(fmt.Sprintf("Exposure: %.2f (CRITICAL)", model.Material.Exposure), "info")
						logToConsole(fmt.Sprintf("Alpha: %.2f", model.Material.Alpha), "info")
						if model.Material.TexturePath != "" {
							logToConsole(fmt.Sprintf("Texture: %s (ID: %d)", filepath.Base(model.Material.TexturePath), model.Material.TextureID), "info")
						} else {
							logToConsole("Texture: None", "info")
						}
					}
					break
				}
			}
			if !found {
				logToConsole(fmt.Sprintf("Model '%s' not found", modelName), "error")
			}
		} else {
			logToConsole("Usage: inspect <model_name>", "warning")
		}
	
	case "fix-materials":
		openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
		if !ok {
			logToConsole("ERROR: Cannot access renderer", "error")
			return
		}
		models := openglRenderer.GetModels()
		fixed := 0
		for _, model := range models {
			if model.Material != nil {
				needsFix := false
				if model.Material.Exposure == 0 {
					model.Material.Exposure = 1.0
					needsFix = true
				}
				if model.Material.Alpha == 0 {
					model.Material.Alpha = 1.0
					needsFix = true
				}
				if needsFix {
					model.IsDirty = true
					fixed++
				}
			}
		}
		logToConsole(fmt.Sprintf("Fixed %d models with incorrect material values", fixed), "info")
		
	default:
		logToConsole(fmt.Sprintf("Unknown command: %s (type 'help' for commands)", command), "error")
	}
}

// ============================================
// Camera Focus
// ============================================

func focusCameraOnModel(model *renderer.Model) {
	// Calculate distance to view the entire model
	// Use bounding sphere radius if available, otherwise estimate from scale
	distance := model.BoundingSphereRadius
	if distance <= 0 {
		// Estimate from scale (use largest dimension)
		maxScale := model.Scale.X()
		if model.Scale.Y() > maxScale {
			maxScale = model.Scale.Y()
		}
		if model.Scale.Z() > maxScale {
			maxScale = model.Scale.Z()
		}
		distance = maxScale * 3.0 // View from 3x the size
	} else {
		distance *= 2.5 // View from 2.5x the bounding radius
	}
	
	// Position camera to look at the model
	// Place camera in front and slightly above the model
	targetPos := model.Position
	cameraPos := mgl.Vec3{
		targetPos.X(),
		targetPos.Y() + distance * 0.3, // Slightly above
		targetPos.Z() + distance,        // In front
	}
	
	// Set camera position
	eng.Camera.Position = cameraPos
	
	// Calculate direction to look at target
	direction := targetPos.Sub(cameraPos).Normalize()
	
	// Calculate yaw and pitch from direction vector
	eng.Camera.Yaw = float32(math.Atan2(float64(direction.X()), float64(direction.Z()))) * 180.0 / 3.14159
	eng.Camera.Pitch = float32(math.Asin(float64(direction.Y()))) * 180.0 / 3.14159
	
	// Camera vectors will update automatically on next frame
	
	logToConsole(fmt.Sprintf("Focused camera on '%s'", model.Name), "info")
}

// ============================================
// Scene Management
// ============================================

type SceneData struct {
	Models []SceneModel `json:"models"`
	Lights []SceneLight `json:"lights"`
}

type SceneModel struct {
	Name            string     `json:"name"`
	Path            string     `json:"path"`
	Position        [3]float32 `json:"position"`
	Scale           [3]float32 `json:"scale"`
	Rotation        [3]float32 `json:"rotation"`
	
	// Complete Material Properties
	DiffuseColor    [3]float32 `json:"diffuse_color"`
	SpecularColor   [3]float32 `json:"specular_color"`
	Shininess       float32    `json:"shininess"`
	Metallic        float32    `json:"metallic"`
	Roughness       float32    `json:"roughness"`
	Exposure        float32    `json:"exposure"`
	Alpha           float32    `json:"alpha"`
	TexturePath     string     `json:"texture_path"`
}

type SceneLight struct {
	Name            string     `json:"name"`
	Mode            string     `json:"mode"` // "directional" or "point"
	Position        [3]float32 `json:"position"`
	Direction       [3]float32 `json:"direction"`
	Color           [3]float32 `json:"color"`
	Intensity       float32    `json:"intensity"`
	AmbientStrength float32    `json:"ambient_strength"`
	Temperature     float32    `json:"temperature"`
	ConstantAtten   float32    `json:"constant_atten"`
	LinearAtten     float32    `json:"linear_atten"`
	QuadraticAtten  float32    `json:"quadratic_atten"`
}

func newScene() {
	if sceneModified {
		// TODO: Add confirmation dialog
		logToConsole("Creating new scene (unsaved changes will be lost)", "warning")
	}
	
	openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
	if !ok {
		logToConsole("ERROR: Cannot access renderer", "error")
		return
	}
	
	// Clear all models
	models := openglRenderer.GetModels()
	for i := len(models) - 1; i >= 0; i-- {
		openglRenderer.RemoveModel(models[i])
	}
	
	// Clear all lights except default
	lights := openglRenderer.GetLights()
	for i := len(lights) - 1; i >= 0; i-- {
		if lights[i].Name != "Sun" {
			openglRenderer.RemoveLight(lights[i])
		}
	}
	
	// Ensure we have at least one default light
	lights = openglRenderer.GetLights()
	if len(lights) == 0 {
		defaultLight := renderer.CreateDirectionalLight(
			mgl.Vec3{-0.3, -1, -0.5},
			mgl.Vec3{1.0, 0.95, 0.85},
			1.5,
		)
		defaultLight.Name = "Sun"
		defaultLight.AmbientStrength = 0.3
		defaultLight.Type = renderer.STATIC_LIGHT
		openglRenderer.AddLight(defaultLight)
		eng.Light = defaultLight
	} else {
		// Ensure eng.Light points to the first light (should be Sun)
		eng.Light = lights[0]
	}
	
	currentScenePath = ""
	sceneModified = false
	selectedModelIndex = -1
	selectedLightIndex = -1
	selectedType = ""
	
	logToConsole("New scene created", "info")
}

func saveScene() {
	// Open save dialog
	filename, err := dialog.File().
		Filter("Scene Files", "json").
		Title("Save Scene").
		Save()
	
	if err != nil || filename == "" {
		return
	}
	
	// Ensure .json extension
	if !strings.HasSuffix(filename, ".json") {
		filename += ".json"
	}
	
	openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
	if !ok {
		logToConsole("ERROR: Cannot access renderer", "error")
		return
	}
	
	// Collect scene data
	sceneData := SceneData{
		Models: []SceneModel{},
		Lights: []SceneLight{},
	}
	
	// Save models with complete material properties
	models := openglRenderer.GetModels()
	for _, model := range models {
		sceneModel := SceneModel{
			Name:     model.Name,
			Path:     model.SourcePath,
			Position: [3]float32{model.Position.X(), model.Position.Y(), model.Position.Z()},
			Scale:    [3]float32{model.Scale.X(), model.Scale.Y(), model.Scale.Z()},
			Rotation: [3]float32{0, 0, 0}, // TODO: Rotation not yet implemented
		}
		if model.Material != nil {
			sceneModel.DiffuseColor = model.Material.DiffuseColor
			sceneModel.SpecularColor = model.Material.SpecularColor
			sceneModel.Shininess = model.Material.Shininess
			sceneModel.Metallic = model.Material.Metallic
			sceneModel.Roughness = model.Material.Roughness
			sceneModel.Exposure = model.Material.Exposure
			sceneModel.Alpha = model.Material.Alpha
			sceneModel.TexturePath = model.Material.TexturePath
		}
		sceneData.Models = append(sceneData.Models, sceneModel)
	}
	
	// Save lights with complete properties
	lights := openglRenderer.GetLights()
	for _, light := range lights {
		sceneLight := SceneLight{
			Name:            light.Name,
			Mode:            light.Mode,
			Position:        [3]float32{light.Position.X(), light.Position.Y(), light.Position.Z()},
			Direction:       [3]float32{light.Direction.X(), light.Direction.Y(), light.Direction.Z()},
			Color:           [3]float32{light.Color.X(), light.Color.Y(), light.Color.Z()},
			Intensity:       light.Intensity,
			AmbientStrength: light.AmbientStrength,
			Temperature:     light.Temperature,
			ConstantAtten:   light.ConstantAtten,
			LinearAtten:     light.LinearAtten,
			QuadraticAtten:  light.QuadraticAtten,
		}
		sceneData.Lights = append(sceneData.Lights, sceneLight)
	}
	
	// Write to file
	jsonData, err := json.MarshalIndent(sceneData, "", "  ")
	if err != nil {
		logToConsole(fmt.Sprintf("Failed to serialize scene: %v", err), "error")
		return
	}
	
	err = ioutil.WriteFile(filename, jsonData, 0644)
	if err != nil {
		logToConsole(fmt.Sprintf("Failed to save scene: %v", err), "error")
		return
	}
	
	currentScenePath = filename
	sceneModified = false
	logToConsole(fmt.Sprintf("Scene saved: %s", filepath.Base(filename)), "info")
}

func loadScene() {
	// Open load dialog
	filename, err := dialog.File().
		Filter("Scene Files", "json").
		Title("Load Scene").
		Load()
	
	if err != nil || filename == "" {
		return
	}
	
	// Read file
	jsonData, err := ioutil.ReadFile(filename)
	if err != nil {
		logToConsole(fmt.Sprintf("Failed to load scene: %v", err), "error")
		return
	}
	
	// Parse JSON
	var sceneData SceneData
	err = json.Unmarshal(jsonData, &sceneData)
	if err != nil {
		logToConsole(fmt.Sprintf("Failed to parse scene: %v", err), "error")
		return
	}
	
	// Clear current scene first (this resets selection)
	// But save the selection reset for AFTER we load, so UI updates correctly
	newScene()
	
	openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
	if !ok {
		logToConsole("ERROR: Cannot access renderer", "error")
		return
	}
	
	// When loading a scene, clear ALL lights (including defaults)
	// so we only have the lights from the scene file
	lights := openglRenderer.GetLights()
	for i := len(lights) - 1; i >= 0; i-- {
		openglRenderer.RemoveLight(lights[i])
	}
	// Clear engine's main light reference to avoid dangling pointer
	eng.Light = nil
	
	// Clear selection before loading models to ensure indices stay valid
	selectedModelIndex = -1
	selectedLightIndex = -1
	selectedType = ""
	
	// Load models
	for _, sceneModel := range sceneData.Models {
		if sceneModel.Path == "" {
			logToConsole(fmt.Sprintf("Skipping model '%s' (no path stored)", sceneModel.Name), "warning")
			continue
		}
		
		logToConsole(fmt.Sprintf("Loading model: %s from %s", sceneModel.Name, filepath.Base(sceneModel.Path)), "info")
		
		model, err := loader.LoadObjectWithPath(sceneModel.Path, true)
		if err != nil {
			logToConsole(fmt.Sprintf("Failed to load model: %v", err), "error")
			continue
		}
		
		model.Name = sceneModel.Name
		model.SetPosition(sceneModel.Position[0], sceneModel.Position[1], sceneModel.Position[2])
		model.SetScale(sceneModel.Scale[0], sceneModel.Scale[1], sceneModel.Scale[2])
		
		// CRITICAL: Ensure each model gets unique material instances
		// This prevents materials from being shared between different models
		var originalMainMaterial *renderer.Material = nil
		if model.Material != nil {
			// Save pointer to original material before replacing
			originalMainMaterial = model.Material
			// Create a unique copy of the material to avoid sharing with other models
			// Ensure defaults are set if values are invalid
			exposure := originalMainMaterial.Exposure
			if exposure <= 0 {
				exposure = 1.0
			}
			alpha := originalMainMaterial.Alpha
			if alpha <= 0 {
				alpha = 1.0
			}
			model.Material = &renderer.Material{
				Name:          originalMainMaterial.Name,
				DiffuseColor:  originalMainMaterial.DiffuseColor,
				SpecularColor: originalMainMaterial.SpecularColor,
				Shininess:     originalMainMaterial.Shininess,
				Metallic:      originalMainMaterial.Metallic,
				Roughness:     originalMainMaterial.Roughness,
				Exposure:      exposure,
				Alpha:         alpha,
				TextureID:     0,
				TexturePath:   originalMainMaterial.TexturePath,
			}
		}
		
		// Also ensure material groups have unique instances
		for i := range model.MaterialGroups {
			if model.MaterialGroups[i].Material != nil {
				originalMat := model.MaterialGroups[i].Material
				// If this material group points to the original model.Material, use the new unique instance
				if originalMat == originalMainMaterial {
					model.MaterialGroups[i].Material = model.Material
					continue
				}
				// Create unique copy for this material group
				// Ensure defaults are set if values are invalid
				exposure := originalMat.Exposure
				if exposure <= 0 {
					exposure = 1.0
				}
				alpha := originalMat.Alpha
				if alpha <= 0 {
					alpha = 1.0
				}
				model.MaterialGroups[i].Material = &renderer.Material{
					Name:          originalMat.Name,
					DiffuseColor:  originalMat.DiffuseColor,
					SpecularColor: originalMat.SpecularColor,
					Shininess:     originalMat.Shininess,
					Metallic:      originalMat.Metallic,
					Roughness:     originalMat.Roughness,
					Exposure:      exposure,
					Alpha:         alpha,
					TextureID:     0,
					TexturePath:   originalMat.TexturePath,
				}
			}
		}
		
		// FIRST: Restore material properties BEFORE AddModel
		// This ensures materials have correct values from the start, preventing black rendering
		// Track unique materials to avoid restoring the same material multiple times
		restoredMaterials := make(map[*renderer.Material]bool)
		
		if model.Material != nil {
			model.Material.DiffuseColor = sceneModel.DiffuseColor
			model.Material.SpecularColor = sceneModel.SpecularColor
			model.Material.Shininess = sceneModel.Shininess
			model.Material.Metallic = sceneModel.Metallic
			model.Material.Roughness = sceneModel.Roughness
			// Ensure Exposure is never 0 (which would make model completely black)
			if sceneModel.Exposure > 0 {
				model.Material.Exposure = sceneModel.Exposure
			} else {
				model.Material.Exposure = 1.0 // Default if saved as 0
			}
			// Ensure Alpha is never 0 (which would make model invisible)
			if sceneModel.Alpha > 0 {
				model.Material.Alpha = sceneModel.Alpha
			} else {
				model.Material.Alpha = 1.0 // Default if saved as 0
			}
			restoredMaterials[model.Material] = true
		}
		
		// Restore properties for ALL unique materials in material groups
		for i := range model.MaterialGroups {
			if model.MaterialGroups[i].Material != nil {
				groupMat := model.MaterialGroups[i].Material
				
				// Only restore if we haven't already restored this material
				if !restoredMaterials[groupMat] {
					groupMat.DiffuseColor = sceneModel.DiffuseColor
					groupMat.SpecularColor = sceneModel.SpecularColor
					groupMat.Shininess = sceneModel.Shininess
					groupMat.Metallic = sceneModel.Metallic
					groupMat.Roughness = sceneModel.Roughness
					// Ensure Exposure is never 0 (which would make model completely black)
					if sceneModel.Exposure > 0 {
						groupMat.Exposure = sceneModel.Exposure
					} else {
						groupMat.Exposure = 1.0 // Default if saved as 0
					}
					// Ensure Alpha is never 0 (which would make model invisible)
					if sceneModel.Alpha > 0 {
						groupMat.Alpha = sceneModel.Alpha
					} else {
						groupMat.Alpha = 1.0 // Default if saved as 0
					}
					restoredMaterials[groupMat] = true
				}
			}
		}
		
		// SECOND: Set texture paths BEFORE AddModel so textures load correctly
		if sceneModel.TexturePath != "" {
			if model.Material != nil {
				model.Material.TexturePath = sceneModel.TexturePath
				model.Material.TextureID = 0 // Clear so loadModelTextures will load it
			}
			// Also set for material groups so they load correctly
			for i := range model.MaterialGroups {
				if model.MaterialGroups[i].Material != nil {
					model.MaterialGroups[i].Material.TexturePath = sceneModel.TexturePath
					model.MaterialGroups[i].Material.TextureID = 0
				}
			}
		}
		
		// THIRD: Add model to initialize OpenGL resources and load textures
		// Materials are now properly configured with correct exposure/alpha before this call
		eng.AddModel(model)
		
		// Mark model as dirty to ensure uniforms are updated on next render
		model.IsDirty = true
	}
	
	// Load lights with complete properties
	isFirstLight := true
	for _, sceneLight := range sceneData.Lights {
		var light *renderer.Light
		if sceneLight.Mode == "directional" {
			light = renderer.CreateDirectionalLight(
				mgl.Vec3{sceneLight.Direction[0], sceneLight.Direction[1], sceneLight.Direction[2]},
				mgl.Vec3{sceneLight.Color[0], sceneLight.Color[1], sceneLight.Color[2]},
				sceneLight.Intensity,
			)
		} else if sceneLight.Mode == "point" {
			light = renderer.CreatePointLight(
				mgl.Vec3{sceneLight.Position[0], sceneLight.Position[1], sceneLight.Position[2]},
				mgl.Vec3{sceneLight.Color[0], sceneLight.Color[1], sceneLight.Color[2]},
				sceneLight.Intensity,
				100.0, // Default range
			)
		}
		if light != nil {
			light.Name = sceneLight.Name
			light.AmbientStrength = sceneLight.AmbientStrength
			light.Temperature = sceneLight.Temperature
			light.ConstantAtten = sceneLight.ConstantAtten
			light.LinearAtten = sceneLight.LinearAtten
			light.QuadraticAtten = sceneLight.QuadraticAtten
			openglRenderer.AddLight(light)
			// Always set the first light as the engine's main light (for backward compatibility)
			if isFirstLight {
				eng.Light = light
				isFirstLight = false
			}
		}
	}
	
	currentScenePath = filename
	sceneModified = false
	logToConsole(fmt.Sprintf("Scene loaded: %s (%d models, %d lights)", filepath.Base(filename), len(sceneData.Models), len(sceneData.Lights)), "info")
}
