package editor

import (
	"Gopher3D/internal/behaviour"
	"Gopher3D/internal/loader"
	"Gopher3D/internal/renderer"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"path/filepath"
	"strings"

	mgl "github.com/go-gl/mathgl/mgl32"
	"github.com/sqweek/dialog"
)

type SceneData struct {
	// New unified GameObject system
	GameObjects []SceneGameObject `json:"game_objects,omitempty"`

	// Legacy data (for backward compatibility)
	Models    []SceneModel          `json:"models,omitempty"`
	Lights    []SceneLight          `json:"lights,omitempty"`
	Camera    *SceneCamera          `json:"camera,omitempty"`
	Cameras   []SceneCamera         `json:"cameras,omitempty"`
	Water     *SceneWater           `json:"water,omitempty"`
	Skybox    *SceneSkybox          `json:"skybox,omitempty"`
	Rendering *SceneRenderingConfig `json:"rendering,omitempty"`
}

type SceneRenderingConfig struct {
	Bloom       bool       `json:"bloom"`
	FXAA        bool       `json:"fxaa"`
	DepthTest   bool       `json:"depth_test"`
	FaceCulling bool       `json:"face_culling"`
	Wireframe   bool       `json:"wireframe"`
	SkyboxColor [3]float32 `json:"skybox_color"`
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
	Name     string     `json:"name"`
	Path     string     `json:"path,omitempty"`
	Position [3]float32 `json:"position"`
	Scale    [3]float32 `json:"scale"`
	Rotation [3]float32 `json:"rotation"`

	// Complete Material Properties
	DiffuseColor  [3]float32 `json:"diffuse_color"`
	SpecularColor [3]float32 `json:"specular_color"`
	Shininess     float32    `json:"shininess"`
	Metallic      float32    `json:"metallic"`
	Roughness     float32    `json:"roughness"`
	Exposure      float32    `json:"exposure"`
	Alpha         float32    `json:"alpha"`
	TexturePath   string     `json:"texture_path,omitempty"`

	// Serialized mesh data (for procedural/voxel models)
	MeshDataFile string `json:"mesh_data_file,omitempty"`

	// Voxel Specific Data (for regeneration if needed)
	VoxelConfig *VoxelConfig `json:"voxel_config,omitempty"`

	// Components
	Components []SceneComponent `json:"components,omitempty"`
}

type SceneComponent struct {
	Type       string                 `json:"type"`
	Category   string                 `json:"category"` // "Script", "Mesh", "Water", "Voxel", etc.
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// SceneGameObject represents a GameObject in the scene (unified representation)
type SceneGameObject struct {
	Name       string           `json:"name"`
	Tag        string           `json:"tag,omitempty"`
	Active     bool             `json:"active"`
	Position   [3]float32       `json:"position"`
	Rotation   [3]float32       `json:"rotation"` // Euler angles
	Scale      [3]float32       `json:"scale"`
	Components []SceneComponent `json:"components,omitempty"`
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
	Type      string     `json:"type"`       // "image" or "color"
	ImagePath string     `json:"image_path"` // Path to skybox texture (for image type)
	Color     [3]float32 `json:"color"`      // RGB color (for color type)
}

func newScene() {
	if sceneModified {
		// TODO: Add confirmation dialog
		logToConsole("Creating new scene (unsaved changes will be lost)", "warning")
	}

	openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
	if !ok {
		logToConsole("ERROR: Cannot access renderer", "error")
		return
	}

	// Clear all GameObjects first (this also cleans up their models)
	behaviour.GlobalComponentManager.Clear()
	modelToGameObject = make(map[*renderer.Model]*behaviour.GameObject)

	// Clear all models
	models := openglRenderer.GetModels()
	for i := len(models) - 1; i >= 0; i-- {
		openglRenderer.RemoveModel(models[i])
	}

	// Clear all lights except default
	lights := openglRenderer.GetLights()
	var lightsToRemove []*renderer.Light
	for _, light := range lights {
		if light.Name != "DirectionalLight" {
			lightsToRemove = append(lightsToRemove, light)
		}
	}
	for _, light := range lightsToRemove {
		openglRenderer.RemoveLight(light)
	}

	// RE-FETCH lights to ensure we have the up-to-date list after removals
	lights = openglRenderer.GetLights()

	if len(lights) == 0 {
		defaultLight := renderer.CreateDirectionalLight(
			mgl.Vec3{-0.3, 0.8, -0.5}.Normalize(),
			mgl.Vec3{1.0, 0.95, 0.85},
			1.0,
		)
		defaultLight.Name = "DirectionalLight"
		defaultLight.AmbientStrength = 0.3
		defaultLight.Type = renderer.STATIC_LIGHT
		openglRenderer.AddLight(defaultLight)
		Eng.Light = defaultLight
	} else {
		// Ensure Eng.Light points to the first light
		Eng.Light = lights[0]
	}

	// Reset Water
	if activeWaterSim != nil {
		behaviour.GlobalBehaviourManager.Remove(activeWaterSim)
	}
	activeWaterSim = nil

	// Reset Cameras
	SceneCameras = nil

	currentScenePath = ""
	sceneModified = false
	selectedModelIndex = -1
	selectedLightIndex = -1
	selectedGameObjectIndex = -1
	selectedCameraIndex = -1
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

	openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
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

		// Check for voxel configuration
		if model.Metadata != nil {
			if config, ok := model.Metadata["voxelConfig"].(VoxelConfig); ok {
				sceneModel.VoxelConfig = &config
			}
		}

		// Save components
		if obj := getGameObjectForModel(model); obj != nil {
			sceneModel.Components = serializeComponents(obj.Components)
		}

		sceneData.Models = append(sceneData.Models, sceneModel)
	}

	// Save all GameObjects (new unified system)
	allGameObjects := behaviour.GlobalComponentManager.GetAllGameObjects()
	for _, obj := range allGameObjects {
		// Skip GameObjects that are already saved as models
		if obj.GetModel() != nil {
			continue
		}

		sceneGO := SceneGameObject{
			Name:       obj.Name,
			Tag:        obj.Tag,
			Active:     obj.Active,
			Position:   [3]float32{obj.Transform.Position.X(), obj.Transform.Position.Y(), obj.Transform.Position.Z()},
			Rotation:   quatToEulerArray(obj.Transform.Rotation),
			Scale:      [3]float32{obj.Transform.Scale.X(), obj.Transform.Scale.Y(), obj.Transform.Scale.Z()},
			Components: serializeComponents(obj.Components),
		}
		sceneData.GameObjects = append(sceneData.GameObjects, sceneGO)
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

	// Save water if it exists
	if activeWaterSim != nil && activeWaterSim.model != nil {
		sceneData.Water = &SceneWater{
			OceanSize:           activeWaterSim.oceanSize,
			BaseAmplitude:       activeWaterSim.baseAmplitude,
			WaterColor:          [3]float32{activeWaterSim.WaterColor.X(), activeWaterSim.WaterColor.Y(), activeWaterSim.WaterColor.Z()},
			Transparency:        activeWaterSim.Transparency,
			WaveSpeedMultiplier: activeWaterSim.WaveSpeedMultiplier,
			Position:            [3]float32{activeWaterSim.model.Position.X(), activeWaterSim.model.Position.Y(), activeWaterSim.model.Position.Z()},
			FoamEnabled:         activeWaterSim.FoamEnabled,
			FoamIntensity:       activeWaterSim.FoamIntensity,
			CausticsEnabled:     activeWaterSim.CausticsEnabled,
			CausticsIntensity:   activeWaterSim.CausticsIntensity,
			CausticsScale:       activeWaterSim.CausticsScale,
			SpecularIntensity:   activeWaterSim.SpecularIntensity,
			NormalStrength:      activeWaterSim.NormalStrength,
			DistortionStrength:  activeWaterSim.DistortionStrength,
			ShadowStrength:      activeWaterSim.ShadowStrength,
		}
	}

	// Save camera settings (both legacy single camera and multiple cameras)
	if Eng.Camera != nil {
		// Save legacy camera for backward compatibility
		sceneData.Camera = &SceneCamera{
			Name:        "Main Camera",
			Position:    [3]float32{Eng.Camera.Position.X(), Eng.Camera.Position.Y(), Eng.Camera.Position.Z()},
			Rotation:    [3]float32{Eng.Camera.Yaw, Eng.Camera.Pitch, 0},
			Speed:       Eng.Camera.Speed,
			FOV:         Eng.Camera.Fov,
			Near:        Eng.Camera.Near,
			Far:         Eng.Camera.Far,
			InvertMouse: false, // Default to non-inverted for exported games
			IsActive:    true,
		}
	}

	// Save all scene cameras
	if len(SceneCameras) > 0 {
		sceneData.Cameras = make([]SceneCamera, len(SceneCameras))
		for i, cam := range SceneCameras {
			sceneData.Cameras[i] = SceneCamera{
				Name:        cam.Name,
				Position:    [3]float32{cam.Position.X(), cam.Position.Y(), cam.Position.Z()},
				Rotation:    [3]float32{cam.Yaw, cam.Pitch, 0},
				Speed:       cam.Speed,
				FOV:         cam.Fov,
				Near:        cam.Near,
				Far:         cam.Far,
				InvertMouse: cam.InvertMouse,
				IsActive:    cam.IsActive,
			}
		}
	}

	// Save skybox if it exists
	if skyboxColorMode {
		sceneData.Skybox = &SceneSkybox{
			Type:  "color",
			Color: skyboxSolidColor,
		}
	} else if skyboxTexturePath != "" {
		sceneData.Skybox = &SceneSkybox{
			Type:      "image",
			ImagePath: skyboxTexturePath,
		}
	}

	// Save rendering configuration
	sceneData.Rendering = &SceneRenderingConfig{
		Bloom:       openglRenderer.EnableBloom,
		FXAA:        openglRenderer.EnableFXAA,
		DepthTest:   renderer.DepthTestEnabled,
		FaceCulling: renderer.FaceCullingEnabled,
		Wireframe:   renderer.Debug,
		SkyboxColor: skyboxSolidColor,
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

	openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
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
	Eng.Light = nil

	// Clear selection before loading models to ensure indices stay valid
	selectedModelIndex = -1
	selectedLightIndex = -1
	selectedType = ""

	// Load models
	for _, sceneModel := range sceneData.Models {
		var model *renderer.Model

		// Case 1: Voxel Terrain
		if sceneModel.VoxelConfig != nil {
			logToConsole(fmt.Sprintf("Regenerating voxel terrain: %s", sceneModel.Name), "info")
			model = regenerateVoxelTerrain(*sceneModel.VoxelConfig)
			if model == nil {
				logToConsole(fmt.Sprintf("Failed to regenerate voxel terrain: %s", sceneModel.Name), "error")
				continue
			}

			// Restore texture path for voxels BEFORE adding model
			if sceneModel.TexturePath != "" {
				if model.Material == nil {
					model.Material = &renderer.Material{}
				}
				model.Material.TexturePath = sceneModel.TexturePath
				model.Material.TextureID = 0
				logToConsole(fmt.Sprintf("Voxel texture will be loaded: %s", filepath.Base(sceneModel.TexturePath)), "info")
			}

			// Manually add model here since regenerateVoxelTerrain no longer does it
			Eng.AddModel(model)
		} else {
			// Case 2: Standard Model
			if sceneModel.Path == "" {
				logToConsole(fmt.Sprintf("Skipping model '%s' (no path stored)", sceneModel.Name), "warning")
				continue
			}

			logToConsole(fmt.Sprintf("Loading model: %s from %s", sceneModel.Name, filepath.Base(sceneModel.Path)), "info")

			var err error
			model, err = loader.LoadObjectWithPath(sceneModel.Path, true)
			if err != nil {
				logToConsole(fmt.Sprintf("Failed to load model: %v", err), "error")
				continue
			}
			Eng.AddModel(model)
		}

		// Common setup for both types
		model.Name = sceneModel.Name
		model.SetPosition(sceneModel.Position[0], sceneModel.Position[1], sceneModel.Position[2])
		model.SetScale(sceneModel.Scale[0], sceneModel.Scale[1], sceneModel.Scale[2])

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

		// FIRST: Restore material properties BEFORE AddModel (actually model is already added for voxels, but that's fine)
		// This ensures materials have correct values from the start, preventing black rendering
		// Track unique materials to avoid restoring the same material multiple times
		restoredMaterials := make(map[*renderer.Material]bool)

		if model.Material != nil {
			// Use saved values only if they seem valid (not default zero)
			// This protects newly regenerated voxels (which have good defaults) from being overwritten by bad save data
			if sceneModel.DiffuseColor != [3]float32{0, 0, 0} {
				model.Material.DiffuseColor = sceneModel.DiffuseColor
			}
			if sceneModel.SpecularColor != [3]float32{0, 0, 0} {
				model.Material.SpecularColor = sceneModel.SpecularColor
			}

			model.Material.Shininess = sceneModel.Shininess
			model.Material.Metallic = sceneModel.Metallic
			model.Material.Roughness = sceneModel.Roughness

			// Ensure Exposure is never 0 (which would make model completely black)
			if sceneModel.Exposure > 0.01 {
				model.Material.Exposure = sceneModel.Exposure
			} else if model.Material.Exposure < 0.01 {
				// If saved is 0 AND current is 0, force default
				model.Material.Exposure = 1.0
			}
			// If saved is 0 but current is valid (from regenerate), keep current!

			// Ensure Alpha is never 0 (which would make model invisible)
			if sceneModel.Alpha > 0.01 {
				model.Material.Alpha = sceneModel.Alpha
			} else if model.Material.Alpha < 0.01 {
				model.Material.Alpha = 1.0
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

		// Mark model as dirty to ensure uniforms are updated on next render
		model.IsDirty = true

		// Force texture loading if path is set (renderer will auto-load on next AddModel/render)
		if model.Material != nil && model.Material.TexturePath != "" && model.Material.TextureID == 0 {
			// Texture will be loaded automatically by renderer on next render
			logToConsole(fmt.Sprintf("Texture queued for loading: %s", filepath.Base(model.Material.TexturePath)), "info")
		}

		// Create GameObject and restore components
		obj := createGameObjectForModel(model)
		for _, sceneComp := range sceneModel.Components {
			comp := behaviour.CreateScript(sceneComp.Type)
			if comp != nil {
				obj.AddComponent(comp)
				logToConsole(fmt.Sprintf("Restored component %s to %s", sceneComp.Type, model.Name), "info")
			}
		}
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
				Eng.Light = light
				isFirstLight = false
			}
		}
	}

	// Fallback: If no lights were loaded, create a default Sun to prevent black scene
	if Eng.Light == nil {
		logToConsole("No lights found in scene file, creating default Sun", "warning")
		defaultLight := renderer.CreateDirectionalLight(
			mgl.Vec3{-0.3, 0.8, -0.5}.Normalize(),
			mgl.Vec3{1.0, 0.95, 0.85},
			1.0,
		)
		defaultLight.Name = "DirectionalLight"
		defaultLight.AmbientStrength = 0.3
		defaultLight.Type = renderer.STATIC_LIGHT
		openglRenderer.AddLight(defaultLight)
		Eng.Light = defaultLight
	}

	// Load water if it exists in scene
	if sceneData.Water != nil {
		ws := NewWaterSimulation(Eng, sceneData.Water.OceanSize, sceneData.Water.BaseAmplitude)
		ws.WaterColor = mgl.Vec3{sceneData.Water.WaterColor[0], sceneData.Water.WaterColor[1], sceneData.Water.WaterColor[2]}
		ws.Transparency = sceneData.Water.Transparency
		ws.WaveSpeedMultiplier = sceneData.Water.WaveSpeedMultiplier
		ws.FoamEnabled = sceneData.Water.FoamEnabled
		ws.FoamIntensity = sceneData.Water.FoamIntensity
		ws.CausticsEnabled = sceneData.Water.CausticsEnabled
		ws.CausticsIntensity = sceneData.Water.CausticsIntensity
		ws.CausticsScale = sceneData.Water.CausticsScale
		ws.SpecularIntensity = sceneData.Water.SpecularIntensity
		ws.NormalStrength = sceneData.Water.NormalStrength
		ws.DistortionStrength = sceneData.Water.DistortionStrength
		ws.ShadowStrength = sceneData.Water.ShadowStrength

		activeWaterSim = ws
		behaviour.GlobalBehaviourManager.Add(ws)

		logToConsole("Water loaded from scene", "info")
	}

	// Load camera settings if present
	if sceneData.Camera != nil && Eng.Camera != nil {
		Eng.Camera.Position = mgl.Vec3{sceneData.Camera.Position[0], sceneData.Camera.Position[1], sceneData.Camera.Position[2]}
		Eng.Camera.Yaw = sceneData.Camera.Rotation[0]
		Eng.Camera.Pitch = sceneData.Camera.Rotation[1]
		if sceneData.Camera.Speed > 0 {
			Eng.Camera.Speed = sceneData.Camera.Speed
		}
		if sceneData.Camera.FOV > 0 {
			Eng.Camera.Fov = sceneData.Camera.FOV
		}
		if sceneData.Camera.Near > 0 {
			Eng.Camera.Near = sceneData.Camera.Near
		}
		if sceneData.Camera.Far > 0 {
			Eng.Camera.Far = sceneData.Camera.Far
		}
		Eng.Camera.UpdateProjection()
		logToConsole("Camera settings loaded from scene", "info")
	}

	// Load multiple cameras if present
	if len(sceneData.Cameras) > 0 {
		SceneCameras = make([]*renderer.Camera, len(sceneData.Cameras))
		for i, camData := range sceneData.Cameras {
			cam := &renderer.Camera{
				Name:        camData.Name,
				Position:    mgl.Vec3{camData.Position[0], camData.Position[1], camData.Position[2]},
				Yaw:         camData.Rotation[0],
				Pitch:       camData.Rotation[1],
				Speed:       camData.Speed,
				Fov:         camData.FOV,
				Near:        camData.Near,
				Far:         camData.Far,
				InvertMouse: camData.InvertMouse,
				IsActive:    camData.IsActive,
				WorldUp:     mgl.Vec3{0, 1, 0},
				Front:       mgl.Vec3{0, 0, -1},
				Up:          mgl.Vec3{0, 1, 0},
				Sensitivity: 0.1,
			}
			if cam.Near == 0 {
				cam.Near = 0.1
			}
			if cam.Far == 0 {
				cam.Far = 10000.0
			}
			if cam.Fov == 0 {
				cam.Fov = 45.0
			}
			if cam.Speed == 0 {
				cam.Speed = 70.0
			}
			cam.UpdateProjection()
			SceneCameras[i] = cam
		}
		logToConsole(fmt.Sprintf("Loaded %d cameras from scene", len(SceneCameras)), "info")
	}

	// Load skybox if it exists in scene
	if sceneData.Skybox != nil {
		if sceneData.Skybox.Type == "color" {
			skyboxColorMode = true
			skyboxSolidColor = sceneData.Skybox.Color
			openglRenderer.ClearColorR = sceneData.Skybox.Color[0]
			openglRenderer.ClearColorG = sceneData.Skybox.Color[1]
			openglRenderer.ClearColorB = sceneData.Skybox.Color[2]
			logToConsole("Skybox color loaded from scene", "info")
		} else if sceneData.Skybox.Type == "image" && sceneData.Skybox.ImagePath != "" {
			skyboxColorMode = false
			skyboxTexturePath = sceneData.Skybox.ImagePath
			// Load the skybox texture
			skybox, err := renderer.CreateSkybox(skyboxTexturePath)
			if err != nil {
				logToConsole(fmt.Sprintf("Failed to load skybox: %v", err), "error")
			} else {
				openglRenderer.SetSkybox(skybox)
				logToConsole("Skybox image loaded from scene", "info")
			}
		}
	} else {
		// No skybox saved - use default color
		skyboxColorMode = true
		skyboxSolidColor = [3]float32{0.5, 0.7, 1.0}
		openglRenderer.ClearColorR = 0.5
		openglRenderer.ClearColorG = 0.7
		openglRenderer.ClearColorB = 1.0
		logToConsole("Using default skybox color", "info")
	}

	// Load rendering configuration if present
	if sceneData.Rendering != nil {
		openglRenderer.EnableBloom = sceneData.Rendering.Bloom
		openglRenderer.EnableFXAA = sceneData.Rendering.FXAA
		renderer.DepthTestEnabled = sceneData.Rendering.DepthTest
		renderer.FaceCullingEnabled = sceneData.Rendering.FaceCulling
		renderer.Debug = sceneData.Rendering.Wireframe
		logToConsole("Rendering configuration loaded from scene", "info")
	}

	currentScenePath = filename
	sceneModified = false
	logToConsole(fmt.Sprintf("Scene loaded: %s (%d models, %d lights)", filepath.Base(filename), len(sceneData.Models), len(sceneData.Lights)), "info")
}

func addModelToScene(path string, name string) *renderer.Model {
	fmt.Printf("Loading model: %s from %s\n", name, path)
	logToConsole(fmt.Sprintf("Loading model: %s", name), "info")

	model, err := loader.LoadObjectWithPath(path, true)
	if err != nil {
		fmt.Printf("ERROR: Failed to load model: %v\n", err)
		logToConsole(fmt.Sprintf("Failed to load model: %v", err), "error")
		return nil
	}

	model.Name = name

	// Position new models slightly offset so they don't overlap
	models := Eng.GetRenderer().(*renderer.OpenGLRenderer).GetModels()
	offset := float32(len(models)) * 5.0
	model.SetPosition(offset, 10, 0)
	model.SetScale(10, 10, 10)

	// Ensure proper material defaults
	if model.Material == nil {
		model.Material = &renderer.Material{
			Name:         "Default",
			DiffuseColor: [3]float32{0.8, 0.8, 0.8},
			Metallic:     0.0,
			Roughness:    0.5,
			Alpha:        1.0,
			Exposure:     1.0,
		}
	} else {
		// Fix common issues with loaded materials
		if model.Material.Exposure == 0 {
			model.Material.Exposure = 1.0
		}
		if model.Material.Alpha == 0 {
			model.Material.Alpha = 1.0
		}
	}

	// Always ensure model has proper exposure set via method
	model.SetExposure(1.0)

	// Apply default advanced rendering configuration to new models
	if globalAdvancedRenderingEnabled {
		defaultConfig := renderer.DefaultAdvancedRenderingConfig()
		renderer.ApplyAdvancedRenderingConfig(model, defaultConfig)
		logToConsole("Applied advanced rendering config to new model", "info")
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

	Eng.AddModel(model)

	createGameObjectForModel(model)

	return model
}

func createGameObjectForModel(model *renderer.Model) *behaviour.GameObject {
	obj := behaviour.NewGameObject(model.Name)
	obj.SetModel(model)
	obj.Transform.SetPosition(model.Position)
	obj.Transform.SetRotation(model.Rotation)
	obj.Transform.SetScale(model.Scale)

	modelToGameObject[model] = obj
	behaviour.GlobalComponentManager.RegisterGameObject(obj)

	return obj
}

// addEmptyGameObject creates an empty GameObject with no model attached
// Users can add components to it via the Inspector
func addEmptyGameObject() {
	// Find unique name
	baseName := "GameObject"
	name := baseName
	counter := 1

	// Check existing GameObjects for name conflicts
	existingNames := make(map[string]bool)
	for _, obj := range behaviour.GlobalComponentManager.GetAllGameObjects() {
		existingNames[obj.Name] = true
	}

	for existingNames[name] {
		name = fmt.Sprintf("%s (%d)", baseName, counter)
		counter++
	}

	obj := behaviour.NewGameObject(name)
	obj.Transform.SetPosition(mgl.Vec3{0, 0, 0})
	obj.Transform.Rotation = mgl.QuatIdent()
	obj.Transform.SetScale(mgl.Vec3{1, 1, 1})

	behaviour.GlobalComponentManager.RegisterGameObject(obj)

	logToConsole(fmt.Sprintf("Created empty GameObject: %s", name), "info")
}

func getGameObjectForModel(model *renderer.Model) *behaviour.GameObject {
	return modelToGameObject[model]
}

func removeGameObjectForModel(model *renderer.Model) {
	if obj, exists := modelToGameObject[model]; exists {
		behaviour.GlobalComponentManager.UnregisterGameObject(obj)
		delete(modelToGameObject, model)
	}
}

func getComponentTypeName(comp behaviour.Component) string {
	typeName := fmt.Sprintf("%T", comp)
	parts := strings.Split(typeName, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return typeName
}

// AddSceneCamera creates a new camera in the scene
func AddSceneCamera(name string) *renderer.Camera {
	if name == "" {
		name = fmt.Sprintf("Camera %d", len(SceneCameras)+1)
	}

	cam := &renderer.Camera{
		Name:        name,
		Position:    mgl.Vec3{0, 10, 30},
		Front:       mgl.Vec3{0, 0, -1},
		Up:          mgl.Vec3{0, 1, 0},
		WorldUp:     mgl.Vec3{0, 1, 0},
		Pitch:       0,
		Yaw:         -90,
		Speed:       70,
		Sensitivity: 0.1,
		Fov:         45,
		Near:        0.1,
		Far:         10000,
		InvertMouse: false,
		IsActive:    len(SceneCameras) == 0, // First camera is active by default
	}
	cam.UpdateProjection()

	SceneCameras = append(SceneCameras, cam)
	logToConsole(fmt.Sprintf("Added camera: %s", name), "info")
	return cam
}

// RemoveSceneCamera removes a camera from the scene
func RemoveSceneCamera(index int) {
	if index < 0 || index >= len(SceneCameras) {
		return
	}

	name := SceneCameras[index].Name
	SceneCameras = append(SceneCameras[:index], SceneCameras[index+1:]...)
	logToConsole(fmt.Sprintf("Removed camera: %s", name), "info")
}

// SetActiveCamera sets which camera is the active one for the game
func SetActiveCamera(index int) {
	for i := range SceneCameras {
		SceneCameras[i].IsActive = (i == index)
	}
}

func SetupEditorScene() {
	fmt.Println("Setting up editor scene...")

	// Check if camera is ready (should be by now, but be safe)
	if Eng.Camera == nil {
		fmt.Println("Warning: Camera not ready yet, skipping scene setup")
		SceneSetup = false
		return
	}

	Eng.Camera.Position = mgl.Vec3{0, 50, 150}
	Eng.Camera.Speed = 100
	Eng.Camera.InvertMouse = false

	// Create default light
	defaultLight := renderer.CreateDirectionalLight(
		mgl.Vec3{-0.3, 0.8, -0.5}.Normalize(),
		mgl.Vec3{1.0, 0.95, 0.85},
		1.0,
	)
	defaultLight.Name = "DirectionalLight"
	defaultLight.AmbientStrength = 0.3
	defaultLight.Type = renderer.STATIC_LIGHT
	Eng.Light = defaultLight

	// Add light to renderer's lights array (so editor can manage it)
	openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
	if ok {
		openglRenderer.AddLight(defaultLight)
	}

	// Note: Grid floor removed - it was interfering with the scene
	// TODO: Implement proper debug grid lines if needed

	fmt.Println("✓ Editor scene ready!")

	LoadConfig()

	// Add initial console message
	logToConsole("Editor initialized - Type 'help' for available commands", "info")
}

// serializeComponents converts components to serializable format
func serializeComponents(components []behaviour.Component) []SceneComponent {
	result := make([]SceneComponent, 0)

	for _, comp := range components {
		sceneComp := SceneComponent{
			Type:       behaviour.GetComponentTypeName(comp),
			Category:   string(behaviour.GetComponentCategory(comp)),
			Properties: make(map[string]interface{}),
		}

		// Serialize component-specific properties
		switch c := comp.(type) {
		case *behaviour.MeshComponent:
			sceneComp.Properties["mesh_path"] = c.MeshPath
			sceneComp.Properties["material_path"] = c.MaterialPath
			sceneComp.Properties["diffuse_color"] = c.DiffuseColor
			sceneComp.Properties["specular_color"] = c.SpecularColor
			sceneComp.Properties["metallic"] = c.Metallic
			sceneComp.Properties["roughness"] = c.Roughness
			sceneComp.Properties["alpha"] = c.Alpha

		case *behaviour.WaterComponent:
			sceneComp.Properties["ocean_size"] = c.OceanSize
			sceneComp.Properties["base_amplitude"] = c.BaseAmplitude
			sceneComp.Properties["water_color"] = c.WaterColor
			sceneComp.Properties["transparency"] = c.Transparency
			sceneComp.Properties["wave_speed"] = c.WaveSpeedMultiplier
			sceneComp.Properties["foam_enabled"] = c.FoamEnabled
			sceneComp.Properties["foam_intensity"] = c.FoamIntensity
			sceneComp.Properties["caustics_enabled"] = c.CausticsEnabled
			sceneComp.Properties["caustics_intensity"] = c.CausticsIntensity

		case *behaviour.VoxelTerrainComponent:
			sceneComp.Properties["scale"] = c.Scale
			sceneComp.Properties["amplitude"] = c.Amplitude
			sceneComp.Properties["seed"] = c.Seed
			sceneComp.Properties["threshold"] = c.Threshold
			sceneComp.Properties["octaves"] = c.Octaves
			sceneComp.Properties["chunk_size"] = c.ChunkSize
			sceneComp.Properties["world_size"] = c.WorldSize
			sceneComp.Properties["biome"] = c.Biome
			sceneComp.Properties["tree_density"] = c.TreeDensity
			sceneComp.Properties["grass_color"] = c.GrassColor
			sceneComp.Properties["dirt_color"] = c.DirtColor
			sceneComp.Properties["stone_color"] = c.StoneColor
			sceneComp.Properties["sand_color"] = c.SandColor
			sceneComp.Properties["wood_color"] = c.WoodColor
			sceneComp.Properties["leaves_color"] = c.LeavesColor

		case *behaviour.LightComponent:
			sceneComp.Properties["light_mode"] = c.LightMode
			sceneComp.Properties["color"] = c.Color
			sceneComp.Properties["intensity"] = c.Intensity
			sceneComp.Properties["range"] = c.Range
			sceneComp.Properties["ambient_strength"] = c.AmbientStrength

		case *behaviour.CameraComponent:
			sceneComp.Properties["fov"] = c.FOV
			sceneComp.Properties["near"] = c.Near
			sceneComp.Properties["far"] = c.Far
			sceneComp.Properties["is_main"] = c.IsMain

		case *behaviour.ScriptComponent:
			sceneComp.Properties["script_name"] = c.ScriptName
		}

		result = append(result, sceneComp)
	}

	return result
}

// deserializeComponents reconstructs components from serialized data
func deserializeComponents(sceneComps []SceneComponent) []behaviour.Component {
	result := make([]behaviour.Component, 0)

	for _, sc := range sceneComps {
		var comp behaviour.Component

		switch sc.Category {
		case string(behaviour.ComponentTypeMesh):
			c := behaviour.NewMeshComponent()
			if v, ok := sc.Properties["mesh_path"].(string); ok {
				c.MeshPath = v
			}
			if v, ok := sc.Properties["material_path"].(string); ok {
				c.MaterialPath = v
			}
			if v, ok := sc.Properties["metallic"].(float64); ok {
				c.Metallic = float32(v)
			}
			if v, ok := sc.Properties["roughness"].(float64); ok {
				c.Roughness = float32(v)
			}
			if v, ok := sc.Properties["alpha"].(float64); ok {
				c.Alpha = float32(v)
			}
			comp = c

		case string(behaviour.ComponentTypeWater):
			c := behaviour.NewWaterComponent()
			if v, ok := sc.Properties["ocean_size"].(float64); ok {
				c.OceanSize = float32(v)
			}
			if v, ok := sc.Properties["base_amplitude"].(float64); ok {
				c.BaseAmplitude = float32(v)
			}
			if v, ok := sc.Properties["transparency"].(float64); ok {
				c.Transparency = float32(v)
			}
			if v, ok := sc.Properties["wave_speed"].(float64); ok {
				c.WaveSpeedMultiplier = float32(v)
			}
			if v, ok := sc.Properties["foam_enabled"].(bool); ok {
				c.FoamEnabled = v
			}
			if v, ok := sc.Properties["caustics_enabled"].(bool); ok {
				c.CausticsEnabled = v
			}
			comp = c

		case string(behaviour.ComponentTypeVoxel):
			c := behaviour.NewVoxelTerrainComponent()
			if v, ok := sc.Properties["scale"].(float64); ok {
				c.Scale = float32(v)
			}
			if v, ok := sc.Properties["amplitude"].(float64); ok {
				c.Amplitude = float32(v)
			}
			if v, ok := sc.Properties["seed"].(float64); ok {
				c.Seed = int32(v)
			}
			if v, ok := sc.Properties["threshold"].(float64); ok {
				c.Threshold = float32(v)
			}
			if v, ok := sc.Properties["octaves"].(float64); ok {
				c.Octaves = int32(v)
			}
			if v, ok := sc.Properties["chunk_size"].(float64); ok {
				c.ChunkSize = int32(v)
			}
			if v, ok := sc.Properties["world_size"].(float64); ok {
				c.WorldSize = int32(v)
			}
			if v, ok := sc.Properties["biome"].(float64); ok {
				c.Biome = int32(v)
			}
			if v, ok := sc.Properties["tree_density"].(float64); ok {
				c.TreeDensity = float32(v)
			}
			comp = c

		case string(behaviour.ComponentTypeLight):
			c := behaviour.NewLightComponent()
			if v, ok := sc.Properties["light_mode"].(string); ok {
				c.LightMode = v
			}
			if v, ok := sc.Properties["intensity"].(float64); ok {
				c.Intensity = float32(v)
			}
			if v, ok := sc.Properties["range"].(float64); ok {
				c.Range = float32(v)
			}
			if v, ok := sc.Properties["ambient_strength"].(float64); ok {
				c.AmbientStrength = float32(v)
			}
			comp = c

		case string(behaviour.ComponentTypeCamera):
			c := behaviour.NewCameraComponent()
			if v, ok := sc.Properties["fov"].(float64); ok {
				c.FOV = float32(v)
			}
			if v, ok := sc.Properties["near"].(float64); ok {
				c.Near = float32(v)
			}
			if v, ok := sc.Properties["far"].(float64); ok {
				c.Far = float32(v)
			}
			if v, ok := sc.Properties["is_main"].(bool); ok {
				c.IsMain = v
			}
			comp = c

		case string(behaviour.ComponentTypeScript):
			if scriptName, ok := sc.Properties["script_name"].(string); ok {
				script := behaviour.CreateScript(scriptName)
				if script != nil {
					comp = behaviour.NewScriptComponent(scriptName, script)
				}
			}

		default:
			// Try to create by type name for backward compatibility
			if sc.Type != "" {
				script := behaviour.CreateScript(sc.Type)
				if script != nil {
					comp = behaviour.NewScriptComponent(sc.Type, script)
				}
			}
		}

		if comp != nil {
			result = append(result, comp)
		}
	}

	return result
}

// quatToEulerArray converts quaternion to euler angles array
func quatToEulerArray(q mgl.Quat) [3]float32 {
	euler := quatToEuler(q)
	return [3]float32{euler.X(), euler.Y(), euler.Z()}
}
