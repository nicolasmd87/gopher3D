package engine

import (
	behaviour "Gopher3D/internal/behaviour"
	"Gopher3D/internal/logger"
	"Gopher3D/internal/renderer"
	"runtime"
	"time"

	mgl "github.com/go-gl/mathgl/mgl32"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"go.uber.org/zap"
)

var COLOR_ACTIVECAPTION int32 = 2

// Initialize to the center of the window
var lastX, lastY float64
var firstMouse bool = true
var camera renderer.Camera
var refreshRate time.Duration = 1000 / 144 // 144 FPS

// Enum for rendererAPIs Vulkan and OpenGL
type rendAPI int

const (
	OPENGL rendAPI = iota
	VULKAN
)

// TODO: Separate window into an abstract class with width and height as fields
type Gopher struct {
	Width            int32
	Height           int32
	ModelChan        chan *renderer.Model
	ModelBatchChan   chan []*renderer.Model
	Light            *renderer.Light
	rendererAPI      renderer.Render
	window           *glfw.Window
	skybox           *renderer.Skybox
	skyboxPath       string // Store path until OpenGL is ready
	Camera           *renderer.Camera
	frameTrackId     int
	onRenderCallback func(deltaTime float64) // Optional callback for custom rendering (e.g., editor UI)
	EnableCameraInput bool // Control whether camera processes keyboard/mouse input (for editor)
}

func NewGopher(rendererAPI rendAPI) *Gopher {
	logger.Init()
	logger.Log.Info("Gopher3D initializing...")
	//Default renderer is OpenGL until we get Vulkan working
	var rendAPI renderer.Render
	if rendererAPI == OPENGL {
		rendAPI = &renderer.OpenGLRenderer{}
	} else {
		rendAPI = &renderer.VulkanRenderer{}
	}
	return &Gopher{
		//TODO: We need to be able to set width and height of the window
		rendererAPI:       rendAPI,
		Width:             1024,
		Height:            768,
		ModelChan:         make(chan *renderer.Model, 1000000),
		ModelBatchChan:    make(chan []*renderer.Model, 1000000),
		frameTrackId:      0,
		EnableCameraInput: true, // Enabled by default
	}
}

// Gopher API
func (gopher *Gopher) Render(x, y int) {
	lastX, lastY = float64(gopher.Width/2), float64(gopher.Width/2)
	runtime.LockOSThread()

	if err := glfw.Init(); err != nil {
		logger.Log.Error("Could not initialize glfw: %v", zap.Error(err))
	}
	defer glfw.Terminate()

	// Set GLFW window hints here
	glfw.WindowHint(glfw.Decorated, glfw.True)
	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.DepthBits, 32) // Request 32-bit depth buffer for better precision

	var err error

	switch gopher.rendererAPI.(type) {
	case *renderer.VulkanRenderer:
		// Set GLFW to not create an OpenGL context
		glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	case *renderer.OpenGLRenderer:
		glfw.WindowHint(glfw.ContextVersionMajor, 4)
		glfw.WindowHint(glfw.ContextVersionMinor, 1)
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
		glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	default:
		logger.Log.Error("Unknown renderer type", zap.String("fun", "Render"))
		return
	}

	gopher.window, err = glfw.CreateWindow(int(gopher.Width), int(gopher.Height), "Gopher3D", nil, nil)

	if err != nil {
		logger.Log.Error("Could not create glfw window: %v", zap.Error(err))
	}

	if _, ok := gopher.rendererAPI.(*renderer.OpenGLRenderer); ok {
		gopher.window.MakeContextCurrent()
		if err := gl.Init(); err != nil {
			logger.Log.Error("Could not initialize OpenGL: %v", zap.Error(err))
			return
		}
		gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	}

	gopher.window.SetPos(x, y)

	gopher.rendererAPI.Init(gopher.Width, gopher.Height, gopher.window)

	// Fixed camera in each scene for now
	gopher.Camera = renderer.NewDefaultCamera(gopher.Width, gopher.Height)

	// TODO: This should be set in the window class
	//window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled) // Hide and capture the cursor
	gopher.window.SetInputMode(glfw.CursorMode, glfw.CursorNormal) // Set cursor to normal mode initially

	gopher.window.SetCursorPosCallback(gopher.mouseCallback) // Set the callback function for mouse movement

	gopher.RenderLoop()
}

func (gopher *Gopher) RenderLoop() {
	var lastTime = glfw.GetTime()
	var lastWidth, lastHeight int32 = gopher.Width, gopher.Height

	for !gopher.window.ShouldClose() {
		currentTime := glfw.GetTime()
		deltaTime := currentTime - lastTime
		lastTime = currentTime

		// Check actual window size and update if it changed
		actualWidth, actualHeight := gopher.window.GetSize()
		if int32(actualWidth) != gopher.Width || int32(actualHeight) != gopher.Height {
			gopher.Width = int32(actualWidth)
			gopher.Height = int32(actualHeight)
		}

		// Update viewport and camera aspect ratio if window size changed (after OpenGL is initialized)
		if gopher.Width != lastWidth || gopher.Height != lastHeight {
			gopher.rendererAPI.UpdateViewport(gopher.Width, gopher.Height)
			lastWidth, lastHeight = gopher.Width, gopher.Height
		}

		// Only process camera input if enabled (can be disabled by editor when UI wants keyboard)
		if gopher.EnableCameraInput {
			gopher.Camera.ProcessKeyboard(gopher.window, float32(deltaTime))
		}

		//TODO: Rignt now it's fixed but maybe in the future we can make it confgigurable?
		if gopher.frameTrackId >= 2 {
			behaviour.GlobalBehaviourManager.UpdateAllFixed()
			gopher.frameTrackId = 0
		}
		behaviour.GlobalBehaviourManager.UpdateAll()

		// Check if a skybox needs to be created (can happen dynamically from behaviors)
		if gopher.skyboxPath != "" && gopher.skybox == nil {
			skybox, err := renderer.CreateSkybox(gopher.skyboxPath)
			if err != nil {
				logger.Log.Error("Failed to create skybox", zap.String("path", gopher.skyboxPath), zap.Error(err))
			} else {
				gopher.skybox = skybox
				gopher.rendererAPI.SetSkybox(skybox)
				logger.Log.Info("Skybox created and set", zap.String("path", gopher.skyboxPath))
			}
		}

		gopher.rendererAPI.Render(*gopher.Camera, gopher.Light)

		// Call custom render callback if set (for editor UI, etc.)
		if gopher.onRenderCallback != nil {
			gopher.onRenderCallback(deltaTime)
		}

		switch gopher.rendererAPI.(type) {
		case *renderer.OpenGLRenderer:
			gopher.window.SwapBuffers()
		}
		gopher.frameTrackId++
		glfw.PollEvents()
	}
	gopher.rendererAPI.Cleanup()
}

// SetOnRenderCallback sets a callback that will be called each frame after the 3D scene is rendered
func (gopher *Gopher) SetOnRenderCallback(callback func(deltaTime float64)) {
	gopher.onRenderCallback = callback
}

// TODO: Get rid of this, and use a renderer global variable
func (gopher *Gopher) SetDebugMode(debug bool) {
	renderer.Debug = debug
}

func (gopher *Gopher) SetFrustumCulling(enabled bool) {
	renderer.FrustumCullingEnabled = enabled
}

func (gopher *Gopher) SetFaceCulling(enabled bool) {
	renderer.FaceCullingEnabled = enabled
}

// SetSkybox sets a skybox for the engine
func (g *Gopher) SetSkybox(texturePath string) error {
	// Store the path - skybox will be created after OpenGL initialization
	g.skyboxPath = texturePath
	return nil
}

// UpdateSkyboxColor updates the color of the existing skybox (for solid color skyboxes only)
func (gopher *Gopher) UpdateSkyboxColor(r, g, b float32) {
	if gopher.skybox != nil {
		gopher.skybox.UpdateColor(r, g, b)
	}
}

func (gopher *Gopher) AddModel(model *renderer.Model) {
	gopher.rendererAPI.AddModel(model)
}

func (gopher *Gopher) RemoveModel(model *renderer.Model) {
	gopher.rendererAPI.RemoveModel(model)
}

func (g *Gopher) GetMousePosition() mgl.Vec2 {
	x, y := g.window.GetCursorPos()
	return mgl.Vec2{float32(x), float32(y)}
}

func (g *Gopher) IsMouseButtonPressed(button glfw.MouseButton) bool {
	return g.window.GetMouseButton(button) == glfw.Press
}

func (gopher *Gopher) AddModelBatch(models []*renderer.Model) {
	for _, model := range models {
		gopher.rendererAPI.AddModel(model)
	}
}

// GetDefaultShader returns the default shader for models that need it
func (gopher *Gopher) GetDefaultShader() renderer.Shader {
	if openglRenderer, ok := gopher.rendererAPI.(*renderer.OpenGLRenderer); ok {
		return openglRenderer.GetDefaultShader()
	}
	// Return empty shader for other renderers (they'll handle it internally)
	return renderer.Shader{}
}

// GetWindow returns the GLFW window (for editor/advanced use)
func (gopher *Gopher) GetWindow() *glfw.Window {
	return gopher.window
}

// GetRenderer returns the renderer API (for editor/advanced use)
func (gopher *Gopher) GetRenderer() renderer.Render {
	return gopher.rendererAPI
}

// Mouse callback function
func (gopher *Gopher) mouseCallback(w *glfw.Window, xpos, ypos float64) {
	// Check if the window is focused and the right mouse button is pressed
	// Only process if camera input is enabled (can be disabled by editor when UI wants mouse)
	if gopher.EnableCameraInput && w.GetAttrib(glfw.Focused) == glfw.True && w.GetMouseButton(glfw.MouseButtonRight) == glfw.Press {
		if firstMouse {
			lastX = xpos
			lastY = ypos
			firstMouse = false
			return
		}

		xoffset := xpos - lastX
		yoffset := lastY - ypos // Reversed since y-coordinates go from bottom to top
		lastX = xpos
		lastY = ypos

		gopher.Camera.ProcessMouseMovement(float32(xoffset), float32(yoffset), true)
	} else {
		firstMouse = true
	}

}
