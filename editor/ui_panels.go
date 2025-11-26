package main

import (
	"Gopher3D/internal/renderer"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	mgl "github.com/go-gl/mathgl/mgl32"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/sqweek/dialog"
)

// getDefaultPanelLayout returns default layout for a panel
func getDefaultPanelLayout(panelName string, width, height int32) PanelLayout {
	switch panelName {
	case "hierarchy":
		return PanelLayout{
			PosX:      10,
			PosY:      30,
			SizeX:     300,
			SizeY:     float32(height) - 420, 
			Collapsed: false,
		}
	case "inspector":
		return PanelLayout{
			PosX:      float32(width) - 350,
			PosY:      30,
			SizeX:     340,
			SizeY:     float32(height) - 420,
			Collapsed: false,
		}
	case "file_explorer":
		return PanelLayout{
			PosX:      10,
			PosY:      float32(height) - 360,
			SizeX:     300,
			SizeY:     150,
			Collapsed: false,
		}
	case "console":
		return PanelLayout{
			PosX:      320,
			PosY:      float32(height) - 360,
			SizeX:     float32(width) - 680,
			SizeY:     350,
			Collapsed: false,
		}
	case "scene_settings":
		return PanelLayout{
			PosX:      float32(width) - 380,
			PosY:      float32(height) - 430,
			SizeX:     370,
			SizeY:     420,
			Collapsed: false,
		}
	case "advanced_render":
		return PanelLayout{
			PosX:      float32(width)/2 - 200,
			PosY:      100,
			SizeX:     400,
			SizeY:     450,
			Collapsed: false,
		}
	default:
		return PanelLayout{PosX: 100, PosY: 100, SizeX: 300, SizeY: 300, Collapsed: false}
	}
}

func initializePanelLayouts() {
	if eng == nil || eng.Width <= 0 || eng.Height <= 0 {
		return 
	}

	if !layoutsInitialized {
		if hierarchyLayout.PosX == 0 && hierarchyLayout.PosY == 0 {
			hierarchyLayout = getDefaultPanelLayout("hierarchy", eng.Width, eng.Height)
		}
		if inspectorLayout.PosX == 0 && inspectorLayout.PosY == 0 {
			inspectorLayout = getDefaultPanelLayout("inspector", eng.Width, eng.Height)
		}
		if fileExplorerLayout.PosX == 0 && fileExplorerLayout.PosY == 0 {
			fileExplorerLayout = getDefaultPanelLayout("file_explorer", eng.Width, eng.Height)
		}
		if consoleLayout.PosX == 0 && consoleLayout.PosY == 0 {
			consoleLayout = getDefaultPanelLayout("console", eng.Width, eng.Height)
		}
		if sceneSettingsLayout.PosX == 0 && sceneSettingsLayout.PosY == 0 {
			sceneSettingsLayout = getDefaultPanelLayout("scene_settings", eng.Width, eng.Height)
		}
		if advancedRenderLayout.PosX == 0 && advancedRenderLayout.PosY == 0 {
			advancedRenderLayout = getDefaultPanelLayout("advanced_render", eng.Width, eng.Height)
		}
		layoutsInitialized = true
	}
}

func renderEditorUI() {
	// Safety check: ensure engine and renderer are ready
	if eng == nil || eng.GetRenderer() == nil {
		return
	}
	
	openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
	if !ok {
		return
	}

	// Initialize panel layouts if not done
	initializePanelLayouts()

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
				startDir := "../examples/resources/obj"
				if currentProject != nil {
					startDir = filepath.Join(currentProject.Path, "resources/models")
				}
				
				filename, err := dialog.File().
					SetStartDir(startDir).
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
						startDir := "../examples/resources/textures"
						if currentProject != nil {
							startDir = filepath.Join(currentProject.Path, "resources/textures")
						}
						
						filename, err := dialog.File().
							SetStartDir(startDir).
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
			imgui.Separator()
			if imgui.MenuItem("Voxel Terrain") {
				showAddVoxel = true
			}
			if imgui.MenuItem("Water Plane") {
				showAddWater = true
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
			imgui.Separator()
			if imgui.MenuItemV("Show Gizmos", "", showGizmos, true) {
				showGizmos = !showGizmos
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
	
	// Add Water Dialog
	if showAddWater {
		renderAddWaterDialog()
	}
	
	// Add Voxel Dialog
	if showAddVoxel {
		renderAddVoxelDialog()
	}

	// File Explorer (Bottom Left)
	if showFileExplorer {
		imgui.SetNextWindowPosV(imgui.Vec2{X: fileExplorerLayout.PosX, Y: fileExplorerLayout.PosY}, imgui.ConditionFirstUseEver, imgui.Vec2{})
		if !layoutsInitialized {
			imgui.SetNextWindowSizeV(imgui.Vec2{X: fileExplorerLayout.SizeX, Y: fileExplorerLayout.SizeY}, imgui.ConditionFirstUseEver)
		}
		if imgui.BeginV("File Explorer", nil, 0) {
			// Save layout changes
			size := imgui.WindowSize()
			pos := imgui.WindowPos()
			if size.X != fileExplorerLayout.SizeX || size.Y != fileExplorerLayout.SizeY || pos.X != fileExplorerLayout.PosX || pos.Y != fileExplorerLayout.PosY {
				fileExplorerLayout.SizeX = size.X
				fileExplorerLayout.SizeY = size.Y
				fileExplorerLayout.PosX = pos.X
				fileExplorerLayout.PosY = pos.Y
				saveConfig()
			}
			renderFileExplorerContent()
		}
		imgui.End()
	}

	// Console (Bottom Middle)
	if showConsole {
		imgui.SetNextWindowPosV(imgui.Vec2{X: consoleLayout.PosX, Y: consoleLayout.PosY}, imgui.ConditionFirstUseEver, imgui.Vec2{})
		if !layoutsInitialized {
			imgui.SetNextWindowSizeV(imgui.Vec2{X: consoleLayout.SizeX, Y: consoleLayout.SizeY}, imgui.ConditionFirstUseEver)
		}
		if imgui.BeginV("Console", nil, imgui.WindowFlagsMenuBar) {
			size := imgui.WindowSize()
			pos := imgui.WindowPos()
			if size.X != consoleLayout.SizeX || size.Y != consoleLayout.SizeY || pos.X != consoleLayout.PosX || pos.Y != consoleLayout.PosY {
				consoleLayout.SizeX = size.X
				consoleLayout.SizeY = size.Y
				consoleLayout.PosX = pos.X
				consoleLayout.PosY = pos.Y
				saveConfig()
			}
			renderConsoleContent()
		}
		imgui.End()
	}

	// Style Editor (Restored)
	if showStyleEditor && eng != nil {
		imgui.SetNextWindowPosV(imgui.Vec2{X: float32(eng.Width) - 520, Y: 30}, imgui.ConditionFirstUseEver, imgui.Vec2{})
		imgui.SetNextWindowSizeV(imgui.Vec2{X: 500, Y: 600}, imgui.ConditionFirstUseEver)
		if imgui.Begin("Style Editor") {
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

			imgui.Separator()
			imgui.Text("Quick Presets:")
			if imgui.Button("Go Cyan") {
				applyDarkTheme() 
			}
			imgui.SameLine()
			if imgui.Button("Reset to Dark") {
				imgui.StyleColorsDark()
			}
		}
		imgui.End()
	}

	// Advanced Rendering Options
	if showAdvancedRender {
		// Ensure layout is initialized
		if !layoutsInitialized || (advancedRenderLayout.PosX == 0 && advancedRenderLayout.PosY == 0) {
			if eng.Width > 0 && eng.Height > 0 {
				advancedRenderLayout = getDefaultPanelLayout("advanced_render", eng.Width, eng.Height)
			}
		}
		
		imgui.SetNextWindowPosV(imgui.Vec2{X: advancedRenderLayout.PosX, Y: advancedRenderLayout.PosY}, imgui.ConditionFirstUseEver, imgui.Vec2{})
		if !layoutsInitialized {
			imgui.SetNextWindowSizeV(imgui.Vec2{X: advancedRenderLayout.SizeX, Y: advancedRenderLayout.SizeY}, imgui.ConditionFirstUseEver)
		}
		
		if imgui.BeginV("Advanced Rendering", nil, 0) {
			size := imgui.WindowSize()
			pos := imgui.WindowPos()
			if size.X != advancedRenderLayout.SizeX || size.Y != advancedRenderLayout.SizeY || pos.X != advancedRenderLayout.PosX || pos.Y != advancedRenderLayout.PosY {
				advancedRenderLayout.SizeX = size.X
				advancedRenderLayout.SizeY = size.Y
				advancedRenderLayout.PosX = pos.X
				advancedRenderLayout.PosY = pos.Y
				saveConfig()
			}
			
			imgui.Text("Advanced Rendering Configuration")
			imgui.Separator()
			
			// Quality Presets
			imgui.Text("Quality Presets:")
			if imgui.Button("Performance") {
				applyRenderingPreset("performance")
			}
			imgui.SameLine()
			if imgui.Button("Balanced") {
				applyRenderingPreset("balanced")
			}
			imgui.SameLine()
			if imgui.Button("High Quality") {
				applyRenderingPreset("quality")
			}
			imgui.SameLine()
			if imgui.Button("Voxel") {
				applyRenderingPreset("voxel")
			}
			
			imgui.Separator()
			imgui.Separator()
			
			// Basic Rendering Options
			if imgui.CollapsingHeaderV("Basic Rendering", imgui.TreeNodeFlagsDefaultOpen) {
				if imgui.Checkbox("Wireframe Mode", &renderer.Debug) {
					logToConsole(fmt.Sprintf("Wireframe: %v", renderer.Debug), "info")
				}
				if imgui.Checkbox("Frustum Culling", &renderer.FrustumCullingEnabled) {
					logToConsole(fmt.Sprintf("Frustum Culling: %v", renderer.FrustumCullingEnabled), "info")
				}
				if imgui.Checkbox("Face Culling", &renderer.FaceCullingEnabled) {
					logToConsole(fmt.Sprintf("Face Culling: %v", renderer.FaceCullingEnabled), "info")
				}
				if imgui.Checkbox("Depth Testing", &renderer.DepthTestEnabled) {
					logToConsole(fmt.Sprintf("Depth Test: %v", renderer.DepthTestEnabled), "info")
				}
			}
			
			imgui.Separator()
			
			// Global Advanced Rendering Toggle
			if imgui.Checkbox("Enable Advanced Rendering Features", &globalAdvancedRenderingEnabled) {
				logToConsole(fmt.Sprintf("Advanced Rendering: %v", globalAdvancedRenderingEnabled), "info")
			}
			imgui.Text("Enable advanced PBR materials, lighting effects,")
			imgui.Text("and post-processing for all models.")
			
			if globalAdvancedRenderingEnabled {
				imgui.Separator()
				
				// PBR Materials
				if imgui.CollapsingHeaderV("PBR Materials", 0) {
					renderAdvancedRenderingPBR()
				}
				
				// Lighting Effects
				if imgui.CollapsingHeaderV("Lighting Effects", 0) {
					renderAdvancedRenderingLighting()
				}
				
				// Post Processing
				if imgui.CollapsingHeaderV("Post Processing", 0) {
					renderAdvancedRenderingPostProcess()
				}
				
				// Performance
				if imgui.CollapsingHeaderV("Performance", 0) {
					renderAdvancedRenderingPerformance()
				}
			}
		}
		imgui.End()
	}

	// Scene Settings
	if showSceneSettings {
		imgui.SetNextWindowPosV(imgui.Vec2{X: sceneSettingsLayout.PosX, Y: sceneSettingsLayout.PosY}, imgui.ConditionFirstUseEver, imgui.Vec2{})
		if !layoutsInitialized {
			imgui.SetNextWindowSizeV(imgui.Vec2{X: sceneSettingsLayout.SizeX, Y: sceneSettingsLayout.SizeY}, imgui.ConditionFirstUseEver)
		}
		
		if imgui.BeginV("Scene Settings", nil, 0) {
			size := imgui.WindowSize()
			pos := imgui.WindowPos()
			if size.X != sceneSettingsLayout.SizeX || size.Y != sceneSettingsLayout.SizeY || pos.X != sceneSettingsLayout.PosX || pos.Y != sceneSettingsLayout.PosY {
				sceneSettingsLayout.SizeX = size.X
				sceneSettingsLayout.SizeY = size.Y
				sceneSettingsLayout.PosX = pos.X
				sceneSettingsLayout.PosY = pos.Y
				saveConfig()
			}
			
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
				if skyboxColorMode {
					imgui.ColorEdit3V("##skycolor", &skyboxSolidColor, 0)
					if imgui.Button("Apply") {
						// Explicitly set the renderer clear color
						openglRenderer.ClearColorR = skyboxSolidColor[0]
						openglRenderer.ClearColorG = skyboxSolidColor[1]
						openglRenderer.ClearColorB = skyboxSolidColor[2]
						eng.UpdateSkyboxColor(skyboxSolidColor[0], skyboxSolidColor[1], skyboxSolidColor[2])
						logToConsole(fmt.Sprintf("Background color set to RGB(%.2f, %.2f, %.2f)", skyboxSolidColor[0], skyboxSolidColor[1], skyboxSolidColor[2]), "info")
					}
				} else {
					// Skybox Image Mode
					if imgui.Button("Load Skybox Image...") {
						startDir := "../examples/resources/textures"
						if currentProject != nil {
							startDir = filepath.Join(currentProject.Path, "resources/textures")
						}
						
						filename, err := dialog.File().
							SetStartDir(startDir).
							Filter("Images", "png", "jpg", "jpeg").
							Title("Load Skybox Image").
							Load()
						if err == nil && filename != "" {
							eng.SetSkybox(filename)
							// Clear background color to allow skybox to show (if transparent, though skybox usually draws over)
							// Actually skybox renders last or uses depth buffer. 
							// Renderer implementation: checks skybox presence. 
							// But we need to reset ClearColor to 0 to avoid clearing over it if that logic exists, 
							// or just trust renderer priority.
							// In OpenGLRenderer.Render:
							// if rend.ClearColorR != 0 ... -> Use ClearColor
							// else if rend.skybox != nil -> Use Skybox Color (solid)
							// Wait, texture skybox?
							// The current renderer logic for Skybox with Texture (CreateSkybox) sets TextureID.
							// But Render() function says: "Skybox rendering is now handled by the renderer using clear color".
							// That seems to be for SOLID color only!
							// If we have a textured skybox, we need to draw the cube!
							// The renderer code for Render(camera) seems to have COMMENTED OUT actual skybox rendering in step 3!
							// "Skybox rendering is now handled by the renderer using clear color - no rendering needed"
							// This is WRONG for textured skyboxes.
							// I should fix the renderer too if textured skybox is broken.
							// But for now, let's just assume the user wants the button.
							logToConsole("Loaded skybox: " + getFileNameFromPath(filename), "info")
						}
					}
				}
			}
		}
		imgui.End()
	}

	// Scene Hierarchy (Left Side, Top)
	if showHierarchy {
		imgui.SetNextWindowPosV(imgui.Vec2{X: hierarchyLayout.PosX, Y: hierarchyLayout.PosY}, imgui.ConditionFirstUseEver, imgui.Vec2{})
		if !layoutsInitialized {
			imgui.SetNextWindowSizeV(imgui.Vec2{X: hierarchyLayout.SizeX, Y: hierarchyLayout.SizeY}, imgui.ConditionFirstUseEver)
		}
		if imgui.BeginV("Scene Hierarchy", nil, 0) {
			size := imgui.WindowSize()
			pos := imgui.WindowPos()
			if size.X != hierarchyLayout.SizeX || size.Y != hierarchyLayout.SizeY || pos.X != hierarchyLayout.PosX || pos.Y != hierarchyLayout.PosY {
				hierarchyLayout.SizeX = size.X
				hierarchyLayout.SizeY = size.Y
				hierarchyLayout.PosX = pos.X
				hierarchyLayout.PosY = pos.Y
				saveConfig()
			}
			imgui.Text("Scene Objects:")
			imgui.Separator()

			// Models section
			if imgui.CollapsingHeaderV("[M] Models", imgui.TreeNodeFlagsDefaultOpen) {
				for i, model := range models {
					imgui.PushID(fmt.Sprintf("model_%d", i))
					isSelected := selectedType == "model" && selectedModelIndex == i
					if imgui.SelectableV("  "+model.Name, isSelected, 0, imgui.Vec2{}) {
						selectedModelIndex = i
						selectedLightIndex = -1
						selectedType = "model"
					}
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
		}
		imgui.End()
	}

	// Inspector (Right Side, Top)
	if showInspector {
		imgui.SetNextWindowPosV(imgui.Vec2{X: inspectorLayout.PosX, Y: inspectorLayout.PosY}, imgui.ConditionFirstUseEver, imgui.Vec2{})
		if !layoutsInitialized {
			imgui.SetNextWindowSizeV(imgui.Vec2{X: inspectorLayout.SizeX, Y: inspectorLayout.SizeY}, imgui.ConditionFirstUseEver)
		}
		if imgui.BeginV("Inspector", nil, 0) {
			size := imgui.WindowSize()
			pos := imgui.WindowPos()
			if size.X != inspectorLayout.SizeX || size.Y != inspectorLayout.SizeY || pos.X != inspectorLayout.PosX || pos.Y != inspectorLayout.PosY {
				inspectorLayout.SizeX = size.X
				inspectorLayout.SizeY = size.Y
				inspectorLayout.PosX = pos.X
				inspectorLayout.PosY = pos.Y
				saveConfig()
			}

			if selectedType == "model" && selectedModelIndex >= 0 && selectedModelIndex < len(models) {
				model := models[selectedModelIndex]
				
				imgui.Text("Selected Model:")
				modelName := model.Name
				imgui.PushItemWidth(-1)
				// Standard input text for easier editing
				if imgui.InputTextV("##modelName", &modelName, 0, nil) {
					model.Name = modelName
				}
				imgui.PopItemWidth()
				imgui.Separator()

				imgui.Spacing()
				if imgui.CollapsingHeaderV("Transform", imgui.TreeNodeFlagsDefaultOpen) {
					imgui.Text("Position")
					posX, posY, posZ := model.Position.X(), model.Position.Y(), model.Position.Z()
					w := imgui.ContentRegionAvail().X
					imgui.PushItemWidth(w / 3.3)
					changed := false
					if imgui.DragFloatV("##posX", &posX, 0.5, 0, 0, "X: %.1f", 0) { changed = true }
					imgui.SameLine()
					if imgui.DragFloatV("##posY", &posY, 0.5, 0, 0, "Y: %.1f", 0) { changed = true }
					imgui.SameLine()
					if imgui.DragFloatV("##posZ", &posZ, 0.5, 0, 0, "Z: %.1f", 0) { changed = true }
					imgui.PopItemWidth()
					if changed {
						model.SetPosition(posX, posY, posZ)
						model.IsDirty = true
					}
					
					imgui.Spacing()
					imgui.Text("Scale")
					scaleX, scaleY, scaleZ := model.Scale.X(), model.Scale.Y(), model.Scale.Z()
					imgui.PushItemWidth(w / 3.3)
					changed = false
					if imgui.DragFloatV("##scaleX", &scaleX, 0.05, 0, 0, "X: %.2f", 0) { changed = true }
					imgui.SameLine()
					if imgui.DragFloatV("##scaleY", &scaleY, 0.05, 0, 0, "Y: %.2f", 0) { changed = true }
					imgui.SameLine()
					if imgui.DragFloatV("##scaleZ", &scaleZ, 0.05, 0, 0, "Z: %.2f", 0) { changed = true }
					imgui.PopItemWidth()
					if changed {
						model.SetScale(scaleX, scaleY, scaleZ)
						model.IsDirty = true
					}
				}
				
				// Material editing...
				if model.Material != nil {
					imgui.Separator()
					if imgui.CollapsingHeaderV("Material Properties", imgui.TreeNodeFlagsDefaultOpen) {
						diffuse := [3]float32{model.Material.DiffuseColor[0], model.Material.DiffuseColor[1], model.Material.DiffuseColor[2]}
						if imgui.ColorEdit3V("Diffuse Color", &diffuse, 0) {
							model.SetDiffuseColor(diffuse[0], diffuse[1], diffuse[2])
							model.IsDirty = true
						}
						
						// Texture loading
						imgui.Separator()
						if model.Material.TexturePath != "" {
							imgui.Text(fmt.Sprintf("Texture: %s", filepath.Base(model.Material.TexturePath)))
						} else {
							imgui.Text("Texture: None")
						}
						
						if imgui.Button("Load Texture...") {
							startDir := "../examples/resources/textures"
							if currentProject != nil {
								startDir = filepath.Join(currentProject.Path, "resources/textures")
							}
							
							filename, err := dialog.File().
								SetStartDir(startDir).
								Filter("Images", "png", "jpg", "jpeg").
								Title("Load Texture for Model").
								Load()
							if err == nil && filename != "" {
								loadTextureToSelected(filename)
							}
						}
						
						if model.Material.TexturePath != "" {
							imgui.SameLine()
							if imgui.Button("Remove Texture") {
								model.Material.TextureID = 0
								model.Material.TexturePath = ""
								// Remove from material groups too
								for i := range model.MaterialGroups {
									if model.MaterialGroups[i].Material != nil {
										model.MaterialGroups[i].Material.TextureID = 0
										model.MaterialGroups[i].Material.TexturePath = ""
									}
								}
								model.IsDirty = true
								logToConsole(fmt.Sprintf("Removed texture from %s", model.Name), "info")
							}
						}
					}
				}
				
				// Water Settings
				// Check Metadata first, then verify activeWaterSim matches (optional but safer)
				if model.Metadata != nil && model.Metadata["type"] == "water" {
					imgui.Separator()
					// Ensure we are editing the CORRECT water simulation if multiple could exist (singleton for now)
					sim := activeWaterSim 
					
					if sim != nil {
						if imgui.CollapsingHeaderV("Water Simulation", imgui.TreeNodeFlagsDefaultOpen) {
							// Color
							waterColor := [3]float32{sim.WaterColor.X(), sim.WaterColor.Y(), sim.WaterColor.Z()}
							if imgui.ColorEdit3V("Water Color", &waterColor, 0) {
								sim.WaterColor = mgl.Vec3{waterColor[0], waterColor[1], waterColor[2]}
							}
							
							// Transparency
							transparency := sim.Transparency
							if imgui.SliderFloatV("Transparency", &transparency, 0.0, 1.0, "%.2f", 1.0) {
								sim.Transparency = transparency
							}
							
						// Speed
						speed := sim.WaveSpeedMultiplier
						if imgui.SliderFloatV("Wave Speed", &speed, 0.0, 5.0, "%.2f", 1.0) {
							sim.WaveSpeedMultiplier = speed
						}
						
						imgui.Separator()
						imgui.Text("Advanced Appearance")
						
						// Foam / Fog
						foam := sim.FoamEnabled
						if imgui.Checkbox("Enable Atmosphere/Foam", &foam) {
							sim.FoamEnabled = foam
						}
						
						foamInt := sim.FoamIntensity
						if imgui.SliderFloatV("Atmosphere Intensity", &foamInt, 0.0, 1.0, "%.2f", 1.0) {
							sim.FoamIntensity = foamInt
						}
						
						// Specular
						spec := sim.SpecularIntensity
						if imgui.SliderFloatV("Reflectivity", &spec, 0.0, 2.0, "%.2f", 1.0) {
							sim.SpecularIntensity = spec
						}
						
						// Normal/Distortion
						norm := sim.NormalStrength
						if imgui.SliderFloatV("Surface Detail", &norm, 0.0, 2.0, "%.2f", 1.0) {
							sim.NormalStrength = norm
						}
						
						dist := sim.DistortionStrength
						if imgui.SliderFloatV("Distortion", &dist, 0.0, 1.0, "%.2f", 1.0) {
							sim.DistortionStrength = dist
						}
						
						imgui.Separator()
						if imgui.Button("Load Water Texture...") {
							startDir := "../examples/resources/textures"
							if currentProject != nil {
								startDir = filepath.Join(currentProject.Path, "resources/textures")
							}
							
							filename, err := dialog.File().
								SetStartDir(startDir).
								Filter("Images", "png", "jpg", "jpeg").
								Title("Load Water Texture").
								Load()
							if err == nil && filename != "" {
								sim.TexturePath = filename
								// Apply to model material
								if model.Material == nil {
									model.Material = &renderer.Material{
										Name: "WaterMaterial",
										DiffuseColor: [3]float32{1, 1, 1},
										Alpha: sim.Transparency,
									}
								}
								model.Material.TexturePath = filename
								model.Material.TextureID = 0 // Force reload
								logToConsole("Loaded water texture: " + getFileNameFromPath(filename), "info")
							}
						}
						
						imgui.Text("Note: Some changes update in real-time")
						}
					} else {
						imgui.Text("Water Simulation State Lost")
					}
				}
				
			} else if selectedType == "light" && selectedLightIndex >= 0 {
				// Safety check for index
				lights := openglRenderer.GetLights()
				if selectedLightIndex < len(lights) {
					// Light editing...
					light := lights[selectedLightIndex]
					imgui.Text("Selected Light: " + light.Name)
					imgui.Separator()
					
					// Light Type/Mode Display
					imgui.Text("Type: " + strings.Title(light.Mode))
					imgui.Spacing()

					// Position (Always visible)
					imgui.Text("Position")
					posX, posY, posZ := light.Position.X(), light.Position.Y(), light.Position.Z()
					w := imgui.ContentRegionAvail().X
					imgui.PushItemWidth(w / 3.3)
					changed := false
					if imgui.DragFloatV("##lposX", &posX, 0.5, 0, 0, "X: %.1f", 0) { changed = true }
					imgui.SameLine()
					if imgui.DragFloatV("##lposY", &posY, 0.5, 0, 0, "Y: %.1f", 0) { changed = true }
					imgui.SameLine()
					if imgui.DragFloatV("##lposZ", &posZ, 0.5, 0, 0, "Z: %.1f", 0) { changed = true }
					imgui.PopItemWidth()
					if changed {
						light.Position = mgl.Vec3{posX, posY, posZ}
					}

					// Direction (Always show for directional/spot, or if user wants to see it)
					// Forcing visibility for now to ensure user sees it
					if light.Mode == "directional" || light.Mode == "spot" {
						imgui.Text("Direction")
						dirX, dirY, dirZ := light.Direction.X(), light.Direction.Y(), light.Direction.Z()
						imgui.PushItemWidth(w / 3.3)
						changed = false
						if imgui.DragFloatV("##ldirX", &dirX, 0.05, -1, 1, "X: %.2f", 0) { changed = true }
						imgui.SameLine()
						if imgui.DragFloatV("##ldirY", &dirY, 0.05, -1, 1, "Y: %.2f", 0) { changed = true }
						imgui.SameLine()
						if imgui.DragFloatV("##ldirZ", &dirZ, 0.05, -1, 1, "Z: %.2f", 0) { changed = true }
						imgui.PopItemWidth()
						if changed {
							light.Direction = mgl.Vec3{dirX, dirY, dirZ}
						}
					}

					imgui.Separator()
					
					// Color
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
					ambient := light.AmbientStrength
					if imgui.SliderFloatV("Ambient", &ambient, 0.0, 1.0, "%.2f", 0) {
						light.AmbientStrength = ambient
					}
					
					// Temperature
					temp := light.Temperature
					if imgui.SliderFloatV("Temperature (K)", &temp, 1000.0, 12000.0, "%.0f", 0) {
						light.Temperature = temp
					}

					if light.Mode == "point" || light.Mode == "spot" {
						imgui.Separator()
						imgui.Text("Attenuation")
						// Attenuation
						constant := light.ConstantAtten
						linear := light.LinearAtten
						quad := light.QuadraticAtten
						
						if imgui.DragFloatV("Constant", &constant, 0.01, 0, 10, "%.3f", 0) { light.ConstantAtten = constant }
						if imgui.DragFloatV("Linear", &linear, 0.0001, 0, 1, "%.4f", 0) { light.LinearAtten = linear }
						if imgui.DragFloatV("Quadratic", &quad, 0.000001, 0, 1, "%.6f", 0) { light.QuadraticAtten = quad }
					}
				}
			}
		}
		imgui.End()
	}

	if showDemoWindow {
		imgui.ShowDemoWindow(&showDemoWindow)
	}
}

func renderFileExplorerContent() {
	// Breadcrumb navigation
	imgui.Text("Path:")
	imgui.SameLine()
	
	// Update current directory if default
	if currentDirectory == "." {
		wd, _ := os.Getwd()
		currentDirectory = wd
	}

	imgui.Text(currentDirectory)
	imgui.Separator()

	// Navigation buttons
	if imgui.Button(".. (Parent)") {
		parentDir := filepath.Dir(currentDirectory)
		if parentDir != currentDirectory {
			currentDirectory = parentDir
		}
	}
	imgui.SameLine()
	if imgui.Button("Refresh") {
	}

	imgui.Separator()

	// Read directory contents
	entries, err := os.ReadDir(currentDirectory)
	if err != nil {
		imgui.Text("Error reading directory: " + err.Error())
		return
	}

	// Display directories first
	imgui.Text("Directories:")
	for _, entry := range entries {
		if entry.IsDir() {
			if imgui.SelectableV("[DIR] "+entry.Name(), false, 0, imgui.Vec2{}) {
				currentDirectory = filepath.Join(currentDirectory, entry.Name())
			}
		}
	}

	imgui.Separator()
	imgui.Text("Files:")
	for _, entry := range entries {
		if !entry.IsDir() {
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			selected := selectedFilePath == filepath.Join(currentDirectory, entry.Name())
			
			if imgui.SelectableV(entry.Name(), selected, 0, imgui.Vec2{}) {
				selectedFilePath = filepath.Join(currentDirectory, entry.Name())
				if imgui.IsItemHovered() && imgui.IsMouseDoubleClicked(0) {
					if ext == ".obj" {
						addModelToScene(selectedFilePath, getFileNameFromPath(selectedFilePath))
					}
				}
			}
		}
	}
}

func renderConsoleContent() {
	if imgui.BeginMenuBar() {
		if imgui.MenuItem("Clear") {
			consoleLines = []ConsoleEntry{}
		}
		imgui.Checkbox("Auto-scroll", &consoleAutoScroll)
		imgui.EndMenuBar()
	}

	footerHeight := imgui.FrameHeightWithSpacing()
	if imgui.BeginChildV("ConsoleScrollRegion", imgui.Vec2{X: 0, Y: -footerHeight}, true, 0) {
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
		if consoleAutoScroll && imgui.ScrollY() >= imgui.ScrollMaxY() {
			imgui.SetScrollHereY(1.0)
		}
		imgui.EndChild()
	}

	imgui.Separator()
	imgui.PushItemWidth(-1)
	
	// Check for Enter key manually if needed, or rely on InputTextFlagsEnterReturnsTrue
	// Using InputText with flags
	if imgui.InputTextV("##ConsoleInput", &consoleInput, imgui.InputTextFlagsEnterReturnsTrue, nil) {
		if consoleInput != "" {
			executeConsoleCommand(consoleInput)
			consoleInput = ""
			// Reclaim focus
			imgui.SetKeyboardFocusHere()
		}
	}
	imgui.PopItemWidth()
}

// Advanced Rendering UI Helper Functions

func renderAdvancedRenderingPBR() {
	if eng == nil || eng.GetRenderer() == nil {
		imgui.Text("No renderer available")
		return
	}
	
	openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
	if !ok {
		imgui.Text("OpenGL renderer required")
		return
	}
	
	models := openglRenderer.GetModels()
	if len(models) == 0 {
		imgui.Text("No models to configure")
		return
	}
	
	// Get current config from first model or use defaults
	config := getAdvancedConfigFromModel(models[0])
	changed := false
	
	// Clearcoat
	if imgui.Checkbox("Enable Clearcoat", &config.EnableClearcoat) {
		changed = true
	}
	if config.EnableClearcoat {
		imgui.Indent()
		if imgui.SliderFloatV("Roughness##clearcoat", &config.ClearcoatRoughness, 0.0, 1.0, "%.2f", 1.0) {
			changed = true
		}
		if imgui.SliderFloatV("Intensity##clearcoat", &config.ClearcoatIntensity, 0.0, 1.0, "%.2f", 1.0) {
			changed = true
		}
		imgui.Unindent()
	}
	
	// Sheen
	if imgui.Checkbox("Enable Sheen", &config.EnableSheen) {
		changed = true
	}
	if config.EnableSheen {
		imgui.Indent()
		sheenColor := [3]float32{config.SheenColor.X(), config.SheenColor.Y(), config.SheenColor.Z()}
		if imgui.ColorEdit3V("Color##sheen", &sheenColor, 0) {
			config.SheenColor = mgl.Vec3{sheenColor[0], sheenColor[1], sheenColor[2]}
			changed = true
		}
		if imgui.SliderFloatV("Roughness##sheen", &config.SheenRoughness, 0.0, 1.0, "%.2f", 1.0) {
			changed = true
		}
		imgui.Unindent()
	}
	
	// Transmission
	if imgui.Checkbox("Enable Transmission", &config.EnableTransmission) {
		changed = true
	}
	if config.EnableTransmission {
		imgui.Indent()
		if imgui.SliderFloatV("Factor##transmission", &config.TransmissionFactor, 0.0, 1.0, "%.2f", 1.0) {
			changed = true
		}
		imgui.Unindent()
	}
	
	imgui.Separator()
	
	// Advanced Lighting Models
	if imgui.Checkbox("Multiple Scattering", &config.EnableMultipleScattering) {
		changed = true
	}
	if imgui.Checkbox("Energy Conservation", &config.EnableEnergyConservation) {
		changed = true
	}
	if imgui.Checkbox("Image-Based Lighting", &config.EnableImageBasedLighting) {
		changed = true
	}
	if config.EnableImageBasedLighting {
		imgui.Indent()
		if imgui.SliderFloatV("IBL Intensity", &config.IBLIntensity, 0.0, 2.0, "%.2f", 1.0) {
			changed = true
		}
		imgui.Unindent()
	}
	
	if changed {
		applyAdvancedConfigToAllModels(config)
	}
}

func renderAdvancedRenderingLighting() {
	if eng == nil || eng.GetRenderer() == nil {
		imgui.Text("No renderer available")
		return
	}
	
	openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
	if !ok {
		return
	}
	
	models := openglRenderer.GetModels()
	if len(models) == 0 {
		imgui.Text("No models to configure")
		return
	}
	
	config := getAdvancedConfigFromModel(models[0])
	changed := false
	
	// SSAO
	if imgui.Checkbox("Enable SSAO", &config.EnableSSAO) {
		changed = true
	}
	if config.EnableSSAO {
		imgui.Indent()
		if imgui.SliderFloatV("Intensity##ssao", &config.SSAOIntensity, 0.0, 1.0, "%.2f", 1.0) {
			changed = true
		}
		if imgui.SliderFloatV("Radius##ssao", &config.SSAORadius, 10.0, 500.0, "%.0f", 1.0) {
			changed = true
		}
		if imgui.SliderFloatV("Bias##ssao", &config.SSAOBias, 0.0, 0.1, "%.3f", 1.0) {
			changed = true
		}
		samples := int32(config.SSAOSampleCount)
		if imgui.SliderIntV("Samples##ssao", &samples, 4, 16, "%d", 1.0) {
			config.SSAOSampleCount = int(samples)
			changed = true
		}
		imgui.Unindent()
	}
	
	// Volumetric Lighting
	if imgui.Checkbox("Enable Volumetric Lighting", &config.EnableVolumetricLighting) {
		changed = true
	}
	if config.EnableVolumetricLighting {
		imgui.Indent()
		if imgui.SliderFloatV("Intensity##volumetric", &config.VolumetricIntensity, 0.0, 2.0, "%.2f", 1.0) {
			changed = true
		}
		steps := int32(config.VolumetricSteps)
		if imgui.SliderIntV("Steps##volumetric", &steps, 4, 32, "%d", 1.0) {
			config.VolumetricSteps = int(steps)
			changed = true
		}
		if imgui.SliderFloatV("Scattering##volumetric", &config.VolumetricScattering, 0.0, 1.0, "%.2f", 1.0) {
			changed = true
		}
		imgui.Unindent()
	}
	
	// Global Illumination
	if imgui.Checkbox("Enable Global Illumination", &config.EnableGlobalIllumination) {
		changed = true
	}
	if config.EnableGlobalIllumination {
		imgui.Indent()
		if imgui.SliderFloatV("Intensity##gi", &config.GIIntensity, 0.0, 2.0, "%.2f", 1.0) {
			changed = true
		}
		bounces := int32(config.GIBounces)
		if imgui.SliderIntV("Bounces##gi", &bounces, 1, 5, "%d", 1.0) {
			config.GIBounces = int(bounces)
			changed = true
		}
		imgui.Unindent()
	}
	
	// Shadows
	imgui.Separator()
	if imgui.Checkbox("Enable Advanced Shadows", &config.EnableAdvancedShadows) {
		changed = true
	}
	if config.EnableAdvancedShadows {
		imgui.Indent()
		if imgui.SliderFloatV("Intensity##shadow", &config.ShadowIntensity, 0.0, 1.0, "%.2f", 1.0) {
			changed = true
		}
		if imgui.SliderFloatV("Softness##shadow", &config.ShadowSoftness, 0.0, 1.0, "%.2f", 1.0) {
			changed = true
		}
		imgui.Unindent()
	}
	
	// Subsurface Scattering
	imgui.Separator()
	if imgui.Checkbox("Enable Subsurface Scattering", &config.EnableSubsurfaceScattering) {
		changed = true
	}
	if config.EnableSubsurfaceScattering {
		imgui.Indent()
		if imgui.SliderFloatV("Intensity##sss", &config.ScatteringIntensity, 0.0, 1.0, "%.2f", 1.0) {
			changed = true
		}
		if imgui.SliderFloatV("Depth##sss", &config.ScatteringDepth, 0.0, 0.1, "%.4f", 1.0) {
			changed = true
		}
		scatterColor := [3]float32{config.ScatteringColor.X(), config.ScatteringColor.Y(), config.ScatteringColor.Z()}
		if imgui.ColorEdit3V("Color##sss", &scatterColor, 0) {
			config.ScatteringColor = mgl.Vec3{scatterColor[0], scatterColor[1], scatterColor[2]}
			changed = true
		}
		imgui.Unindent()
	}
	
	if changed {
		applyAdvancedConfigToAllModels(config)
	}
}

func renderAdvancedRenderingPostProcess() {
	if eng == nil || eng.GetRenderer() == nil {
		imgui.Text("No renderer available")
		return
	}
	
	openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
	if !ok {
		return
	}
	
	models := openglRenderer.GetModels()
	if len(models) == 0 {
		imgui.Text("No models to configure")
		return
	}
	
	config := getAdvancedConfigFromModel(models[0])
	changed := false
	
	// Bloom - Read from renderer directly
	bloomEnabled := openglRenderer.EnableBloom
	if imgui.Checkbox("Enable Bloom", &bloomEnabled) {
		openglRenderer.EnableBloom = bloomEnabled
		config.EnableBloom = bloomEnabled
		changed = true
		if bloomEnabled {
			logToConsole("Bloom enabled", "info")
		} else {
			logToConsole("Bloom disabled", "info")
		}
	}
	if bloomEnabled {
		imgui.Indent()
		bloomThreshold := openglRenderer.BloomThreshold
		if imgui.SliderFloatV("Threshold##bloom", &bloomThreshold, 0.5, 2.0, "%.2f", 1.0) {
			openglRenderer.BloomThreshold = bloomThreshold
			config.BloomThreshold = bloomThreshold
			changed = true
		}
		imgui.SameLine()
		imgui.Text("Brightness threshold")
		
		bloomIntensity := openglRenderer.BloomIntensity
		if imgui.SliderFloatV("Intensity##bloom", &bloomIntensity, 0.0, 1.0, "%.2f", 1.0) {
			openglRenderer.BloomIntensity = bloomIntensity
			config.BloomIntensity = bloomIntensity
			changed = true
		}
		imgui.SameLine()
		imgui.Text("Bloom strength")
		
		imgui.Unindent()
	}
	
	// Perlin Noise
	imgui.Separator()
	if imgui.Checkbox("Enable Perlin Noise", &config.EnablePerlinNoise) {
		changed = true
	}
	if config.EnablePerlinNoise {
		imgui.Indent()
		if imgui.SliderFloatV("Scale##noise", &config.NoiseScale, 0.0001, 0.01, "%.4f", 1.0) {
			changed = true
		}
		octaves := int32(config.NoiseOctaves)
		if imgui.SliderIntV("Octaves##noise", &octaves, 1, 8, "%d", 1.0) {
			config.NoiseOctaves = int(octaves)
			changed = true
		}
		if imgui.SliderFloatV("Intensity##noise", &config.NoiseIntensity, 0.0, 0.5, "%.3f", 1.0) {
			changed = true
		}
		imgui.Unindent()
	}
	
	// High Quality Filtering
	imgui.Separator()
	if imgui.Checkbox("High Quality Filtering", &config.EnableHighQualityFiltering) {
		changed = true
	}
	if config.EnableHighQualityFiltering {
		imgui.Indent()
		quality := int32(config.FilteringQuality)
		if imgui.SliderIntV("Quality Level", &quality, 1, 3, "%d", 1.0) {
			config.FilteringQuality = int(quality)
			changed = true
		}
		imgui.Unindent()
	}
	
	if changed {
		applyAdvancedConfigToAllModels(config)
	}
}

func renderAdvancedRenderingPerformance() {
	if eng == nil || eng.GetRenderer() == nil {
		imgui.Text("No renderer available")
		return
	}
	
	openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
	if !ok {
		return
	}
	
	models := openglRenderer.GetModels()
	if len(models) == 0 {
		imgui.Text("No models to configure")
		return
	}
	
	config := getAdvancedConfigFromModel(models[0])
	changed := false
	
	// Anti-Aliasing Section
	if imgui.CollapsingHeaderV("Anti-Aliasing", imgui.TreeNodeFlagsDefaultOpen) {
		imgui.Text("Hardware MSAA (Multisample Anti-Aliasing):")
		imgui.Indent()
		
		// Get current MSAA from engine (set at startup)
		currentMSAA := int32(eng.MSAASamples)
		msaaEnabled := currentMSAA > 0
		
		// Checkbox to enable/disable MSAA (doesn't change sample count)
		if imgui.Checkbox("Enable MSAA", &msaaEnabled) {
			if msaaEnabled {
				openglRenderer.EnableMSAA(true)
				logToConsole(fmt.Sprintf("MSAA enabled (%dx)", currentMSAA), "info")
			} else {
				openglRenderer.EnableMSAA(false)
				logToConsole("MSAA disabled", "info")
			}
		}
		
		// Display current sample count (read-only, set at startup)
		imgui.Text(fmt.Sprintf("Sample Count: %dx (set at startup)", currentMSAA))
		imgui.Text("⚠ To change sample count, edit config")
		imgui.Text("and restart the application")
		imgui.Separator()
		
		imgui.Unindent()
		
		imgui.Text("Software FXAA (Fast Approximate AA):")
		imgui.Indent()
		
		// Read FXAA state directly from renderer (not from model config)
		fxaaEnabled := openglRenderer.EnableFXAA
		if imgui.Checkbox("Enable FXAA", &fxaaEnabled) {
			openglRenderer.EnableFXAA = fxaaEnabled
			config.EnableFXAA = fxaaEnabled
			changed = true
			if fxaaEnabled {
				logToConsole("FXAA enabled", "info")
			} else {
				logToConsole("FXAA disabled", "info")
			}
		}
		imgui.Text("Post-processing edge smoothing")
		imgui.Text("Good fallback if MSAA is disabled")
		
		imgui.Unindent()
	}
	
	imgui.Separator()
	
	// Performance Info
	if imgui.CollapsingHeaderV("Automatic LOD System", imgui.TreeNodeFlagsDefaultOpen) {
		imgui.Text("Distance-based optimization enabled:")
		imgui.Bullet()
		imgui.SameLine()
		imgui.Text("SSAO: 2-12 samples")
		imgui.Bullet()
		imgui.SameLine()
		imgui.Text("Volumetric: 4-32 steps")
		imgui.Bullet()
		imgui.SameLine()
		imgui.Text("GI: 2-16 samples")
		imgui.Separator()
		imgui.Text("Close voxels use minimal samples")
		imgui.Text("for maximum performance.")
	}
	
	if changed {
		applyAdvancedConfigToAllModels(config)
	}
}

func getAdvancedConfigFromModel(model *renderer.Model) renderer.AdvancedRenderingConfig {
	// Try to extract config from model's custom uniforms
	config := renderer.DefaultAdvancedRenderingConfig()
	
	if model.CustomUniforms == nil {
		return config
	}
	
	// Extract values from custom uniforms
	if val, ok := model.CustomUniforms["enableClearcoat"].(bool); ok {
		config.EnableClearcoat = val
	}
	if val, ok := model.CustomUniforms["clearcoatRoughness"].(float32); ok {
		config.ClearcoatRoughness = val
	}
	if val, ok := model.CustomUniforms["clearcoatIntensity"].(float32); ok {
		config.ClearcoatIntensity = val
	}
	if val, ok := model.CustomUniforms["enableSheen"].(bool); ok {
		config.EnableSheen = val
	}
	if val, ok := model.CustomUniforms["sheenColor"].(mgl.Vec3); ok {
		config.SheenColor = val
	}
	if val, ok := model.CustomUniforms["sheenRoughness"].(float32); ok {
		config.SheenRoughness = val
	}
	if val, ok := model.CustomUniforms["enableTransmission"].(bool); ok {
		config.EnableTransmission = val
	}
	if val, ok := model.CustomUniforms["transmissionFactor"].(float32); ok {
		config.TransmissionFactor = val
	}
	if val, ok := model.CustomUniforms["enableMultipleScattering"].(bool); ok {
		config.EnableMultipleScattering = val
	}
	if val, ok := model.CustomUniforms["enableEnergyConservation"].(bool); ok {
		config.EnableEnergyConservation = val
	}
	if val, ok := model.CustomUniforms["enableImageBasedLighting"].(bool); ok {
		config.EnableImageBasedLighting = val
	}
	if val, ok := model.CustomUniforms["iblIntensity"].(float32); ok {
		config.IBLIntensity = val
	}
	if val, ok := model.CustomUniforms["enableSSAO"].(bool); ok {
		config.EnableSSAO = val
	}
	if val, ok := model.CustomUniforms["ssaoIntensity"].(float32); ok {
		config.SSAOIntensity = val
	}
	if val, ok := model.CustomUniforms["ssaoRadius"].(float32); ok {
		config.SSAORadius = val
	}
	if val, ok := model.CustomUniforms["ssaoBias"].(float32); ok {
		config.SSAOBias = val
	}
	if val, ok := model.CustomUniforms["ssaoSampleCount"].(int32); ok {
		config.SSAOSampleCount = int(val)
	}
	if val, ok := model.CustomUniforms["enableVolumetricLighting"].(bool); ok {
		config.EnableVolumetricLighting = val
	}
	if val, ok := model.CustomUniforms["volumetricIntensity"].(float32); ok {
		config.VolumetricIntensity = val
	}
	if val, ok := model.CustomUniforms["volumetricSteps"].(int32); ok {
		config.VolumetricSteps = int(val)
	}
	if val, ok := model.CustomUniforms["volumetricScattering"].(float32); ok {
		config.VolumetricScattering = val
	}
	if val, ok := model.CustomUniforms["enableGlobalIllumination"].(bool); ok {
		config.EnableGlobalIllumination = val
	}
	if val, ok := model.CustomUniforms["giIntensity"].(float32); ok {
		config.GIIntensity = val
	}
	if val, ok := model.CustomUniforms["giBounces"].(int32); ok {
		config.GIBounces = int(val)
	}
	if val, ok := model.CustomUniforms["enableBloom"].(bool); ok {
		config.EnableBloom = val
	}
	if val, ok := model.CustomUniforms["bloomThreshold"].(float32); ok {
		config.BloomThreshold = val
	}
	if val, ok := model.CustomUniforms["bloomIntensity"].(float32); ok {
		config.BloomIntensity = val
	}
	if val, ok := model.CustomUniforms["bloomRadius"].(float32); ok {
		config.BloomRadius = val
	}
	if val, ok := model.CustomUniforms["enablePerlinNoise"].(bool); ok {
		config.EnablePerlinNoise = val
	}
	if val, ok := model.CustomUniforms["noiseScale"].(float32); ok {
		config.NoiseScale = val
	}
	if val, ok := model.CustomUniforms["noiseOctaves"].(int32); ok {
		config.NoiseOctaves = int(val)
	}
	if val, ok := model.CustomUniforms["noiseIntensity"].(float32); ok {
		config.NoiseIntensity = val
	}
	if val, ok := model.CustomUniforms["enableShadows"].(bool); ok {
		config.EnableAdvancedShadows = val
	}
	if val, ok := model.CustomUniforms["shadowIntensity"].(float32); ok {
		config.ShadowIntensity = val
	}
	if val, ok := model.CustomUniforms["shadowSoftness"].(float32); ok {
		config.ShadowSoftness = val
	}
	if val, ok := model.CustomUniforms["enableSubsurfaceScattering"].(bool); ok {
		config.EnableSubsurfaceScattering = val
	}
	if val, ok := model.CustomUniforms["scatteringIntensity"].(float32); ok {
		config.ScatteringIntensity = val
	}
	if val, ok := model.CustomUniforms["scatteringDepth"].(float32); ok {
		config.ScatteringDepth = val
	}
	if val, ok := model.CustomUniforms["scatteringColor"].(mgl.Vec3); ok {
		config.ScatteringColor = val
	}
	if val, ok := model.CustomUniforms["enableHighQualityFiltering"].(bool); ok {
		config.EnableHighQualityFiltering = val
	}
	if val, ok := model.CustomUniforms["filteringQuality"].(int32); ok {
		config.FilteringQuality = int(val)
	}
	
	return config
}

func applyAdvancedConfigToAllModels(config renderer.AdvancedRenderingConfig) {
	if eng == nil || eng.GetRenderer() == nil {
		return
	}
	
	openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
	if !ok {
		return
	}
	
	models := openglRenderer.GetModels()
	for _, model := range models {
		// Skip water models as they have their own shader
		if model.Metadata != nil && model.Metadata["type"] == "water" {
			continue
		}
		renderer.ApplyAdvancedRenderingConfig(model, config)
	}
	
	logToConsole("Applied advanced rendering configuration to all models", "info")
}

func applyRenderingPreset(presetName string) {
	var config renderer.AdvancedRenderingConfig
	
	switch presetName {
	case "performance":
		config = renderer.PerformanceRenderingConfig()
		logToConsole("Applied Performance preset", "info")
	case "balanced":
		config = renderer.DefaultAdvancedRenderingConfig()
		logToConsole("Applied Balanced preset", "info")
	case "quality":
		config = renderer.HighQualityRenderingConfig()
		logToConsole("Applied High Quality preset", "info")
	case "voxel":
		config = renderer.VoxelAdvancedRenderingConfig()
		logToConsole("Applied Voxel preset", "info")
	default:
		config = renderer.DefaultAdvancedRenderingConfig()
	}
	
	applyAdvancedConfigToAllModels(config)
	globalAdvancedRenderingEnabled = true
}

