package main

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
	Models []SceneModel `json:"models"`
	Lights []SceneLight `json:"lights"`
	Water  *SceneWater  `json:"water,omitempty"` // Optional water configuration
}

type SceneModel struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
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
	TexturePath   string     `json:"texture_path"`
	
	// Voxel Specific Data
	VoxelConfig *VoxelConfig `json:"voxel_config,omitempty"`
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
	// Create a separate slice for removal to avoid index issues or modifying the slice we are iterating
	var lightsToRemove []*renderer.Light
	for _, light := range lights {
		if light.Name != "Sun" {
			lightsToRemove = append(lightsToRemove, light)
		}
	}
	for _, light := range lightsToRemove {
		openglRenderer.RemoveLight(light)
	}

	// RE-FETCH lights to ensure we have the up-to-date list after removals
	lights = openglRenderer.GetLights()

	// Ensure we have at least one default light
	if len(lights) == 0 {
		defaultLight := renderer.CreateDirectionalLight(
			mgl.Vec3{0.3, -0.8, 0.5}.Normalize(),
			mgl.Vec3{1.0, 0.95, 0.85},
			4.5, // Much higher intensity for water reflections
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
	
	// Reset Water
	activeWaterSim = nil

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
		
		// Check for voxel configuration
		if model.Metadata != nil {
			if config, ok := model.Metadata["voxelConfig"].(VoxelConfig); ok {
				sceneModel.VoxelConfig = &config
			}
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

	// Save water if it exists
	if activeWaterSim != nil && activeWaterSim.model != nil {
		sceneData.Water = &SceneWater{
			OceanSize:           activeWaterSim.oceanSize,
			BaseAmplitude:       activeWaterSim.baseAmplitude,
			WaterColor:          [3]float32{activeWaterSim.WaterColor.X(), activeWaterSim.WaterColor.Y(), activeWaterSim.WaterColor.Z()},
			Transparency:        activeWaterSim.Transparency,
			WaveSpeedMultiplier: activeWaterSim.WaveSpeedMultiplier,
			Position:            [3]float32{activeWaterSim.model.Position.X(), activeWaterSim.model.Position.Y(), activeWaterSim.model.Position.Z()},
		}
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
		var model *renderer.Model
		
		// Case 1: Voxel Terrain
		if sceneModel.VoxelConfig != nil {
			logToConsole(fmt.Sprintf("Regenerating voxel terrain: %s", sceneModel.Name), "info")
			model = regenerateVoxelTerrain(*sceneModel.VoxelConfig)
			if model == nil {
				logToConsole(fmt.Sprintf("Failed to regenerate voxel terrain: %s", sceneModel.Name), "error")
				continue
			}
			// Manually add model here since regenerateVoxelTerrain no longer does it
			eng.AddModel(model)
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
			eng.AddModel(model)
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
			if sceneModel.DiffuseColor != [3]float32{0,0,0} {
				model.Material.DiffuseColor = sceneModel.DiffuseColor
			}
			if sceneModel.SpecularColor != [3]float32{0,0,0} {
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

	// Fallback: If no lights were loaded, create a default Sun to prevent black scene
	if eng.Light == nil {
		logToConsole("No lights found in scene file, creating default Sun", "warning")
		defaultLight := renderer.CreateDirectionalLight(
			mgl.Vec3{0.3, -0.8, 0.5}.Normalize(),
			mgl.Vec3{1.0, 0.95, 0.85},
			4.5, // Much higher intensity for water reflections
		)
		defaultLight.Name = "Sun"
		defaultLight.AmbientStrength = 0.3
		defaultLight.Type = renderer.STATIC_LIGHT
		openglRenderer.AddLight(defaultLight)
		eng.Light = defaultLight
	}

	// Load water if it exists in scene
	if sceneData.Water != nil {
		ws := NewWaterSimulation(eng, sceneData.Water.OceanSize, sceneData.Water.BaseAmplitude)
		ws.WaterColor = mgl.Vec3{sceneData.Water.WaterColor[0], sceneData.Water.WaterColor[1], sceneData.Water.WaterColor[2]}
		ws.Transparency = sceneData.Water.Transparency
		ws.WaveSpeedMultiplier = sceneData.Water.WaveSpeedMultiplier
		
		activeWaterSim = ws
		behaviour.GlobalBehaviourManager.Add(ws)
		
		logToConsole("Water loaded from scene", "info")
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
	return model
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
		mgl.Vec3{0.3, -0.8, 0.5}.Normalize(), // Sun angle from upper-right
		mgl.Vec3{1.0, 0.95, 0.85},
		4.5, // Much higher intensity for water reflections
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
