package main

import (
	"Gopher3D/internal/engine"
	"Gopher3D/internal/loader"
	"Gopher3D/internal/renderer"
	"encoding/json"
	"fmt"
	mgl "github.com/go-gl/mathgl/mgl32"
	"os"
	"path/filepath"
	"runtime"
)

var (
	gameEngine *engine.Gopher
	sceneReady = false
)

func main() {
	runtime.LockOSThread()
	fmt.Println("Starting game...")

	gameEngine = engine.NewGopher(engine.OPENGL)
	gameEngine.Width = 1280
	gameEngine.Height = 720
	gameEngine.WindowDecorated = true

	gameEngine.SetOnRenderCallback(func(deltaTime float64) {
		if !sceneReady && gameEngine.Camera != nil {
			loadGame()
			sceneReady = true
		}
	})

	gameEngine.Render(-1, -1)
}

func loadGame() {
	fmt.Println("Loading game...")

	// Find and load scene
	scenePath := findAsset("scene.json")
	if scenePath == "" {
		fmt.Println("No scene.json found, starting with empty scene")
		setupDefaultScene()
		return
	}

	data, err := os.ReadFile(scenePath)
	if err != nil {
		fmt.Printf("Failed to read scene: %v\n", err)
		setupDefaultScene()
		return
	}

	var scene SceneData
	if err := json.Unmarshal(data, &scene); err != nil {
		fmt.Printf("Failed to parse scene: %v\n", err)
		setupDefaultScene()
		return
	}

	loadSceneData(&scene, filepath.Dir(scenePath))
	fmt.Println("Game loaded!")
}

func setupDefaultScene() {
	// Fallback when no scene.json is found
	// Sets up minimal defaults so the game doesn't crash
	fmt.Println("Warning: No scene file found, using minimal defaults")

	gameEngine.Camera.Position = mgl.Vec3{0, 10, 30}
	gameEngine.Camera.Speed = 50

	// Minimal ambient light so the scene isn't completely dark
	light := renderer.CreateDirectionalLight(
		mgl.Vec3{0, -1, 0}.Normalize(),
		mgl.Vec3{1.0, 1.0, 1.0},
		0.5,
	)
	light.Name = "FallbackLight"
	light.AmbientStrength = 0.5
	gameEngine.Light = light

	if r, ok := gameEngine.GetRenderer().(*renderer.OpenGLRenderer); ok {
		r.AddLight(light)
	}
}

func loadSceneData(scene *SceneData, assetsDir string) {
	r, ok := gameEngine.GetRenderer().(*renderer.OpenGLRenderer)
	if !ok {
		fmt.Println("Could not get renderer")
		return
	}

	// Load models
	for _, m := range scene.Models {
		var model *renderer.Model
		var err error

		if m.MeshDataFile != "" {
			// Load from serialized mesh data
			meshPath := filepath.Join(assetsDir, m.MeshDataFile)
			meshData, readErr := os.ReadFile(meshPath)
			if readErr != nil {
				fmt.Printf("Failed to read mesh file %s: %v\n", m.MeshDataFile, readErr)
				continue
			}

			mesh, decodeErr := renderer.DecodeMeshBinary(meshData)
			if decodeErr != nil {
				fmt.Printf("Failed to decode mesh %s: %v\n", m.MeshDataFile, decodeErr)
				continue
			}

			model = renderer.DeserializeMesh(mesh)
			fmt.Printf("Loaded mesh: %s\n", m.Name)
		} else if m.Path != "" {
			// Load from file
			modelPath := resolveAssetPath(m.Path, assetsDir)
			if modelPath == "" {
				fmt.Printf("Model file not found: %s (path: %s)\n", m.Name, m.Path)
				continue
			}

			model, err = loader.LoadModel(modelPath, true)
			if err != nil {
				fmt.Printf("Failed to load model %s: %v\n", m.Name, err)
				continue
			}
		} else {
			fmt.Printf("Skipping model %s: no path or mesh data\n", m.Name)
			continue
		}

		model.Name = m.Name
		model.SetPosition(m.Position[0], m.Position[1], m.Position[2])
		model.SetScale(m.Scale[0], m.Scale[1], m.Scale[2])

		// Apply material properties
		model.SetDiffuseColor(m.DiffuseColor[0], m.DiffuseColor[1], m.DiffuseColor[2])
		model.SetMaterialPBR(m.Metallic, m.Roughness)
		model.SetAlpha(m.Alpha)
		model.SetExposure(1.0) // Ensure proper exposure for visibility

		// Ensure material is properly initialized
		if model.Material != nil {
			if model.Material.Exposure == 0 {
				model.Material.Exposure = 1.0
			}
			if model.Material.Alpha == 0 {
				model.Material.Alpha = 1.0
			}
		}

		r.AddModel(model)
		fmt.Printf("Loaded: %s\n", m.Name)
	}

	// Load lights
	for i, l := range scene.Lights {
		var light *renderer.Light
		if l.Mode == "point" {
			light = renderer.CreatePointLight(
				mgl.Vec3{l.Position[0], l.Position[1], l.Position[2]},
				mgl.Vec3{l.Color[0], l.Color[1], l.Color[2]},
				l.Intensity,
				100.0, // Default range
			)
		} else {
			light = renderer.CreateDirectionalLight(
				mgl.Vec3{l.Direction[0], l.Direction[1], l.Direction[2]}.Normalize(),
				mgl.Vec3{l.Color[0], l.Color[1], l.Color[2]},
				l.Intensity,
			)
		}
		light.Name = l.Name
		light.AmbientStrength = l.AmbientStrength

		if i == 0 {
			gameEngine.Light = light
		}
		r.AddLight(light)
	}

	// Setup camera from scene or use defaults
	// Find active camera from multiple cameras, or use legacy single camera
	var activeCamera *SceneCamera
	for i := range scene.Cameras {
		if scene.Cameras[i].IsActive {
			activeCamera = &scene.Cameras[i]
			break
		}
	}
	if activeCamera == nil && scene.Camera != nil {
		activeCamera = scene.Camera
	}

	if activeCamera != nil {
		gameEngine.Camera.Position = mgl.Vec3{activeCamera.Position[0], activeCamera.Position[1], activeCamera.Position[2]}
		gameEngine.Camera.Yaw = activeCamera.Rotation[0]
		gameEngine.Camera.Pitch = activeCamera.Rotation[1]
		gameEngine.Camera.InvertMouse = activeCamera.InvertMouse
		if activeCamera.Speed > 0 {
			gameEngine.Camera.Speed = activeCamera.Speed
		} else {
			gameEngine.Camera.Speed = 100
		}
		if activeCamera.FOV > 0 {
			gameEngine.Camera.Fov = activeCamera.FOV
		}
		if activeCamera.Near > 0 {
			gameEngine.Camera.Near = activeCamera.Near
		}
		if activeCamera.Far > 0 {
			gameEngine.Camera.Far = activeCamera.Far
		}
		gameEngine.Camera.UpdateProjection()
	} else {
		gameEngine.Camera.Position = mgl.Vec3{0, 50, 150}
		gameEngine.Camera.Speed = 100
		gameEngine.Camera.InvertMouse = false
	}

	// If no lights in scene, add a minimal fallback light
	if len(scene.Lights) == 0 {
		fmt.Println("Warning: No lights in scene, adding fallback light")
		light := renderer.CreateDirectionalLight(
			mgl.Vec3{0, -1, -0.5}.Normalize(),
			mgl.Vec3{1.0, 1.0, 1.0},
			1.0,
		)
		light.Name = "FallbackLight"
		light.AmbientStrength = 0.3
		gameEngine.Light = light
		r.AddLight(light)
	}

	// Load skybox
	if scene.Skybox != nil {
		if scene.Skybox.Type == "color" {
			r.ClearColorR = scene.Skybox.Color[0]
			r.ClearColorG = scene.Skybox.Color[1]
			r.ClearColorB = scene.Skybox.Color[2]
		} else if scene.Skybox.ImagePath != "" {
			skyboxPath := resolveAssetPath(scene.Skybox.ImagePath, assetsDir)
			if skyboxPath != "" {
				gameEngine.SetSkybox(skyboxPath)
			}
		}
	}

	// Apply rendering configuration
	if scene.Rendering != nil {
		r.EnableBloom = scene.Rendering.Bloom
		r.EnableFXAA = scene.Rendering.FXAA
		renderer.DepthTestEnabled = scene.Rendering.DepthTest
		renderer.FaceCullingEnabled = scene.Rendering.FaceCulling
		renderer.Debug = scene.Rendering.Wireframe
	}

	// Ensure frustum culling is disabled for stability
	renderer.FrustumCullingEnabled = false

	// Load water if present
	if scene.Water != nil && scene.Water.MeshDataFile != "" {
		meshPath := filepath.Join(assetsDir, scene.Water.MeshDataFile)
		meshData, err := os.ReadFile(meshPath)
		if err != nil {
			fmt.Printf("Warning: Could not read water mesh: %v\n", err)
		} else {
			mesh, err := renderer.DecodeMeshBinary(meshData)
			if err != nil {
				fmt.Printf("Warning: Could not decode water mesh: %v\n", err)
			} else {
				waterModel := renderer.DeserializeMesh(mesh)
				waterModel.Name = "Water"
				waterModel.SetPosition(scene.Water.Position[0], scene.Water.Position[1], scene.Water.Position[2])
				waterModel.SetDiffuseColor(scene.Water.WaterColor[0], scene.Water.WaterColor[1], scene.Water.WaterColor[2])
				waterModel.SetAlpha(scene.Water.Transparency)
				r.AddModel(waterModel)
				fmt.Println("Water loaded (static mesh - no animation)")
			}
		}
	}
}

func findAsset(name string) string {
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	paths := []string{
		filepath.Join(exeDir, "assets", name),
		filepath.Join(exeDir, name),
		filepath.Join("assets", name),
		name,
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func resolveAssetPath(path, assetsDir string) string {
	if path == "" {
		return ""
	}

	// Try direct path first
	if _, err := os.Stat(path); err == nil {
		return path
	}

	// Try in assets directory
	assetPath := filepath.Join(assetsDir, filepath.Base(path))
	if _, err := os.Stat(assetPath); err == nil {
		return assetPath
	}

	// Try relative to executable
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	exeAssetPath := filepath.Join(exeDir, "assets", filepath.Base(path))
	if _, err := os.Stat(exeAssetPath); err == nil {
		return exeAssetPath
	}

	return ""
}

// Scene data structures (must match editor format)
type SceneData struct {
	GameObjects []SceneGameObject     `json:"game_objects,omitempty"`
	Models      []SceneModel          `json:"models,omitempty"`
	Lights      []SceneLight          `json:"lights,omitempty"`
	Camera      *SceneCamera          `json:"camera,omitempty"`
	Cameras     []SceneCamera         `json:"cameras,omitempty"`
	Water       *SceneWater           `json:"water,omitempty"`
	Skybox      *SceneSkybox          `json:"skybox,omitempty"`
	Rendering   *SceneRenderingConfig `json:"rendering,omitempty"`
}

type SceneGameObject struct {
	Name       string           `json:"name"`
	Tag        string           `json:"tag,omitempty"`
	Active     bool             `json:"active"`
	Position   [3]float32       `json:"position"`
	Rotation   [3]float32       `json:"rotation"`
	Scale      [3]float32       `json:"scale"`
	Components []SceneComponent `json:"components,omitempty"`
}

type SceneCamera struct {
	Name        string     `json:"name"`
	Position    [3]float32 `json:"position"`
	Rotation    [3]float32 `json:"rotation"`
	Speed       float32    `json:"speed"`
	FOV         float32    `json:"fov"`
	Near        float32    `json:"near,omitempty"`
	Far         float32    `json:"far,omitempty"`
	InvertMouse bool       `json:"invert_mouse"`
	IsActive    bool       `json:"is_active"`
}

type SceneModel struct {
	Name          string           `json:"name"`
	Path          string           `json:"path,omitempty"`
	MeshDataFile  string           `json:"mesh_data_file,omitempty"`
	Position      [3]float32       `json:"position"`
	Scale         [3]float32       `json:"scale"`
	Rotation      [3]float32       `json:"rotation"`
	DiffuseColor  [3]float32       `json:"diffuse_color"`
	SpecularColor [3]float32       `json:"specular_color"`
	Shininess     float32          `json:"shininess"`
	Metallic      float32          `json:"metallic"`
	Roughness     float32          `json:"roughness"`
	Alpha         float32          `json:"alpha"`
	Components    []SceneComponent `json:"components,omitempty"`
}

type SceneComponent struct {
	Type       string                 `json:"type"`
	Category   string                 `json:"category"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

type SceneLight struct {
	Name            string     `json:"name"`
	Mode            string     `json:"mode"`
	Position        [3]float32 `json:"position"`
	Direction       [3]float32 `json:"direction"`
	Color           [3]float32 `json:"color"`
	Intensity       float32    `json:"intensity"`
	AmbientStrength float32    `json:"ambient_strength"`
}

type SceneWater struct {
	OceanSize           float32    `json:"ocean_size"`
	BaseAmplitude       float32    `json:"base_amplitude"`
	WaterColor          [3]float32 `json:"water_color"`
	Transparency        float32    `json:"transparency"`
	WaveSpeedMultiplier float32    `json:"wave_speed_multiplier"`
	Position            [3]float32 `json:"position"`
	FoamEnabled         bool       `json:"foam_enabled"`
	FoamIntensity       float32    `json:"foam_intensity"`
	CausticsEnabled     bool       `json:"caustics_enabled"`
	CausticsIntensity   float32    `json:"caustics_intensity"`
	CausticsScale       float32    `json:"caustics_scale"`
	SpecularIntensity   float32    `json:"specular_intensity"`
	NormalStrength      float32    `json:"normal_strength"`
	DistortionStrength  float32    `json:"distortion_strength"`
	ShadowStrength      float32    `json:"shadow_strength"`
	MeshDataFile        string     `json:"mesh_data_file,omitempty"`
}

type SceneSkybox struct {
	Type      string     `json:"type"`
	ImagePath string     `json:"image_path"`
	Color     [3]float32 `json:"color"`
}

type SceneRenderingConfig struct {
	Bloom       bool       `json:"bloom"`
	FXAA        bool       `json:"fxaa"`
	DepthTest   bool       `json:"depth_test"`
	FaceCulling bool       `json:"face_culling"`
	Wireframe   bool       `json:"wireframe"`
	SkyboxColor [3]float32 `json:"skybox_color"`
}
