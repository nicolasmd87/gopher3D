package renderer

import (
	"Gopher3D/internal/logger"
	"fmt"
	"image"
	"runtime"
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"
	as "github.com/vulkan-go/asche"
	vk "github.com/vulkan-go/vulkan"
	"go.uber.org/zap"
)

type VulkanRenderer struct {
	VulkanApp             *Application
	platform              as.Platform
	FrustumCullingEnabled bool
	FaceCullingEnabled    bool
	Debug                 bool
	Shader                Shader
	Models                []*Model
	textures              map[string]*Texture
}
type Application struct {
	*Scene
	debugEnabled bool
	window       *glfw.Window
}

func NewVulkanApp(window *glfw.Window) *Application {
	return &Application{window: window, debugEnabled: false, Scene: NewScene(1.0)}
}

func (app *Application) VulkanSurface(instance vk.Instance) (surface vk.Surface) {
	logger.Log.Info("Creating Vulkan Surface")
	surfUint64, err := app.window.CreateWindowSurface(instance, nil)
	if err != nil {
		logger.Log.Error("Failed to create window surface", zap.Error(err))
		return vk.NullSurface
	}
	surface = vk.SurfaceFromPointer(surfUint64)
	if err != nil {
		logger.Log.Error("Failed to create window surface", zap.Error(err))
		return vk.NullSurface
	}
	logger.Log.Info("Vulkan surface created", zap.Any("Surface", surface))
	return surface
}

func (a *Application) VulkanLayers() []string {
	return []string{
		/*
			"VK_LAYER_GOOGLE_threading",
			"VK_LAYER_LUNARG_parameter_validation",
			"VK_LAYER_LUNARG_object_tracker",
			"VK_LAYER_LUNARG_core_validation",
			"VK_LAYER_LUNARG_api_dump",
			"VK_LAYER_LUNARG_swapchain",
			"VK_LAYER_GOOGLE_unique_objects",
		*/
	}
}

func (app *Application) VulkanDebug() bool {
	return false
}

func (app *Application) VulkanAppName() string {
	return "Gopher3D"
}

func (app *Application) VulkanInstanceExtensions() []string {
	extensions := app.window.GetRequiredInstanceExtensions()
	if app.debugEnabled {
		extensions = append(extensions, "VK_EXT_debug_report")
	}
	return extensions
}

func (app *Application) VulkanSwapchainDimensions() *as.SwapchainDimensions {
	return &as.SwapchainDimensions{
		Width: uint32(500), Height: uint32(500), Format: vk.FormatB8g8r8a8Unorm,
	}
}

func (app *Application) VulkanMode() as.VulkanMode {
	return as.DefaultVulkanMode
}

func (app *Application) VulkanDeviceExtensions() []string {
	return []string{
		"VK_KHR_swapchain",
	}
}

func (rend *VulkanRenderer) Init(width, height int32, window *glfw.Window) {
	runtime.LockOSThread()
	vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	rend.VulkanApp = NewVulkanApp(window)
	rend.VulkanApp.window = window
	rend.textures = make(map[string]*Texture) // Initialize the texture map

	logger.Log.Info("Initializing Vulkan Renderer")

	// Initialize Vulkan
	if err := vk.Init(); err != nil {
		logger.Log.Error("Failed to initialize Vulkan", zap.Error(err))
		return
	}

	logger.Log.Info("Creating Asche platform")

	platform, err := as.NewPlatform(rend.VulkanApp)

	if err != nil {
		logger.Log.Error("Failed to create Asche platform", zap.Error(err))
		return
	}

	logger.Log.Info("Asche platform created", zap.Any("Platform", platform))

	dim := rend.VulkanApp.Context().SwapchainDimensions()
	logger.Log.Info("Swapchain Initialized", zap.Any("Swapchain dimensions:", dim))

	rend.platform = platform
	rend.VulkanApp.Scene.Debug = Debug
	logger.Log.Info("Vulkan framework initialized.")
}

func (rend *VulkanRenderer) Render(camera Camera, light *Light) {
	if rend.VulkanApp.Context() == nil {
		logger.Log.Error("Vulkan context is nil")
		return
	}
	updateCamera(rend, camera)
	imageIdx, outdated, err := rend.VulkanApp.Context().AcquireNextImage()
	if err != nil {
		logger.Log.Error("Failed to acquire next image", zap.Error(err))
		return
	}
	if outdated {
		logger.Log.Info("Swapchain outdated")
		imageIdx, outdated, err = rend.VulkanApp.Context().AcquireNextImage()
		if err != nil {
			logger.Log.Error("Failed to acquire next image", zap.Error(err))
		}
	}
	//logger.Log.Info("Rendering frame", zap.Int("Image index", imageIdx))
	rend.VulkanApp.Context().PresentImage(imageIdx)
	rend.VulkanApp.NextFrame()
}

func (rend *VulkanRenderer) AddModel(model *Model) {
	logger.Log.Info("Adding model to renderer")
	// Prepare vertex and index buffers
	vertexBuffer, vertexMemory, err := rend.VulkanApp.Scene.prepareVertexBuffer(model.Vertices)
	if err != nil {
		logger.Log.Error("Failed to prepare vertex buffer", zap.Error(err))
		return
	}
	indexBuffer, indexMemory, err := rend.VulkanApp.Scene.prepareIndexBuffer(model.Indices)
	if err != nil {
		logger.Log.Error("Failed to prepare index buffer", zap.Error(err))
		return
	}
	model.vertexBuffer = vertexBuffer
	model.vertexMemory = vertexMemory
	model.indexBuffer = indexBuffer
	model.indexMemory = indexMemory

	// Add model to the scene's model list
	rend.VulkanApp.Scene.models = append(rend.VulkanApp.Scene.models, model)
}
func (rend *VulkanRenderer) RemoveModel(model *Model) {

}

func (rend *VulkanRenderer) LoadTexture(path string) (uint32, error) {
	if texture, exists := rend.textures[path]; exists {
		logger.Log.Info("Texture already loaded", zap.String("path", path))
		return uint32(uintptr(unsafe.Pointer(texture.image))), nil // Return the existing texture ID
	}
	// Load texture using the Scene's prepareTextureImage method
	logger.Log.Info("Loading texture", zap.String("path", path))
	texture := rend.VulkanApp.Scene.prepareTextureImage(path, vk.ImageTilingOptimal, vk.ImageUsageTransferDstBit|vk.ImageUsageSampledBit, vk.MemoryPropertyDeviceLocalBit)
	if texture == nil {
		return 0, fmt.Errorf("failed to load texture from path: %s", path)
	}
	// Store the texture in the map
	rend.textures[path] = texture
	// Return the texture's image handle as the texture ID
	return uint32(uintptr(unsafe.Pointer(texture.image))), nil
}

func (rend *VulkanRenderer) CreateTextureFromImage(img image.Image) (uint32, error) {
	return 0, nil
}

func (rend *VulkanRenderer) SetSkybox(skybox *Skybox) {
	// TODO: Implement Vulkan skybox support
}

func (rend *VulkanRenderer) Cleanup() {

}

func (rend *VulkanRenderer) UpdateViewport(width, height int32) {
	// Vulkan viewport is handled differently, but we need to implement the interface
}

func (rend *VulkanRenderer) SetDebugMode(debug bool) {
	Debug = debug
}

func (rend *VulkanRenderer) SetFrustumCulling(enabled bool) {
	rend.FrustumCullingEnabled = enabled
}

func (rend *VulkanRenderer) SetFaceCulling(enabled bool) {
	FaceCullingEnabled = enabled
}

// TODO: Mouse movement is not working for some reason, check vectors up, down , etc
func updateCamera(rend *VulkanRenderer, camera Camera) {
	rend.VulkanApp.projectionMatrix = camera.GetViewProjectionVulkan()
	rend.VulkanApp.projectionMatrix[1][1] *= -1 // Flip Y-axis for Vulkan
	rend.VulkanApp.viewMatrix = camera.GetViewMatrixVulkan()
}
