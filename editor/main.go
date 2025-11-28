package main

import (
	"Gopher3D/editor/platforms"
	"Gopher3D/editor/renderers"
	"Gopher3D/internal/engine"
	_ "Gopher3D/scripts"
	"fmt"
	"runtime"

	"github.com/inkyblackness/imgui-go/v4"
)

func main() {
	runtime.LockOSThread()

	fmt.Println("===========================================")
	fmt.Println("   Gopher3D Editor with ImGui")
	fmt.Println("===========================================")

	// Create ImGui context
	context := imgui.CreateContext(nil)
	defer context.Destroy()
	defer saveConfig() // Save editor settings on exit

	eng = engine.NewGopher(engine.OPENGL)
	eng.Width = 1280
	eng.Height = 720
	eng.WindowDecorated = true

	// Set render callback to handle ImGui initialization and rendering on main thread
	eng.SetOnRenderCallback(func(deltaTime float64) {
		// Initialize ImGui on first render (when window exists and we're on main thread)
		if !imguiInitialized && eng.GetWindow() != nil {
			initializeImGui()
		}

		// Start project manager if not yet showing a project
		if imguiInitialized && !sceneSetup && !showProjectManager && currentProject == nil {
			// First run - show project manager
			showProjectManager = true
		}

		// Setup scene once ImGui is ready AND project is loaded
		if imguiInitialized && !showProjectManager && !sceneSetup {
			setupEditorScene()
			sceneSetup = true
		}

		// Control camera input based on ImGui state
		if imguiInitialized {
			io := imgui.CurrentIO()
			// Disable camera input when ImGui wants keyboard or mouse
			wantsKeyboard := io.WantCaptureKeyboard()
			wantsMouse := io.WantCaptureMouse()

			// Additional check: disable camera if any text input is active
			anyItemActive := imgui.IsAnyItemActive()

			// Disable camera if any input field is active or any UI element wants input
			eng.EnableCameraInput = !wantsKeyboard && !wantsMouse && !anyItemActive

			// Also disable camera if project manager is open
			if showProjectManager {
				eng.EnableCameraInput = false
			}
		}

		// Render ImGui UI
		if imguiInitialized {
			renderImGuiFrame()
		}
	})

	fmt.Println("Starting engine...")
	// Start engine with centered window (-1, -1 means center)
	eng.Render(-1, -1)
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

	applyDarkTheme()
	loadConfig()

	// Apply saved style colors if available
	if !styleColorsApplied {
		applyStyleColors(savedStyleColors)
		styleColorsApplied = true
	}

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

	// Render Project Manager if active
	if showProjectManager {
		renderProjectManager()
	} else {
		// Render Editor UI
		renderEditorUI()

		// Render Gizmos on top
		if showGizmos {
			renderGizmos()
		}
	}

	// Render
	imgui.Render()
	displaySize := platform.DisplaySize()
	framebufferSize := platform.FramebufferSize()
	imguiRenderer.Render(displaySize, framebufferSize, imgui.RenderedDrawData())
}
