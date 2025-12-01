package editor

import (
	"Gopher3D/internal/behaviour"
	"Gopher3D/internal/renderer"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	mgl "github.com/go-gl/mathgl/mgl32"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/sqweek/dialog"
)

func getDefaultPanelLayout(panelName string, width, height int32) PanelLayout {
	menuBar := float32(20)
	leftW := float32(280)
	rightW := float32(320)
	bottomH := float32(200)

	switch panelName {
	case "hierarchy":
		return PanelLayout{
			PosX:  0,
			PosY:  menuBar,
			SizeX: leftW,
			SizeY: float32(height) - menuBar - bottomH,
		}
	case "inspector":
		return PanelLayout{
			PosX:  float32(width) - rightW,
			PosY:  menuBar,
			SizeX: rightW,
			SizeY: float32(height) - menuBar,
		}
	case "file_explorer":
		return PanelLayout{
			PosX:  0,
			PosY:  float32(height) - bottomH,
			SizeX: leftW,
			SizeY: bottomH,
		}
	case "console":
		return PanelLayout{
			PosX:  leftW,
			PosY:  float32(height) - bottomH,
			SizeX: float32(width) - leftW - rightW,
			SizeY: bottomH,
		}
	case "scene_settings":
		return PanelLayout{
			PosX:  float32(width)/2 - 150,
			PosY:  80,
			SizeX: 300,
			SizeY: 350,
		}
	case "advanced_render":
		return PanelLayout{
			PosX:  float32(width)/2 - 180,
			PosY:  80,
			SizeX: 360,
			SizeY: 400,
		}
	default:
		return PanelLayout{PosX: 100, PosY: 100, SizeX: 280, SizeY: 300}
	}
}

var (
	lastWindowWidth   int32
	lastWindowHeight  int32
	forceLayoutUpdate bool

	// Panel snapping (experimental - disabled by default)
	snapThreshold  = float32(12.0)
	snapEnabled    = false
	panelNeedsSnap = make(map[string]imgui.Vec2)
)

func initializePanelLayouts() {
	if Eng == nil || Eng.Width <= 0 || Eng.Height <= 0 {
		return
	}

	windowResized := lastWindowWidth != Eng.Width || lastWindowHeight != Eng.Height

	if !layoutsInitialized || windowResized {
		hierarchyLayout = getDefaultPanelLayout("hierarchy", Eng.Width, Eng.Height)
		inspectorLayout = getDefaultPanelLayout("inspector", Eng.Width, Eng.Height)
		fileExplorerLayout = getDefaultPanelLayout("file_explorer", Eng.Width, Eng.Height)
		consoleLayout = getDefaultPanelLayout("console", Eng.Width, Eng.Height)
		sceneSettingsLayout = getDefaultPanelLayout("scene_settings", Eng.Width, Eng.Height)
		advancedRenderLayout = getDefaultPanelLayout("advanced_render", Eng.Width, Eng.Height)

		lastWindowWidth = Eng.Width
		lastWindowHeight = Eng.Height
		forceLayoutUpdate = true
		layoutsInitialized = true
	}
}

func getPanelSizeCondition() imgui.Condition {
	if forceLayoutUpdate {
		return imgui.ConditionAlways
	}
	return imgui.ConditionFirstUseEver
}

// snapValue snaps a value to a target if within threshold
func snapValue(value, target float32) float32 {
	if !snapEnabled {
		return value
	}
	if value > target-snapThreshold && value < target+snapThreshold {
		return target
	}
	return value
}

// getSnapTargets returns all edges that panels can snap to
func getSnapTargets() (xTargets, yTargets []float32) {
	menuBarH := float32(20)

	// Window edges
	xTargets = append(xTargets, 0, float32(Eng.Width))
	yTargets = append(yTargets, menuBarH, float32(Eng.Height))

	// Add panel edges as snap targets
	panels := []PanelLayout{hierarchyLayout, inspectorLayout, fileExplorerLayout, consoleLayout}
	for _, p := range panels {
		if p.SizeX > 0 {
			xTargets = append(xTargets, p.PosX, p.PosX+p.SizeX)
			yTargets = append(yTargets, p.PosY, p.PosY+p.SizeY)
		}
	}

	return xTargets, yTargets
}

// snapPanelPosition snaps panel position to nearby edges and returns snapped position
func snapPanelPosition(pos imgui.Vec2, size imgui.Vec2) imgui.Vec2 {
	if !snapEnabled {
		return pos
	}

	xTargets, yTargets := getSnapTargets()

	// Snap left edge
	for _, target := range xTargets {
		pos.X = snapValue(pos.X, target)
	}

	// Snap right edge
	rightEdge := pos.X + size.X
	for _, target := range xTargets {
		snapped := snapValue(rightEdge, target)
		if snapped != rightEdge {
			pos.X = snapped - size.X
		}
	}

	// Snap top edge
	for _, target := range yTargets {
		pos.Y = snapValue(pos.Y, target)
	}

	// Snap bottom edge
	bottomEdge := pos.Y + size.Y
	for _, target := range yTargets {
		snapped := snapValue(bottomEdge, target)
		if snapped != bottomEdge {
			pos.Y = snapped - size.Y
		}
	}

	return pos
}

// checkAndQueueSnap checks if panel needs snapping and queues it for next frame
func checkAndQueueSnap(panelName string, pos imgui.Vec2, size imgui.Vec2, layout *PanelLayout) {
	snappedPos := snapPanelPosition(pos, size)

	// If position changed, queue snap for next frame
	if snappedPos.X != pos.X || snappedPos.Y != pos.Y {
		panelNeedsSnap[panelName] = snappedPos
		layout.PosX = snappedPos.X
		layout.PosY = snappedPos.Y
	}

	// Update layout size
	if size.X != layout.SizeX || size.Y != layout.SizeY {
		layout.SizeX = size.X
		layout.SizeY = size.Y
	}
}

// getPanelPosCondition returns the condition for setting panel position
func getPanelPosCondition(panelName string) imgui.Condition {
	if _, needsSnap := panelNeedsSnap[panelName]; needsSnap {
		delete(panelNeedsSnap, panelName)
		return imgui.ConditionAlways
	}
	if forceLayoutUpdate {
		return imgui.ConditionAlways
	}
	return imgui.ConditionFirstUseEver
}

func RenderEditorUI() {
	// Safety check: ensure engine and renderer are ready
	if Eng == nil || Eng.GetRenderer() == nil {
		return
	}

	openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
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
				if CurrentProject != nil {
					startDir = filepath.Join(CurrentProject.Path, "resources/models")
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
				openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
				if ok {
					models := openglRenderer.GetModels()
					if selectedModelIndex >= 0 && selectedModelIndex < len(models) {
						startDir := "../examples/resources/textures"
						if CurrentProject != nil {
							startDir = filepath.Join(CurrentProject.Path, "resources/textures")
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
			imgui.Separator()
			if imgui.MenuItem("Export Game...") {
				showExportDialog = true
			}
			imgui.Separator()
			if imgui.MenuItem("Exit") {
				Eng.GetWindow().SetShouldClose(true)
			}
			imgui.EndMenu()
		}
		if imgui.BeginMenu("Add") {
			if imgui.MenuItem("Empty GameObject") {
				addEmptyGameObject()
			}
			imgui.Separator()
			if imgui.MenuItem("Mesh...") {
				ShowAddModel = true
			}
			if imgui.MenuItem("Light...") {
				ShowAddLight = true
			}
			if imgui.MenuItem("Camera") {
				AddSceneCamera("")
			}
			imgui.Separator()
			if imgui.MenuItem("Voxel Terrain") {
				ShowAddVoxel = true
			}
			if imgui.MenuItem("Water Plane") {
				createWaterGameObject()
			}
			imgui.EndMenu()
		}
		if imgui.BeginMenu("View") {
			imgui.Text("Core Panels:")
			if imgui.MenuItemV("Scene Hierarchy", "", ShowHierarchy, true) {
				ShowHierarchy = !ShowHierarchy
				SaveConfig()
			}
			if imgui.MenuItemV("Inspector", "", ShowInspector, true) {
				ShowInspector = !ShowInspector
				SaveConfig()
			}
			imgui.Separator()
			imgui.Text("Utility Panels:")
			if imgui.MenuItemV("File Explorer", "", ShowFileExplorer, true) {
				ShowFileExplorer = !ShowFileExplorer
				SaveConfig()
			}
			if imgui.MenuItemV("Console", "", ShowConsole, true) {
				ShowConsole = !ShowConsole
				SaveConfig()
			}
			imgui.Separator()
			imgui.Text("Settings Panels:")
			if imgui.MenuItemV("Scene Settings", "", ShowSceneSettings, true) {
				ShowSceneSettings = !ShowSceneSettings
				SaveConfig()
			}
			if imgui.MenuItemV("Style Editor", "", ShowStyleEditor, true) {
				ShowStyleEditor = !ShowStyleEditor
			}
			if imgui.MenuItemV("Advanced Rendering", "", ShowAdvancedRender, true) {
				ShowAdvancedRender = !ShowAdvancedRender
				SaveConfig()
			}
			imgui.Separator()
			if imgui.MenuItemV("Show Gizmos", "", ShowGizmos, true) {
				ShowGizmos = !ShowGizmos
			}
			imgui.EndMenu()
		}
		if imgui.BeginMenu("Experimental") {
			if imgui.MenuItemV("Panel Snapping", "", snapEnabled, true) {
				snapEnabled = !snapEnabled
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
	if ShowAddModel {
		renderAddModelDialog()
	}

	// Add Light Dialog
	if ShowAddLight {
		renderAddLightDialog()
	}

	// Add Water Dialog
	if ShowAddWater {
		renderAddWaterDialog()
	}

	// Add Voxel Dialog
	if ShowAddVoxel {
		renderAddVoxelDialog()
	}

	// Export Game Dialog
	if showExportDialog {
		renderExportDialog()
	}

	// Script Browser Panel
	if ShowScriptBrowser {
		renderScriptBrowserPanel()
	}

	// Rebuild Modal
	if ShowRebuildModal() {
		renderRebuildModal()
	}

	// File Explorer (Bottom Left)
	if ShowFileExplorer {
		posCondition := getPanelPosCondition("Project")
		sizeCondition := getPanelSizeCondition()
		imgui.SetNextWindowPosV(imgui.Vec2{X: fileExplorerLayout.PosX, Y: fileExplorerLayout.PosY}, posCondition, imgui.Vec2{})
		imgui.SetNextWindowSizeV(imgui.Vec2{X: fileExplorerLayout.SizeX, Y: fileExplorerLayout.SizeY}, sizeCondition)
		imgui.PushStyleVarFloat(imgui.StyleVarWindowRounding, 0)
		imgui.PushStyleVarFloat(imgui.StyleVarWindowBorderSize, 1)
		if imgui.BeginV("Project", &ShowFileExplorer, 0) {
			size := imgui.WindowSize()
			pos := imgui.WindowPos()
			checkAndQueueSnap("Project", pos, size, &fileExplorerLayout)
			renderFileExplorerContent()
		}
		imgui.End()
		imgui.PopStyleVarV(2)
	}

	// Console (Bottom Middle)
	if ShowConsole {
		posCondition := getPanelPosCondition("Console")
		sizeCondition := getPanelSizeCondition()
		imgui.SetNextWindowPosV(imgui.Vec2{X: consoleLayout.PosX, Y: consoleLayout.PosY}, posCondition, imgui.Vec2{})
		imgui.SetNextWindowSizeV(imgui.Vec2{X: consoleLayout.SizeX, Y: consoleLayout.SizeY}, sizeCondition)
		imgui.PushStyleVarFloat(imgui.StyleVarWindowRounding, 0)
		imgui.PushStyleVarFloat(imgui.StyleVarWindowBorderSize, 1)
		if imgui.BeginV("Console", &ShowConsole, imgui.WindowFlagsMenuBar) {
			size := imgui.WindowSize()
			pos := imgui.WindowPos()
			checkAndQueueSnap("Console", pos, size, &consoleLayout)
			renderConsoleContent()
		}
		imgui.End()
		imgui.PopStyleVarV(2)
	}

	// Style Editor (Restored)
	if ShowStyleEditor && Eng != nil {
		imgui.SetNextWindowPosV(imgui.Vec2{X: float32(Eng.Width) - 520, Y: 30}, imgui.ConditionFirstUseEver, imgui.Vec2{})
		imgui.SetNextWindowSizeV(imgui.Vec2{X: 500, Y: 600}, imgui.ConditionFirstUseEver)
		if imgui.BeginV("Style Editor", &ShowStyleEditor, 0) {
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

			// Window/Panel background
			windowBg := style.Color(imgui.StyleColorWindowBg)
			windowBgVec := [3]float32{windowBg.X, windowBg.Y, windowBg.Z}
			if imgui.ColorEdit3V("Panel Background", &windowBgVec, 0) {
				style.SetColor(imgui.StyleColorWindowBg, imgui.Vec4{X: windowBgVec[0], Y: windowBgVec[1], Z: windowBgVec[2], W: windowBg.W})
				style.SetColor(imgui.StyleColorChildBg, imgui.Vec4{X: windowBgVec[0], Y: windowBgVec[1], Z: windowBgVec[2], W: 0.0})
			}

			// Text color
			textColor := style.Color(imgui.StyleColorText)
			textColorVec := [3]float32{textColor.X, textColor.Y, textColor.Z}
			if imgui.ColorEdit3V("Text", &textColorVec, 0) {
				style.SetColor(imgui.StyleColorText, imgui.Vec4{X: textColorVec[0], Y: textColorVec[1], Z: textColorVec[2], W: 1.0})
			}

			// Frame background (input fields, selectables)
			frameBg := style.Color(imgui.StyleColorFrameBg)
			frameBgVec := [3]float32{frameBg.X, frameBg.Y, frameBg.Z}
			if imgui.ColorEdit3V("Input/List Background", &frameBgVec, 0) {
				style.SetColor(imgui.StyleColorFrameBg, imgui.Vec4{X: frameBgVec[0], Y: frameBgVec[1], Z: frameBgVec[2], W: frameBg.W})
				style.SetColor(imgui.StyleColorFrameBgHovered, imgui.Vec4{X: frameBgVec[0] + 0.1, Y: frameBgVec[1] + 0.1, Z: frameBgVec[2] + 0.1, W: 0.4})
			}

			imgui.Separator()
			imgui.Text("Window Border (OS):")
			windowBorderColor := [3]float32{windowBorderR, windowBorderG, windowBorderB}
			if imgui.ColorEdit3V("Window Border", &windowBorderColor, 0) {
				windowBorderR = windowBorderColor[0]
				windowBorderG = windowBorderColor[1]
				windowBorderB = windowBorderColor[2]
				updateWindowBorderColor()
				SaveConfig()
			}

			imgui.Separator()
			imgui.Text("Quick Presets:")
			if imgui.Button("Go Cyan") {
				ApplyDarkTheme()
			}
			imgui.SameLine()
			if imgui.Button("Reset to Dark") {
				imgui.StyleColorsDark()
			}
		}
		imgui.End()
	}

	// Advanced Rendering Options
	if ShowAdvancedRender {
		// Ensure layout is initialized
		if !layoutsInitialized || (advancedRenderLayout.PosX == 0 && advancedRenderLayout.PosY == 0) {
			if Eng.Width > 0 && Eng.Height > 0 {
				advancedRenderLayout = getDefaultPanelLayout("advanced_render", Eng.Width, Eng.Height)
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
				SaveConfig()
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
	if ShowSceneSettings {
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
				SaveConfig()
			}

			if imgui.CollapsingHeaderV("Skybox / Background", imgui.TreeNodeFlagsDefaultOpen) {
				imgui.Text("Background Mode:")
				if imgui.RadioButton("Solid Color", skyboxColorMode) {
					skyboxColorMode = true
					SaveConfig()
				}
				imgui.SameLine()
				if imgui.RadioButton("Skybox Image", !skyboxColorMode) {
					skyboxColorMode = false
					SaveConfig()
				}
				if skyboxColorMode {
					imgui.ColorEdit3V("##skycolor", &skyboxSolidColor, 0)
					if imgui.Button("Apply") {
						// Explicitly set the renderer clear color
						openglRenderer.ClearColorR = skyboxSolidColor[0]
						openglRenderer.ClearColorG = skyboxSolidColor[1]
						openglRenderer.ClearColorB = skyboxSolidColor[2]
						Eng.UpdateSkyboxColor(skyboxSolidColor[0], skyboxSolidColor[1], skyboxSolidColor[2])
						logToConsole(fmt.Sprintf("Background color set to RGB(%.2f, %.2f, %.2f)", skyboxSolidColor[0], skyboxSolidColor[1], skyboxSolidColor[2]), "info")
					}
				} else {
					// Skybox Image Mode
					if imgui.Button("Load Skybox Image...") {
						startDir := "../resources/textures"
						if CurrentProject != nil {
							startDir = filepath.Join(CurrentProject.Path, "resources/textures")
						}

						filename, err := dialog.File().
							SetStartDir(startDir).
							Filter("Images", "png", "jpg", "jpeg").
							Title("Load Skybox Image").
							Load()
						if err == nil && filename != "" {
							Eng.SetSkybox(filename)
							logToConsole("Loaded skybox: "+getFileNameFromPath(filename), "info")
						}
					}
				}
			}

			if imgui.CollapsingHeaderV("Window", imgui.TreeNodeFlagsNone) {
				window := Eng.GetWindow()
				if window != nil {
					w, h := window.GetSize()
					imgui.Text(fmt.Sprintf("Size: %d x %d", w, h))

					imgui.Spacing()
					if imgui.Button("1280 x 720") {
						window.SetSize(1280, 720)
					}
					imgui.SameLine()
					if imgui.Button("1920 x 1080") {
						window.SetSize(1920, 1080)
					}

					imgui.Spacing()
					if imgui.Button("Maximize") {
						window.Maximize()
					}
					imgui.SameLine()
					if imgui.Button("Restore") {
						window.Restore()
					}
				}
			}
		}
		imgui.End()
	}

	// Scene Hierarchy (Left Side, Top)
	if ShowHierarchy {
		posCondition := getPanelPosCondition("Hierarchy")
		sizeCondition := getPanelSizeCondition()
		imgui.SetNextWindowPosV(imgui.Vec2{X: hierarchyLayout.PosX, Y: hierarchyLayout.PosY}, posCondition, imgui.Vec2{})
		imgui.SetNextWindowSizeV(imgui.Vec2{X: hierarchyLayout.SizeX, Y: hierarchyLayout.SizeY}, sizeCondition)
		imgui.PushStyleVarFloat(imgui.StyleVarWindowRounding, 0)
		imgui.PushStyleVarFloat(imgui.StyleVarWindowBorderSize, 1)
		if imgui.BeginV("Hierarchy", &ShowHierarchy, 0) {
			size := imgui.WindowSize()
			pos := imgui.WindowPos()
			checkAndQueueSnap("Hierarchy", pos, size, &hierarchyLayout)
			imgui.Text("Scene Objects:")
			imgui.Separator()

			// GameObjects section - shows ALL GameObjects
			allGameObjects := behaviour.GlobalComponentManager.GetAllGameObjects()
			if imgui.CollapsingHeaderV("[GO] GameObjects", imgui.TreeNodeFlagsDefaultOpen) {
				for i, obj := range allGameObjects {
					imgui.PushID(fmt.Sprintf("gameobj_%d", i))

					// Determine icon based on components
					icon := "[GO]"
					if hasMeshComponent(obj) {
						icon = "[M]"
					}
					if hasVoxelComponent(obj) {
						icon = "[V]"
					}
					if hasWaterComponent(obj) {
						icon = "[W]"
					}

					isSelected := selectedType == "gameobject" && selectedGameObjectIndex == i
					if imgui.SelectableV("  "+icon+" "+obj.Name, isSelected, 0, imgui.Vec2{}) {
						selectedGameObjectIndex = i
						selectedModelIndex = -1
						selectedLightIndex = -1
						selectedCameraIndex = -1
						selectedType = "gameobject"
					}
					if imgui.IsItemHovered() && imgui.IsMouseDoubleClicked(0) {
						if model, ok := obj.GetModel().(*renderer.Model); ok && model != nil {
							focusCameraOnModel(model)
						}
					}
					imgui.PopID()
				}
			}

			// Legacy Models section (models without GameObjects)
			orphanModels := getOrphanModels(models, allGameObjects)
			if len(orphanModels) > 0 {
				if imgui.CollapsingHeaderV("[Legacy] Models", 0) {
					imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.7, Y: 0.7, Z: 0.3, W: 1})
					imgui.Text("(Convert to GameObjects)")
					imgui.PopStyleColor()
					for i, model := range orphanModels {
						imgui.PushID(fmt.Sprintf("legacy_model_%d", i))
					isSelected := selectedType == "model" && selectedModelIndex == i
					if imgui.SelectableV("  "+model.Name, isSelected, 0, imgui.Vec2{}) {
						selectedModelIndex = i
						selectedLightIndex = -1
							selectedGameObjectIndex = -1
						selectedType = "model"
					}
					imgui.PopID()
					}
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
						selectedGameObjectIndex = -1
						selectedCameraIndex = -1
						selectedType = "light"
					}
					imgui.PopID()
				}
			}

			// Cameras section
			if imgui.CollapsingHeaderV("[C] Cameras", imgui.TreeNodeFlagsDefaultOpen) {
				for i, cam := range SceneCameras {
					imgui.PushID(fmt.Sprintf("camera_%d", i))
					displayName := cam.Name
					if displayName == "" {
						displayName = fmt.Sprintf("Camera %d", i)
					}
					activeMarker := ""
					if cam.IsActive {
						activeMarker = " *"
					}
					isSelected := selectedType == "camera" && selectedCameraIndex == i
					if imgui.SelectableV("  [Cam] "+displayName+activeMarker, isSelected, 0, imgui.Vec2{}) {
						selectedCameraIndex = i
						selectedModelIndex = -1
						selectedLightIndex = -1
						selectedGameObjectIndex = -1
						selectedType = "camera"
					}
					imgui.PopID()
				}
			}
		}
		imgui.End()
		imgui.PopStyleVarV(2)
	}

	// Inspector (Right Side, Top)
	if ShowInspector {
		posCondition := getPanelPosCondition("Inspector")
		sizeCondition := getPanelSizeCondition()
		imgui.SetNextWindowPosV(imgui.Vec2{X: inspectorLayout.PosX, Y: inspectorLayout.PosY}, posCondition, imgui.Vec2{})
		imgui.SetNextWindowSizeV(imgui.Vec2{X: inspectorLayout.SizeX, Y: inspectorLayout.SizeY}, sizeCondition)
		imgui.PushStyleVarFloat(imgui.StyleVarWindowRounding, 0)
		imgui.PushStyleVarFloat(imgui.StyleVarWindowBorderSize, 1)
		if imgui.BeginV("Inspector", &ShowInspector, 0) {
			size := imgui.WindowSize()
			pos := imgui.WindowPos()
			checkAndQueueSnap("Inspector", pos, size, &inspectorLayout)

			if selectedType == "model" && selectedModelIndex >= 0 && selectedModelIndex < len(models) {
				model := models[selectedModelIndex]

				imgui.Text("Name:")
				imgui.SameLine()
				imgui.PushItemWidth(-1)
				// Use model name directly as the buffer
				if imgui.InputTextV("##modelName", &model.Name, 0, nil) {
					model.IsDirty = true
				}
				imgui.PopItemWidth()
				imgui.Separator()

				// Delete Model Button
				if imgui.Button("Delete Model") {
					// Check if this is water - need to clean up activeWaterSim
					isWater := false
					if model.Metadata != nil {
						if model.Metadata["type"] == "water" {
							isWater = true
						}
					}

					// Remove the model
					openglRenderer.RemoveModel(model)

					// If water, clean up the simulation and remove from behavior manager
					if isWater && activeWaterSim != nil {
						behaviour.GlobalBehaviourManager.Remove(activeWaterSim)
						activeWaterSim = nil
						logToConsole("Water simulation removed", "info")
					}

					// Remove GameObject
					removeGameObjectForModel(model)

					// Clear selection
					selectedModelIndex = -1
					selectedType = ""
					// Clear from buffer
					delete(modelNameEditBuffer, selectedModelIndex)
					logToConsole(fmt.Sprintf("Deleted model: %s", model.Name), "info")
				}
				imgui.Separator()

				imgui.Spacing()
				if imgui.CollapsingHeaderV("Transform", imgui.TreeNodeFlagsDefaultOpen) {
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

					imgui.Spacing()
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
				}

				// Material editing (initialize if missing for voxels)
				if model.Material == nil {
					model.Material = &renderer.Material{
						Name:         "DefaultMaterial",
						DiffuseColor: [3]float32{0.8, 0.8, 0.8},
						Metallic:     0.0,
						Roughness:    0.9,
						Alpha:        1.0,
					}
				}

				if model.Material != nil {
					imgui.Separator()
					if imgui.CollapsingHeaderV("Material Properties", imgui.TreeNodeFlagsDefaultOpen) {
						diffuse := [3]float32{model.Material.DiffuseColor[0], model.Material.DiffuseColor[1], model.Material.DiffuseColor[2]}
						if imgui.ColorEdit3V("Diffuse Color", &diffuse, 0) {
							model.SetDiffuseColor(diffuse[0], diffuse[1], diffuse[2])
							model.IsDirty = true
						}

						// PBR Material Properties
						imgui.Separator()
						if imgui.SliderFloatV("Metallic", &model.Material.Metallic, 0.0, 1.0, "%.2f", 1.0) {
							model.SetMaterialPBR(model.Material.Metallic, model.Material.Roughness)
							model.IsDirty = true
						}
						if imgui.SliderFloatV("Roughness", &model.Material.Roughness, 0.0, 1.0, "%.2f", 1.0) {
							model.SetMaterialPBR(model.Material.Metallic, model.Material.Roughness)
							model.IsDirty = true
						}
						imgui.Text("Tip: Metallic 1.0 = metal, 0.0 = dielectric")

						// Texture loading
						imgui.Separator()
						if model.Material.TexturePath != "" {
							imgui.Text(fmt.Sprintf("Texture: %s", filepath.Base(model.Material.TexturePath)))
						} else {
							imgui.Text("Texture: None")
						}

						if imgui.Button("Load Texture...") {
							startDir := "../resources/textures"
							if CurrentProject != nil {
								startDir = filepath.Join(CurrentProject.Path, "resources/textures")
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

				// Scripts Section (Unity-style)
				imgui.Spacing()
				if imgui.CollapsingHeaderV("Scripts", imgui.TreeNodeFlagsDefaultOpen) {
					obj := getGameObjectForModel(model)
					if obj != nil {
						// Show attached scripts
						if len(obj.Components) > 0 {
							componentsToRemove := []behaviour.Component{}
							for i, comp := range obj.Components {
								imgui.PushIDInt(i)

								typeName := getComponentTypeName(comp)
								imgui.Bullet()
								imgui.SameLine()
								imgui.Text(typeName)
								imgui.SameLine()
								if imgui.Button("Remove##" + fmt.Sprintf("%d", i)) {
									componentsToRemove = append(componentsToRemove, comp)
								}

								imgui.PopID()
							}

							for _, comp := range componentsToRemove {
								obj.RemoveComponent(comp)
								logToConsole(fmt.Sprintf("Removed script from %s", model.Name), "info")
							}
						} else {
							imgui.Text("(No scripts attached)")
						}

						imgui.Spacing()
						imgui.Separator()

						// Add Script button - opens script browser panel
						if imgui.Button("Add Script") {
							ShowScriptBrowser = true
							scriptBrowserTarget = obj
							scriptBrowserModelTarget = model
								scriptSearchText = ""
							}
						imgui.SameLine()
						imgui.Text("(?)")
						if imgui.IsItemHovered() {
							imgui.BeginTooltip()
							imgui.Text("Scripts are behaviors attached to objects")
							imgui.Text("Create custom scripts in resources/scripts/ folder")
							imgui.EndTooltip()
						}
					} else {
						imgui.Text("ERROR: No GameObject created")
						imgui.Text("This is an internal error")
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
								sim.ApplyChanges()
							}

							// Speed
							speed := sim.WaveSpeedMultiplier
							if imgui.SliderFloatV("Wave Speed", &speed, 0.0, 5.0, "%.2f", 1.0) {
								sim.WaveSpeedMultiplier = speed
								sim.ApplyChanges()
							}

							// Wave Height
							height := sim.WaveHeight
							if imgui.SliderFloatV("Wave Height", &height, 0.1, 5.0, "%.2f", 1.0) {
								sim.WaveHeight = height
								sim.ApplyChanges()
							}

							// Wave Randomness (for stormy seas)
							randomness := sim.WaveRandomness
							if imgui.SliderFloatV("Wave Randomness", &randomness, 0.0, 1.0, "%.2f", 1.0) {
								sim.WaveRandomness = randomness
								sim.ApplyChanges()
							}
							imgui.Text("Tip: Randomness 0.7+ for stormy sea")

							imgui.Separator()
							imgui.Text("Advanced Appearance")

							// Foam / Fog
							foam := sim.FoamEnabled
							if imgui.Checkbox("Enable Atmosphere/Foam", &foam) {
								sim.FoamEnabled = foam
								sim.ApplyChanges()
							}
							if sim.FoamEnabled {
								imgui.Indent()
								foamInt := sim.FoamIntensity
								if imgui.SliderFloatV("Intensity##foam", &foamInt, 0.0, 1.0, "%.2f", 1.0) {
									sim.FoamIntensity = foamInt
									sim.ApplyChanges()
								}
								imgui.Unindent()
							}

							// Caustics
							imgui.Separator()
							caustics := sim.CausticsEnabled
							if imgui.Checkbox("Enable Caustics", &caustics) {
								sim.CausticsEnabled = caustics
								sim.ApplyChanges()
							}
							if sim.CausticsEnabled {
								imgui.Indent()
								causticsInt := sim.CausticsIntensity
								if imgui.SliderFloatV("Intensity##caustics", &causticsInt, 0.0, 1.0, "%.2f", 1.0) {
									sim.CausticsIntensity = causticsInt
									sim.ApplyChanges()
								}
								causticsScale := sim.CausticsScale
								if imgui.SliderFloatV("Scale##caustics", &causticsScale, 0.001, 0.01, "%.4f", 1.0) {
									sim.CausticsScale = causticsScale
									sim.ApplyChanges()
								}
								imgui.Unindent()
							}

							// Specular
							spec := sim.SpecularIntensity
							if imgui.SliderFloatV("Reflectivity", &spec, 0.0, 2.0, "%.2f", 1.0) {
								sim.SpecularIntensity = spec
								sim.ApplyChanges()
							}

							// Normal/Distortion
							norm := sim.NormalStrength
							if imgui.SliderFloatV("Surface Detail", &norm, 0.0, 2.0, "%.2f", 1.0) {
								sim.NormalStrength = norm
								sim.ApplyChanges()
							}

							dist := sim.DistortionStrength
							if imgui.SliderFloatV("Distortion", &dist, 0.0, 1.0, "%.2f", 1.0) {
								sim.DistortionStrength = dist
								sim.ApplyChanges()
							}

							// Shadows
							imgui.Separator()
							shadow := sim.ShadowStrength
							if imgui.SliderFloatV("Shadow Strength", &shadow, 0.0, 1.0, "%.2f", 1.0) {
								sim.ShadowStrength = shadow
								sim.ApplyChanges()
							}

							imgui.Separator()
							if imgui.Button("Load Water Texture...") {
								startDir := "../resources/textures"
								if CurrentProject != nil {
									startDir = filepath.Join(CurrentProject.Path, "resources/textures")
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
											Name:         "WaterMaterial",
											DiffuseColor: [3]float32{1, 1, 1},
											Alpha:        sim.Transparency,
										}
									}
									model.Material.TexturePath = filename
									model.Material.TextureID = 0 // Force reload
									logToConsole("Loaded water texture: "+getFileNameFromPath(filename), "info")
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
					if imgui.DragFloatV("##lposX", &posX, 0.5, 0, 0, "X: %.1f", 0) {
						changed = true
					}
					imgui.SameLine()
					if imgui.DragFloatV("##lposY", &posY, 0.5, 0, 0, "Y: %.1f", 0) {
						changed = true
					}
					imgui.SameLine()
					if imgui.DragFloatV("##lposZ", &posZ, 0.5, 0, 0, "Z: %.1f", 0) {
						changed = true
					}
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
						if imgui.DragFloatV("##ldirX", &dirX, 0.05, -1, 1, "X: %.2f", 0) {
							changed = true
						}
						imgui.SameLine()
						if imgui.DragFloatV("##ldirY", &dirY, 0.05, -1, 1, "Y: %.2f", 0) {
							changed = true
						}
						imgui.SameLine()
						if imgui.DragFloatV("##ldirZ", &dirZ, 0.05, -1, 1, "Z: %.2f", 0) {
							changed = true
						}
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

						if imgui.DragFloatV("Constant", &constant, 0.01, 0, 10, "%.3f", 0) {
							light.ConstantAtten = constant
						}
						if imgui.DragFloatV("Linear", &linear, 0.0001, 0, 1, "%.4f", 0) {
							light.LinearAtten = linear
						}
						if imgui.DragFloatV("Quadratic", &quad, 0.000001, 0, 1, "%.6f", 0) {
							light.QuadraticAtten = quad
						}
					}
				}
			} else if selectedType == "gameobject" && selectedGameObjectIndex >= 0 {
				// GameObject inspector (all GameObjects)
				allGameObjects := behaviour.GlobalComponentManager.GetAllGameObjects()

				if selectedGameObjectIndex < len(allGameObjects) {
					obj := allGameObjects[selectedGameObjectIndex]

					imgui.Text("Name:")
					imgui.SameLine()
					imgui.PushItemWidth(-1)
					imgui.InputTextV("##goName", &obj.Name, 0, nil)
					imgui.PopItemWidth()
					imgui.Separator()

					// Delete button
					if imgui.Button("Delete GameObject") {
						// Remove associated model from renderer if exists
						if model, ok := obj.GetModel().(*renderer.Model); ok && model != nil {
							openglRenderer.RemoveModel(model)
						}
						// Check for water component and clean up
						for _, comp := range obj.Components {
							if waterComp, ok := comp.(*behaviour.WaterComponent); ok {
								if ws, ok := waterComp.Simulation.(*WaterSimulation); ok {
									behaviour.GlobalBehaviourManager.Remove(ws)
									if activeWaterSim == ws {
										activeWaterSim = nil
									}
								}
							}
						}
						behaviour.GlobalComponentManager.UnregisterGameObject(obj)
						selectedGameObjectIndex = -1
						selectedType = ""
						logToConsole(fmt.Sprintf("Deleted GameObject: %s", obj.Name), "info")
					}

					imgui.Separator()
					imgui.Spacing()

					// Transform section
					if imgui.CollapsingHeaderV("Transform", imgui.TreeNodeFlagsDefaultOpen) {
						pos := obj.Transform.Position
						posX, posY, posZ := pos.X(), pos.Y(), pos.Z()
						w := imgui.ContentRegionAvail().X
						imgui.PushItemWidth(w / 3.3)

						imgui.Text("Position")
						changed := false
						if imgui.DragFloatV("##goPosX", &posX, 0.5, 0, 0, "X: %.1f", 0) {
							changed = true
						}
						imgui.SameLine()
						if imgui.DragFloatV("##goPosY", &posY, 0.5, 0, 0, "Y: %.1f", 0) {
							changed = true
						}
						imgui.SameLine()
						if imgui.DragFloatV("##goPosZ", &posZ, 0.5, 0, 0, "Z: %.1f", 0) {
							changed = true
						}
						if changed {
							obj.Transform.SetPosition(mgl.Vec3{posX, posY, posZ})
						}

						// Convert quaternion to euler angles for display
						euler := quatToEuler(obj.Transform.Rotation)
						rotX, rotY, rotZ := euler.X(), euler.Y(), euler.Z()
						imgui.Text("Rotation (Euler)")
						changed = false
						if imgui.DragFloatV("##goRotX", &rotX, 1.0, -180, 180, "X: %.1f", 0) {
							changed = true
						}
						imgui.SameLine()
						if imgui.DragFloatV("##goRotY", &rotY, 1.0, -180, 180, "Y: %.1f", 0) {
							changed = true
						}
						imgui.SameLine()
						if imgui.DragFloatV("##goRotZ", &rotZ, 1.0, -180, 180, "Z: %.1f", 0) {
							changed = true
						}
						if changed {
							obj.Transform.Rotation = eulerToQuat(mgl.Vec3{rotX, rotY, rotZ})
						}

						scale := obj.Transform.Scale
						scaleX, scaleY, scaleZ := scale.X(), scale.Y(), scale.Z()
						imgui.Text("Scale")
						changed = false
						if imgui.DragFloatV("##goScaleX", &scaleX, 0.1, 0.01, 1000, "X: %.2f", 0) {
							changed = true
						}
						imgui.SameLine()
						if imgui.DragFloatV("##goScaleY", &scaleY, 0.1, 0.01, 1000, "Y: %.2f", 0) {
							changed = true
						}
						imgui.SameLine()
						if imgui.DragFloatV("##goScaleZ", &scaleZ, 0.1, 0.01, 1000, "Z: %.2f", 0) {
							changed = true
						}
						if changed {
							obj.Transform.SetScale(mgl.Vec3{scaleX, scaleY, scaleZ})
						}
						imgui.PopItemWidth()
					}

					imgui.Spacing()

					// Components section
					if imgui.CollapsingHeaderV("Components", imgui.TreeNodeFlagsDefaultOpen) {
						components := obj.GetComponents("")
						for i, comp := range components {
							imgui.PushID(fmt.Sprintf("comp_%d", i))
							typeName := behaviour.GetComponentTypeName(comp)
							category := behaviour.GetComponentCategory(comp)

							// Show category icon
							categoryIcon := "[C]"
							switch category {
							case behaviour.ComponentTypeMesh:
								categoryIcon = "[M]"
							case behaviour.ComponentTypeScript:
								categoryIcon = "[S]"
							case behaviour.ComponentTypeWater:
								categoryIcon = "[W]"
							case behaviour.ComponentTypeVoxel:
								categoryIcon = "[V]"
							case behaviour.ComponentTypeLight:
								categoryIcon = "[L]"
							case behaviour.ComponentTypeCamera:
								categoryIcon = "[Cam]"
							}

							expanded := imgui.TreeNodeV(categoryIcon+" "+typeName, 0)
							if expanded {
								renderComponentInspector(comp, obj)
								imgui.Spacing()
								if imgui.Button("Remove##comp") {
									obj.RemoveComponent(comp)
								}
								imgui.TreePop()
							}
							imgui.PopID()
						}

						imgui.Spacing()
						if imgui.Button("Add Component") {
							imgui.OpenPopup("AddComponentGO")
						}

						if imgui.BeginPopup("AddComponentGO") {
							// Built-in Components
							imgui.Text("Built-in Components:")
							imgui.Separator()
							for _, compName := range behaviour.BuiltInComponents() {
								if imgui.Selectable(compName) {
									comp := behaviour.CreateBuiltInComponent(compName)
									if comp != nil {
										obj.AddComponent(comp)
										logToConsole(fmt.Sprintf("Added %s to %s", compName, obj.Name), "info")
									}
									imgui.CloseCurrentPopup()
								}
							}

							imgui.Spacing()
							imgui.Separator()

							// Scripts - opens script browser
							if imgui.Selectable("Scripts...") {
								ShowScriptBrowser = true
								scriptBrowserTarget = obj
								scriptBrowserModelTarget = nil
								scriptSearchText = ""
								imgui.CloseCurrentPopup()
							}
							imgui.EndPopup()
						}
					}
				}
			} else if selectedType == "camera" && selectedCameraIndex >= 0 && selectedCameraIndex < len(SceneCameras) {
				// Camera inspector
				cam := SceneCameras[selectedCameraIndex]

				imgui.Text("Name:")
				imgui.SameLine()
				imgui.PushItemWidth(-1)
				imgui.InputTextV("##camName", &cam.Name, 0, nil)
				imgui.PopItemWidth()
				imgui.Separator()

				// Active camera toggle
				if imgui.Checkbox("Active (Game Camera)", &cam.IsActive) {
					if cam.IsActive {
						SetActiveCamera(selectedCameraIndex)
					}
				}
				imgui.Separator()

				// Delete Camera Button
				if imgui.Button("Delete Camera") {
					RemoveSceneCamera(selectedCameraIndex)
					selectedCameraIndex = -1
					selectedType = ""
				}

				imgui.Separator()

				// Transform section
				if imgui.CollapsingHeaderV("Transform", imgui.TreeNodeFlagsDefaultOpen) {
					w := imgui.ContentRegionAvail().X
					imgui.PushItemWidth(w / 3.3)

					// Position
					posX, posY, posZ := cam.Position.X(), cam.Position.Y(), cam.Position.Z()
					imgui.Text("Position")
					changed := false
					if imgui.DragFloatV("##camPosX", &posX, 0.5, 0, 0, "X: %.1f", 0) {
						changed = true
					}
					imgui.SameLine()
					if imgui.DragFloatV("##camPosY", &posY, 0.5, 0, 0, "Y: %.1f", 0) {
						changed = true
					}
					imgui.SameLine()
					if imgui.DragFloatV("##camPosZ", &posZ, 0.5, 0, 0, "Z: %.1f", 0) {
						changed = true
					}
					if changed {
						cam.Position = mgl.Vec3{posX, posY, posZ}
					}

					// Rotation (Yaw/Pitch)
					imgui.Text("Rotation")
					if imgui.DragFloatV("##camYaw", &cam.Yaw, 1.0, -180, 180, "Yaw: %.1f", 0) {
					}
					imgui.SameLine()
					if imgui.DragFloatV("##camPitch", &cam.Pitch, 1.0, -89, 89, "Pitch: %.1f", 0) {
					}

					imgui.PopItemWidth()
				}

				// Camera settings
				if imgui.CollapsingHeaderV("Camera Settings", imgui.TreeNodeFlagsDefaultOpen) {
					imgui.PushItemWidth(-1)

					if imgui.SliderFloatV("FOV", &cam.Fov, 10, 120, "%.1f", 0) {
						cam.UpdateProjection()
					}
					if imgui.SliderFloatV("Speed", &cam.Speed, 1, 500, "%.1f", 0) {
					}
					if imgui.SliderFloatV("Sensitivity", &cam.Sensitivity, 0.01, 1.0, "%.2f", 0) {
					}
					imgui.Checkbox("Invert Mouse", &cam.InvertMouse)

					imgui.Separator()
					imgui.Text("Clipping Planes")
					if imgui.DragFloatV("Near", &cam.Near, 0.01, 0.01, 100, "%.2f", 0) {
						cam.UpdateProjection()
					}
					if imgui.DragFloatV("Far", &cam.Far, 10, 100, 100000, "%.0f", 0) {
						cam.UpdateProjection()
					}

					imgui.PopItemWidth()
				}

				// Copy from Editor Camera button
				imgui.Separator()
				if imgui.Button("Copy from Editor Camera") {
					if Eng.Camera != nil {
						cam.Position = Eng.Camera.Position
						cam.Yaw = Eng.Camera.Yaw
						cam.Pitch = Eng.Camera.Pitch
						cam.Fov = Eng.Camera.Fov
						cam.Near = Eng.Camera.Near
						cam.Far = Eng.Camera.Far
						cam.Speed = Eng.Camera.Speed
						cam.UpdateProjection()
						logToConsole("Copied editor camera settings to "+cam.Name, "info")
					}
				}
				imgui.SameLine()
				if imgui.Button("Preview") {
					// Temporarily set the engine camera to this camera's position
					if Eng.Camera != nil {
						Eng.Camera.Position = cam.Position
						Eng.Camera.Yaw = cam.Yaw
						Eng.Camera.Pitch = cam.Pitch
						logToConsole("Previewing camera: "+cam.Name, "info")
					}
				}
			}
		}
		imgui.End()
		imgui.PopStyleVarV(2)
	}

	if ShowDemoWindow {
		imgui.ShowDemoWindow(&ShowDemoWindow)
	}

	forceLayoutUpdate = false
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
		if imgui.MenuItem("Copy All") {
			copyConsoleToClipboard()
		}
		imgui.Checkbox("Auto-scroll", &consoleAutoScroll)
		imgui.EndMenuBar()
	}

	footerHeight := imgui.FrameHeightWithSpacing()
	imgui.BeginChildV("ConsoleScrollRegion", imgui.Vec2{X: 0, Y: -footerHeight}, true, 0)
	for i, entry := range consoleLines {
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

		// Make text selectable for copying
		imgui.PushID(fmt.Sprintf("console_%d", i))
		if imgui.SelectableV(entry.Message, false, imgui.SelectableFlagsAllowDoubleClick, imgui.Vec2{}) {
			if imgui.IsMouseDoubleClicked(0) {
				Platform.SetClipboardText(entry.Message)
				logToConsole("Copied to clipboard", "info")
			}
		}
		imgui.PopID()
		imgui.PopStyleColor()
	}
	if consoleAutoScroll && imgui.ScrollY() >= imgui.ScrollMaxY() {
		imgui.SetScrollHereY(1.0)
	}
	imgui.EndChild()

	imgui.Separator()
	imgui.PushItemWidth(-1)

	if imgui.InputTextV("##ConsoleInput", &consoleInput, imgui.InputTextFlagsEnterReturnsTrue, nil) {
		if consoleInput != "" {
			executeConsoleCommand(consoleInput)
			consoleInput = ""
			imgui.SetKeyboardFocusHere()
		}
	}
	imgui.PopItemWidth()
}

func copyConsoleToClipboard() {
	var sb strings.Builder
	for _, entry := range consoleLines {
		sb.WriteString(entry.Message)
		sb.WriteString("\n")
	}
	Platform.SetClipboardText(sb.String())
	logToConsole("Console output copied to clipboard", "info")
}

// Advanced Rendering UI Helper Functions

func renderAdvancedRenderingPBR() {
	if Eng == nil || Eng.GetRenderer() == nil {
		imgui.Text("No renderer available")
		return
	}

	openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
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
	clearcoatEnabled := config.EnableClearcoat
	if imgui.Checkbox("Enable Clearcoat", &clearcoatEnabled) {
		config.EnableClearcoat = clearcoatEnabled
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
	sheenEnabled := config.EnableSheen
	if imgui.Checkbox("Enable Sheen", &sheenEnabled) {
		config.EnableSheen = sheenEnabled
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
	transmissionEnabled := config.EnableTransmission
	if imgui.Checkbox("Enable Transmission", &transmissionEnabled) {
		config.EnableTransmission = transmissionEnabled
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
	if Eng == nil || Eng.GetRenderer() == nil {
		imgui.Text("No renderer available")
		return
	}

	openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
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
	ssaoEnabled := config.EnableSSAO
	if imgui.Checkbox("Enable SSAO", &ssaoEnabled) {
		config.EnableSSAO = ssaoEnabled
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
	volumetricEnabled := config.EnableVolumetricLighting
	if imgui.Checkbox("Enable Volumetric Lighting", &volumetricEnabled) {
		config.EnableVolumetricLighting = volumetricEnabled
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
	giEnabled := config.EnableGlobalIllumination
	if imgui.Checkbox("Enable Global Illumination", &giEnabled) {
		config.EnableGlobalIllumination = giEnabled
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
	if Eng == nil || Eng.GetRenderer() == nil {
		imgui.Text("No renderer available")
		return
	}

	openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
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

	// Note: Perlin Noise and High Quality Filtering removed - not implemented in shaders

	if changed {
		applyAdvancedConfigToAllModels(config)
	}
}

func renderAdvancedRenderingPerformance() {
	if Eng == nil || Eng.GetRenderer() == nil {
		imgui.Text("No renderer available")
		return
	}

	openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
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

	if imgui.CollapsingHeaderV("Anti-Aliasing", imgui.TreeNodeFlagsDefaultOpen) {
		imgui.Text("MSAA (Hardware):")
		imgui.Indent()

		msaaEnabled := openglRenderer.EnableMSAAState

		if imgui.Checkbox("Enable MSAA", &msaaEnabled) {
			openglRenderer.EnableMSAA(msaaEnabled)
			if msaaEnabled {
				logToConsole(fmt.Sprintf("MSAA enabled (%dx)", Eng.MSAASamples), "info")
			} else {
				logToConsole("MSAA disabled", "info")
			}
		}

		imgui.Text(fmt.Sprintf("Samples: %dx (restart to change)", Eng.MSAASamples))
		imgui.Separator()

		imgui.Unindent()

		imgui.Text("Software FXAA (Fast Approximate AA):")
		imgui.Indent()

		// Read FXAA state directly from renderer (not from model config)
		// FXAA is a post-processing effect, not a per-model setting
		fxaaEnabled := openglRenderer.EnableFXAA
		if imgui.Checkbox("Enable FXAA", &fxaaEnabled) {
			if fxaaEnabled != openglRenderer.EnableFXAA {
				openglRenderer.EnableFXAA = fxaaEnabled
				if fxaaEnabled {
					logToConsole("FXAA enabled", "info")
				} else {
					logToConsole("FXAA disabled", "info")
				}
			}
		}
		imgui.Text("Post-processing edge smoothing")
		imgui.Text("(FXAA is OFF by default)")
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
	if Eng == nil || Eng.GetRenderer() == nil {
		return
	}

	openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
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

// renderComponentInspector renders the inspector UI for a specific component
func renderComponentInspector(comp behaviour.Component, obj *behaviour.GameObject) {
	switch c := comp.(type) {
	case *behaviour.MeshComponent:
		renderMeshComponentInspector(c)
	case *behaviour.WaterComponent:
		renderWaterComponentInspector(c)
	case *behaviour.VoxelTerrainComponent:
		renderVoxelComponentInspector(c, obj)
	case *behaviour.LightComponent:
		renderLightComponentInspector(c)
	case *behaviour.CameraComponent:
		renderCameraComponentInspector(c)
	case *behaviour.ScriptComponent:
		imgui.Text("Script: " + c.ScriptName)
	default:
		imgui.Text("No inspector for this component")
	}
}

func renderMeshComponentInspector(c *behaviour.MeshComponent) {
	imgui.Text("Mesh Path:")
	imgui.SameLine()
	imgui.PushItemWidth(-1)
	imgui.InputTextV("##meshPath", &c.MeshPath, imgui.InputTextFlagsReadOnly, nil)
	imgui.PopItemWidth()

	if c.Loaded {
		imgui.Text("Status: Loaded")
	} else {
		imgui.Text("Status: Not loaded")
	}

	imgui.Separator()
	imgui.Text("Material:")

	diffuse := c.DiffuseColor
	if imgui.ColorEdit3V("Diffuse", &diffuse, 0) {
		c.DiffuseColor = diffuse
		applyMeshMaterial(c)
	}

	specular := c.SpecularColor
	if imgui.ColorEdit3V("Specular", &specular, 0) {
		c.SpecularColor = specular
		applyMeshMaterial(c)
	}

	if imgui.SliderFloatV("Metallic", &c.Metallic, 0, 1, "%.2f", 0) {
		applyMeshMaterial(c)
	}
	if imgui.SliderFloatV("Roughness", &c.Roughness, 0, 1, "%.2f", 0) {
		applyMeshMaterial(c)
	}
	if imgui.SliderFloatV("Alpha", &c.Alpha, 0, 1, "%.2f", 0) {
		applyMeshMaterial(c)
	}
}

func applyMeshMaterial(c *behaviour.MeshComponent) {
	if model, ok := c.Model.(*renderer.Model); ok && model != nil {
		model.SetDiffuseColor(c.DiffuseColor[0], c.DiffuseColor[1], c.DiffuseColor[2])
		model.SetSpecularColor(c.SpecularColor[0], c.SpecularColor[1], c.SpecularColor[2])
		model.SetMaterialPBR(c.Metallic, c.Roughness)
		if model.Material != nil {
			model.Material.Alpha = c.Alpha
		}
	}
}

func renderWaterComponentInspector(c *behaviour.WaterComponent) {
	changed := false

	// Size and waves (these require regeneration, not real-time)
	imgui.Text("Size & Waves:")
	imgui.DragFloatV("Ocean Size", &c.OceanSize, 10, 100, 10000, "%.0f", 0)
	if imgui.DragFloatV("Wave Amplitude", &c.BaseAmplitude, 0.1, 0.1, 20, "%.1f", 0) {
		changed = true
	}
	if imgui.SliderFloatV("Wave Speed", &c.WaveSpeedMultiplier, 0.1, 5.0, "%.1f", 0) {
		changed = true
	}

	imgui.Separator()
	imgui.Text("Appearance:")
	color := c.WaterColor
	if imgui.ColorEdit3V("Water Color", &color, 0) {
		c.WaterColor = color
		changed = true
	}

	if imgui.SliderFloatV("Transparency", &c.Transparency, 0, 1, "%.2f", 0) {
		changed = true
	}

	imgui.Separator()
	imgui.Text("Effects:")
	if imgui.Checkbox("Foam", &c.FoamEnabled) {
		changed = true
	}
	if c.FoamEnabled {
		if imgui.SliderFloatV("Foam Intensity", &c.FoamIntensity, 0, 1, "%.2f", 0) {
			changed = true
		}
	}
	if imgui.Checkbox("Caustics", &c.CausticsEnabled) {
		changed = true
	}
	if c.CausticsEnabled {
		if imgui.SliderFloatV("Caustics Intensity", &c.CausticsIntensity, 0, 1, "%.2f", 0) {
			changed = true
		}
		if imgui.SliderFloatV("Caustics Scale", &c.CausticsScale, 0.1, 5, "%.1f", 0) {
			changed = true
		}
	}

	imgui.Separator()
	imgui.Text("Lighting:")
	if imgui.SliderFloatV("Specular Intensity", &c.SpecularIntensity, 0, 5, "%.1f", 0) {
		changed = true
	}
	if imgui.SliderFloatV("Normal Strength", &c.NormalStrength, 0, 2, "%.1f", 0) {
		changed = true
	}
	if imgui.SliderFloatV("Shadow Strength", &c.ShadowStrength, 0, 1, "%.2f", 0) {
		changed = true
	}
	if imgui.SliderFloatV("Distortion", &c.DistortionStrength, 0, 0.5, "%.2f", 0) {
		changed = true
	}

	// Real-time sync to simulation
	if changed && c.Generated {
		SyncWaterComponentToSimulation(c)
	}

	imgui.Separator()
	if c.Generated {
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.5, Y: 0.8, Z: 0.5, W: 1})
		imgui.Text("Status: Active (real-time editing)")
		imgui.PopStyleColor()
	} else {
		imgui.Text("Status: Not generated")
	}
}

func renderVoxelComponentInspector(c *behaviour.VoxelTerrainComponent, obj *behaviour.GameObject) {
	// Biome selection
	biomeNames := []string{"Plains", "Mountains", "Desert", "Islands", "Caves"}
	if imgui.BeginCombo("Biome", biomeNames[c.Biome]) {
		for i := int32(0); i < 5; i++ {
			if imgui.SelectableV(biomeNames[i], c.Biome == i, 0, imgui.Vec2{}) {
				c.Biome = i
			}
		}
		imgui.EndCombo()
	}

	imgui.Separator()
	imgui.DragFloatV("Scale", &c.Scale, 0.001, 0.001, 10.0, "%.3f", 0)
	imgui.DragFloatV("Amplitude", &c.Amplitude, 0.5, 1.0, 500.0, "%.1f", 0)
	imgui.DragInt("Seed", &c.Seed)
	imgui.DragFloatV("Cave Threshold", &c.Threshold, 0.01, -2.0, 2.0, "%.2f", 0)
	imgui.SliderInt("Octaves", &c.Octaves, 1, 12)
	imgui.SliderInt("Chunk Size", &c.ChunkSize, 8, 128)
	imgui.SliderInt("World Size (Chunks)", &c.WorldSize, 1, 32)
	imgui.DragFloatV("Tree Density", &c.TreeDensity, 0.001, 0.0, 1.0, "%.3f", 0)

	imgui.Separator()
	imgui.Text("Colors:")
	grass := c.GrassColor
	if imgui.ColorEdit3V("Grass", &grass, 0) {
		c.GrassColor = grass
	}
	dirt := c.DirtColor
	if imgui.ColorEdit3V("Dirt", &dirt, 0) {
		c.DirtColor = dirt
	}
	stone := c.StoneColor
	if imgui.ColorEdit3V("Stone", &stone, 0) {
		c.StoneColor = stone
	}
	sand := c.SandColor
	if imgui.ColorEdit3V("Sand", &sand, 0) {
		c.SandColor = sand
	}
	wood := c.WoodColor
	if imgui.ColorEdit3V("Wood (Trunk)", &wood, 0) {
		c.WoodColor = wood
	}
	leaves := c.LeavesColor
	if imgui.ColorEdit3V("Leaves", &leaves, 0) {
		c.LeavesColor = leaves
	}

	imgui.Separator()
	if c.Generated {
		imgui.Text("Status: Generated")
		if imgui.Button("Regenerate Terrain") {
			regenerateVoxelTerrainForComponent(c, obj)
		}
	} else {
		if imgui.Button("Generate Terrain") {
			generateVoxelTerrainForGameObject(c, obj)
		}
	}
}

func renderLightComponentInspector(c *behaviour.LightComponent) {
	lightModes := []string{"directional", "point", "spot"}
	currentIdx := 0
	for i, m := range lightModes {
		if m == c.LightMode {
			currentIdx = i
			break
		}
	}
	if imgui.BeginCombo("Mode", lightModes[currentIdx]) {
		for i, m := range lightModes {
			if imgui.SelectableV(m, i == currentIdx, 0, imgui.Vec2{}) {
				c.LightMode = m
			}
		}
		imgui.EndCombo()
	}

	color := c.Color
	if imgui.ColorEdit3V("Color", &color, 0) {
		c.Color = color
	}

	imgui.SliderFloatV("Intensity", &c.Intensity, 0, 10, "%.1f", 0)
	imgui.SliderFloatV("Ambient", &c.AmbientStrength, 0, 1, "%.2f", 0)

	if c.LightMode == "point" || c.LightMode == "spot" {
		imgui.DragFloatV("Range", &c.Range, 1, 1, 1000, "%.0f", 0)
	}
}

func renderCameraComponentInspector(c *behaviour.CameraComponent) {
	imgui.SliderFloatV("FOV", &c.FOV, 10, 120, "%.0f", 0)
	imgui.DragFloatV("Near", &c.Near, 0.01, 0.01, 100, "%.2f", 0)
	imgui.DragFloatV("Far", &c.Far, 10, 100, 100000, "%.0f", 0)
	imgui.Checkbox("Main Camera", &c.IsMain)
}

// Helper functions for hierarchy display

func hasMeshComponent(obj *behaviour.GameObject) bool {
	for _, comp := range obj.Components {
		if _, ok := comp.(*behaviour.MeshComponent); ok {
			return true
		}
	}
	return false
}

func hasVoxelComponent(obj *behaviour.GameObject) bool {
	for _, comp := range obj.Components {
		if _, ok := comp.(*behaviour.VoxelTerrainComponent); ok {
			return true
		}
	}
	return false
}

func hasWaterComponent(obj *behaviour.GameObject) bool {
	for _, comp := range obj.Components {
		if _, ok := comp.(*behaviour.WaterComponent); ok {
			return true
		}
	}
	return false
}

func getOrphanModels(models []*renderer.Model, gameObjects []*behaviour.GameObject) []*renderer.Model {
	orphans := make([]*renderer.Model, 0)
	for _, model := range models {
		hasGameObject := false
		for _, obj := range gameObjects {
			if obj.GetModel() == model {
				hasGameObject = true
				break
			}
		}
		if !hasGameObject {
			orphans = append(orphans, model)
		}
	}
	return orphans
}

// renderScriptBrowserPanel renders the script browser side panel
func renderScriptBrowserPanel() {
	// Position on the right side of the screen
	panelWidth := float32(350)
	panelHeight := float32(500)

	if Eng != nil {
		imgui.SetNextWindowPosV(imgui.Vec2{X: float32(Eng.Width) - panelWidth - 20, Y: 50}, imgui.ConditionFirstUseEver, imgui.Vec2{})
	}
	imgui.SetNextWindowSizeV(imgui.Vec2{X: panelWidth, Y: panelHeight}, imgui.ConditionFirstUseEver)

	if imgui.BeginV("Script Browser", &ShowScriptBrowser, 0) {
		// Target info
		targetName := "None"
		if scriptBrowserTarget != nil {
			targetName = scriptBrowserTarget.Name
		} else if scriptBrowserModelTarget != nil {
			targetName = scriptBrowserModelTarget.Name
		}
		imgui.Text("Target: " + targetName)
		imgui.Separator()

		// Search bar
		imgui.Text("Search:")
		imgui.PushItemWidth(-1)
		imgui.InputTextWithHint("##scriptBrowserSearch", "Type to search...", &scriptSearchText)
		imgui.PopItemWidth()

		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()

		// Import button
		if imgui.Button("Import Script...") {
			startDir := "."
			if CurrentProject != nil {
				startDir = filepath.Join(CurrentProject.Path, "resources", "scripts")
			}
			filename, err := dialog.File().
				SetStartDir(startDir).
				Filter("Go Scripts", "go").
				Title("Import Script").
				Load()
			if err == nil && filename != "" {
				// Copy script to project scripts folder if not already there
				if CurrentProject != nil {
					scriptsDir := filepath.Join(CurrentProject.Path, "resources", "scripts")
					os.MkdirAll(scriptsDir, 0755)
					destPath := filepath.Join(scriptsDir, filepath.Base(filename))
					if filename != destPath {
						copyScriptFile(filename, destPath)
						logToConsole(fmt.Sprintf("Imported script: %s", filepath.Base(filename)), "info")
						RefreshProjectScripts()
					}
				}
			}
		}
		imgui.SameLine()
		if imgui.Button("Refresh") {
			RefreshProjectScripts()
		}

		imgui.Spacing()
		imgui.Separator()

		// Rebuild Editor button - always visible
		imgui.PushStyleColor(imgui.StyleColorButton, imgui.Vec4{X: 0.2, Y: 0.4, Z: 0.2, W: 1})
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, imgui.Vec4{X: 0.3, Y: 0.6, Z: 0.3, W: 1})
		if imgui.ButtonV("Rebuild Editor", imgui.Vec2{X: -1, Y: 30}) {
			ClearRebuildState()
			TriggerEditorRebuild()
		}
		imgui.PopStyleColorV(2)
		if imgui.IsItemHovered() {
			imgui.BeginTooltip()
			imgui.Text("Recompile editor with new/modified scripts")
			imgui.Text("Scene state will be preserved")
			imgui.EndTooltip()
		}
		imgui.Spacing()

		// Scripts list in a scrollable region
		imgui.BeginChildV("ScriptList", imgui.Vec2{X: 0, Y: -60}, true, 0)

		// Get all scripts
		availableScripts := behaviour.GetAvailableScripts()
		searchLower := strings.ToLower(scriptSearchText)
		visibleCount := 0

		// Available Scripts section - these are ready to use
		if len(availableScripts) > 0 {
			if imgui.CollapsingHeaderV("Available Scripts", imgui.TreeNodeFlagsDefaultOpen) {
				for _, scriptName := range availableScripts {
					if scriptSearchText == "" || strings.Contains(strings.ToLower(scriptName), searchLower) {
						if imgui.SelectableV(scriptName, false, 0, imgui.Vec2{}) {
							addScriptToTarget(scriptName)
							ShowScriptBrowser = false
						}
						if imgui.IsItemHovered() {
							imgui.BeginTooltip()
							imgui.Text("Click to add to GameObject")
							imgui.EndTooltip()
						}
						visibleCount++
					}
				}
			}
		}

		if visibleCount == 0 && scriptSearchText != "" {
			imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.5, Y: 0.5, Z: 0.5, W: 1})
			imgui.Text("No matching scripts found")
			imgui.PopStyleColor()
		}

		if len(availableScripts) == 0 {
			imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.5, Y: 0.5, Z: 0.5, W: 1})
			imgui.Text("No scripts available")
			imgui.Text("Create one below")
			imgui.PopStyleColor()
		}

		imgui.EndChild()

		// Bottom section - Create/Import
		imgui.Separator()
		imgui.Spacing()

		if imgui.Button("+ Create New Script") {
			imgui.OpenPopup("CreateScriptPopup")
		}
		imgui.SameLine()
		if imgui.Button("Import Script") {
			filename, err := dialog.File().
				Filter("Go Scripts", "go").
				Title("Import Script").
				Load()
			if err == nil && filename != "" {
				err := AddScriptToEngine(filename)
				if err != nil {
					logToConsole(fmt.Sprintf("Failed to import: %v", err), "error")
				}
			}
		}
		imgui.SameLine()
		if imgui.Button("Close") {
			ShowScriptBrowser = false
			scriptBrowserTarget = nil
			scriptBrowserModelTarget = nil
		}

		// Create script popup
		if imgui.BeginPopupModal("CreateScriptPopup") {
			imgui.Text("Enter script name:")
			imgui.InputTextV("##newscriptname", &newScriptName, 0, nil)
			imgui.Spacing()

			if imgui.Button("Create") && newScriptName != "" {
				err := CreateAndAddScript(newScriptName)
				if err != nil {
					logToConsole(fmt.Sprintf("Failed to create script: %v", err), "error")
				}
				newScriptName = ""
				imgui.CloseCurrentPopup()
			}
			imgui.SameLine()
			if imgui.Button("Cancel") {
				newScriptName = ""
				imgui.CloseCurrentPopup()
			}
			imgui.EndPopup()
		}
	}
	imgui.End()
}

// addScriptToTarget adds a script to the current target
func addScriptToTarget(scriptName string) {
	if scriptBrowserTarget == nil && scriptBrowserModelTarget == nil {
		logToConsole("No target selected for script", "error")
		return
	}

	rawScript := behaviour.CreateScript(scriptName)
	if rawScript == nil {
		logToConsole(fmt.Sprintf("Failed to create script: %s", scriptName), "error")
		return
	}

	// Wrap the raw script in a ScriptComponent so it has a proper name
	scriptComp := behaviour.NewScriptComponent(scriptName, rawScript)

	if scriptBrowserTarget != nil {
		scriptBrowserTarget.AddComponent(scriptComp)
		logToConsole(fmt.Sprintf("Added %s to %s", scriptName, scriptBrowserTarget.Name), "info")
	} else if scriptBrowserModelTarget != nil {
		obj := getGameObjectForModel(scriptBrowserModelTarget)
		if obj != nil {
			obj.AddComponent(scriptComp)
			logToConsole(fmt.Sprintf("Added %s to %s", scriptName, scriptBrowserModelTarget.Name), "info")
		}
	}
}

// copyScriptFile copies a script file to the destination
func copyScriptFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// renderRebuildModal renders the editor rebuild progress modal
func renderRebuildModal() {
	// Center the modal
	if Eng != nil {
		imgui.SetNextWindowPosV(imgui.Vec2{X: float32(Eng.Width) / 2, Y: float32(Eng.Height) / 2}, imgui.ConditionAlways, imgui.Vec2{X: 0.5, Y: 0.5})
	}
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 500, Y: 350}, imgui.ConditionAlways)

	flags := imgui.WindowFlagsNoResize | imgui.WindowFlagsNoMove | imgui.WindowFlagsNoCollapse
	if imgui.BeginV("Rebuild Editor", nil, flags) {
		if IsRebuilding() {
			// Show progress
			imgui.Text("Building editor...")
			imgui.Spacing()

			// Progress indicator
			imgui.PushStyleColor(imgui.StyleColorFrameBg, imgui.Vec4{X: 0.1, Y: 0.1, Z: 0.1, W: 1})
			imgui.BeginChildV("BuildOutput", imgui.Vec2{X: -1, Y: 200}, true, 0)
			output := GetRebuildOutput()
			// Use Text with word wrapping via PushTextWrapPos
			imgui.PushTextWrapPosV(imgui.ContentRegionAvail().X)
			imgui.Text(output)
			imgui.PopTextWrapPos()
			imgui.EndChild()
			imgui.PopStyleColor()

			imgui.Spacing()
			imgui.Text("Please wait...")
		} else if WasRebuildSuccessful() {
			// Success
			imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.2, Y: 0.9, Z: 0.2, W: 1})
			imgui.Text("Build Successful!")
			imgui.PopStyleColor()
			imgui.Spacing()

			imgui.PushStyleColor(imgui.StyleColorFrameBg, imgui.Vec4{X: 0.1, Y: 0.1, Z: 0.1, W: 1})
			imgui.BeginChildV("BuildOutput", imgui.Vec2{X: -1, Y: 180}, true, 0)
			output := GetRebuildOutput()
			imgui.PushTextWrapPosV(imgui.ContentRegionAvail().X)
			imgui.Text(output)
			imgui.PopTextWrapPos()
			imgui.EndChild()
			imgui.PopStyleColor()

			imgui.Spacing()
			imgui.Text("Click 'Restart' to apply the new scripts.")
			imgui.Text("Your scene will be restored after restart.")
			imgui.Spacing()

			imgui.PushStyleColor(imgui.StyleColorButton, imgui.Vec4{X: 0.2, Y: 0.5, Z: 0.2, W: 1})
			imgui.PushStyleColor(imgui.StyleColorButtonHovered, imgui.Vec4{X: 0.3, Y: 0.7, Z: 0.3, W: 1})
			if imgui.ButtonV("Restart Editor", imgui.Vec2{X: 150, Y: 30}) {
				SaveSceneAndRestart()
			}
			imgui.PopStyleColorV(2)

			imgui.SameLine()
			if imgui.ButtonV("Close", imgui.Vec2{X: 100, Y: 30}) {
				SetShowRebuildModal(false)
			}
		} else if GetRebuildError() != "" {
			// Error
			imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.9, Y: 0.2, Z: 0.2, W: 1})
			imgui.Text("Build Failed!")
			imgui.PopStyleColor()
			imgui.Spacing()

			imgui.PushStyleColor(imgui.StyleColorFrameBg, imgui.Vec4{X: 0.15, Y: 0.05, Z: 0.05, W: 1})
			imgui.BeginChildV("BuildOutput", imgui.Vec2{X: -1, Y: 200}, true, 0)
			output := GetRebuildOutput()
			imgui.PushTextWrapPosV(imgui.ContentRegionAvail().X)
			imgui.Text(output)
			imgui.PopTextWrapPos()
			imgui.EndChild()
			imgui.PopStyleColor()

			imgui.Spacing()
			imgui.Text("Fix the errors in your scripts and try again.")
			imgui.Spacing()

			if imgui.ButtonV("Retry", imgui.Vec2{X: 100, Y: 30}) {
				ClearRebuildState()
				TriggerEditorRebuild()
			}
			imgui.SameLine()
			if imgui.ButtonV("Close", imgui.Vec2{X: 100, Y: 30}) {
				SetShowRebuildModal(false)
			}
		} else {
			// Initial state - shouldn't normally be visible
			imgui.Text("Preparing build...")
		}
	}
	imgui.End()
}
