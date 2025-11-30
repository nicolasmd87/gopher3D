package main

import (
	"Gopher3D/editor/internal"
	"Gopher3D/editor/platforms"
	"Gopher3D/editor/renderers"
	"Gopher3D/internal/engine"
	"fmt"
	"runtime"

	"github.com/inkyblackness/imgui-go/v4"
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
			wantsMouse := io.WantCaptureMouse()

			// Only disable camera input when ImGui explicitly wants keyboard/mouse
			// Don't use IsAnyItemActive as it can get stuck
			editor.Eng.EnableCameraInput = !wantsKeyboard && !wantsMouse

			// Always disable camera when certain panels are open
			if editor.ShowProjectManager || editor.ShowScriptBrowser {
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
	}

	imgui.Render()
	displaySize := editor.Platform.DisplaySize()
	framebufferSize := editor.Platform.FramebufferSize()
	editor.ImguiRenderer.Render(displaySize, framebufferSize, imgui.RenderedDrawData())
}
