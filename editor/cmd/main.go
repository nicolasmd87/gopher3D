package main

import (
	"Gopher3D/editor/internal"
	"Gopher3D/editor/platforms"
	"Gopher3D/editor/renderers"
	"Gopher3D/internal/engine"
	"fmt"
	"runtime"

	"github.com/inkyblackness/imgui-go/v4"

	// Import scripts package to register all scripts via init()
	_ "Gopher3D/scripts"
)

func main() {
	runtime.LockOSThread()

	fmt.Println("===========================================")
	fmt.Println("   Gopher3D Editor with ImGui				")
	fmt.Println("===========================================")

	context := imgui.CreateContext(nil)
	defer context.Destroy()
	defer editor.SaveConfig()

	editor.Eng = engine.NewGopher(engine.OPENGL)
	editor.Eng.Width = 1280
	editor.Eng.Height = 720
	editor.Eng.WindowDecorated = true

	editor.Eng.SetOnRenderCallback(func(deltaTime float64) {
		if !editor.ImguiInitialized && editor.Eng.GetWindow() != nil {
			initializeImGui()
		}

		if editor.ImguiInitialized && !editor.SceneSetup && !editor.ShowProjectManager && editor.CurrentProject == nil {
			editor.ShowProjectManager = true
		}

		if editor.ImguiInitialized && !editor.ShowProjectManager && !editor.SceneSetup {
			editor.SetupEditorScene()
			editor.SceneSetup = true
		}

		if editor.ImguiInitialized {
			io := imgui.CurrentIO()
			wantsKeyboard := io.WantCaptureKeyboard()

			// Check if right mouse button is down (for camera look-around)
			rightMouseDown := imgui.IsMouseDown(1)

			// Enable camera when:
			// 1. Right mouse is held (for look-around) - always enable in this case
			// 2. OR: ImGui doesn't want keyboard and mouse isn't over any panel
			windowHovered := imgui.IsWindowHoveredV(imgui.HoveredFlagsAnyWindow)

			if rightMouseDown {
				// Right mouse held = camera control takes priority
				editor.Eng.EnableCameraInput = true
			} else {
				// Normal mode: camera works when not interacting with panels
				editor.Eng.EnableCameraInput = !wantsKeyboard && !windowHovered
			}

			// Always disable camera when certain panels or modals are open
			if editor.ShowProjectManager || editor.ShowScriptBrowser || editor.ShowRebuildModal() {
				editor.Eng.EnableCameraInput = false
			}
		}

		if editor.ImguiInitialized {
			renderImGuiFrame()
		}
	})

	fmt.Println("Starting engine...")
	editor.Eng.Render(-1, -1)
}

func initializeImGui() {
	fmt.Println("Initializing ImGui on main thread...")

	window := editor.Eng.GetWindow()
	io := imgui.CurrentIO()

	var err error
	editor.Platform, err = platforms.NewGLFWFromExistingWindow(window, io)
	if err != nil {
		fmt.Printf("ERROR: Failed to create GLFW platform: %v\n", err)
		return
	}

	editor.ImguiRenderer, err = renderers.NewOpenGL3(io)
	if err != nil {
		fmt.Printf("ERROR: Failed to create OpenGL3 renderer: %v\n", err)
		return
	}

	editor.ApplyDarkTheme()
	editor.LoadConfig()
	editor.InitHotReload()

	if !editor.StyleColorsApplied {
		editor.ApplyStyleColors(editor.SavedStyleColors)
		editor.StyleColorsApplied = true
	}

	editor.ImguiInitialized = true
	fmt.Println("ImGui initialized successfully!")
}

func renderImGuiFrame() {
	if editor.Platform == nil || editor.ImguiRenderer == nil {
		return
	}

	editor.Platform.NewFrame()
	imgui.NewFrame()

	if editor.ShowProjectManager {
		editor.RenderProjectManager()
	} else {
		editor.RenderEditorUI()

		if editor.ShowGizmos {
			editor.RenderGizmos()
		}

		// Always show orientation gizmo in top-right corner
		editor.RenderOrientationGizmo()
	}

	imgui.Render()
	displaySize := editor.Platform.DisplaySize()
	framebufferSize := editor.Platform.FramebufferSize()
	editor.ImguiRenderer.Render(displaySize, framebufferSize, imgui.RenderedDrawData())
}
