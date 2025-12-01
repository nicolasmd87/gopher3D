package editor

import (
	"Gopher3D/internal/behaviour"
	"Gopher3D/internal/renderer"
	"Gopher3D/internal/water"

	mgl32 "github.com/go-gl/mathgl/mgl32"
)

// WaterSimulation is an alias to the shared water.Simulation for editor use
type WaterSimulation = water.Simulation

// WaterSimulationConfig is an alias to the shared water.Config
type WaterSimulationConfig = water.Config

// NewWaterSimulation creates a new water simulation using the shared package
func NewWaterSimulation(eng interface{}, size float32, amplitude float32) *WaterSimulation {
	// The engine interface needs to be cast properly
	return water.NewSimulation(Eng, size, amplitude)
}

// createWaterGameObject creates a GameObject with a WaterComponent
// Uses the dialog settings (addWaterSize, addWaterAmplitude) from ui_dialogs.go
func createWaterGameObject() *behaviour.GameObject {
	// Create the component with default settings
	waterComp := behaviour.NewWaterComponent()

	// Override with dialog settings if they're set
	if addWaterSize > 0 {
		waterComp.OceanSize = addWaterSize
	}
	if addWaterAmplitude > 0 {
		waterComp.BaseAmplitude = addWaterAmplitude
	}

	// Create GameObject
	obj := behaviour.NewGameObject("Water")
	obj.AddComponent(waterComp)

	// Create the water simulation with the component's settings
	ws := water.NewSimulation(Eng, waterComp.OceanSize, waterComp.BaseAmplitude)

	// Apply component settings to simulation
	ws.WaterColor = mgl32.Vec3{waterComp.WaterColor[0], waterComp.WaterColor[1], waterComp.WaterColor[2]}
	ws.Transparency = waterComp.Transparency
	ws.WaveSpeedMultiplier = waterComp.WaveSpeedMultiplier
	ws.WaveHeight = waterComp.WaveHeight
	ws.WaveRandomness = waterComp.WaveRandomness
	ws.FoamEnabled = waterComp.FoamEnabled
	ws.FoamIntensity = waterComp.FoamIntensity
	ws.CausticsEnabled = waterComp.CausticsEnabled
	ws.CausticsIntensity = waterComp.CausticsIntensity
	ws.CausticsScale = waterComp.CausticsScale
	ws.SpecularIntensity = waterComp.SpecularIntensity
	ws.NormalStrength = waterComp.NormalStrength
	ws.DistortionStrength = waterComp.DistortionStrength
	ws.ShadowStrength = waterComp.ShadowStrength

	// Set sky color from editor
	ws.SetSkyColor(mgl32.Vec3{skyboxSolidColor[0], skyboxSolidColor[1], skyboxSolidColor[2]})

	// Use InitializeMesh() to create the model
	model := ws.InitializeMesh()
	if model == nil {
		logToConsole("Failed to create water mesh", "error")
		return nil
	}

	// Store references
	waterComp.Simulation = ws
	waterComp.Model = model
	waterComp.Generated = true
	obj.SetModel(model)

	// Add model to engine
	Eng.AddModel(model)
	// Register model-to-GameObject mapping for scene saving
	registerModelToGameObject(model, obj)

	// Add to behaviour manager for updates
	behaviour.GlobalBehaviourManager.Add(ws)
	activeWaterSim = ws

	// Register the GameObject
	behaviour.GlobalComponentManager.RegisterGameObject(obj)

	logToConsole("Water GameObject created", "info")
	return obj
}

// SyncWaterComponentToSimulation updates the water simulation from component values
func SyncWaterComponentToSimulation(comp *behaviour.WaterComponent) {
	if comp.Simulation == nil {
		return
	}
	ws, ok := comp.Simulation.(*water.Simulation)
	if !ok {
		return
	}

	ws.OceanSize = comp.OceanSize
	ws.BaseAmplitude = comp.BaseAmplitude
	ws.WaterColor = mgl32.Vec3{comp.WaterColor[0], comp.WaterColor[1], comp.WaterColor[2]}
	ws.Transparency = comp.Transparency
	ws.WaveSpeedMultiplier = comp.WaveSpeedMultiplier
	ws.WaveHeight = comp.WaveHeight
	ws.WaveRandomness = comp.WaveRandomness
	ws.FoamEnabled = comp.FoamEnabled
	ws.FoamIntensity = comp.FoamIntensity
	ws.CausticsEnabled = comp.CausticsEnabled
	ws.CausticsIntensity = comp.CausticsIntensity
	ws.CausticsScale = comp.CausticsScale
	ws.SpecularIntensity = comp.SpecularIntensity
	ws.NormalStrength = comp.NormalStrength
	ws.DistortionStrength = comp.DistortionStrength
	ws.ShadowStrength = comp.ShadowStrength

	// Apply changes immediately
	ws.ApplyChanges()
}

// regenerateWater recreates the water mesh with new size/amplitude settings
func regenerateWater(comp *behaviour.WaterComponent) {
	if comp.Simulation == nil {
		return
	}
	ws, ok := comp.Simulation.(*water.Simulation)
	if !ok {
		return
	}

	// Get current position before removing
	var oldPos mgl32.Vec3
	if ws.Model != nil {
		oldPos = ws.Model.Position
	}

	// Remove old model from engine
	if ws.Model != nil {
		if openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer); ok {
			openglRenderer.RemoveModel(ws.Model)
		}
	}

	// Update simulation size parameters
	ws.OceanSize = comp.OceanSize
	ws.BaseAmplitude = comp.BaseAmplitude

	// Clear existing model reference so InitializeMesh creates a new one
	ws.Model = nil

	// Create new mesh
	model := ws.InitializeMesh()
	if model == nil {
		logToConsole("Failed to regenerate water mesh", "error")
		return
	}

	// Restore position
	model.SetPosition(oldPos.X(), oldPos.Y(), oldPos.Z())

	// Update component references
	comp.Model = model

	// Update GameObject model reference
	if gameObj := comp.GetGameObject(); gameObj != nil {
		gameObj.SetModel(model)
		// Update the mapping
		registerModelToGameObject(model, gameObj)
	}

	// Add new model to engine
	Eng.AddModel(model)

	// Sync all other properties
	SyncWaterComponentToSimulation(comp)

	logToConsole("Water regenerated with new size", "info")
}

// UpdateWaterSkyColor updates the sky color for all water simulations
// Called when the editor's skybox color changes
func UpdateWaterSkyColor() {
	if activeWaterSim != nil {
		activeWaterSim.SetSkyColor(mgl32.Vec3{skyboxSolidColor[0], skyboxSolidColor[1], skyboxSolidColor[2]})
	}
}

// RestoreWaterSimulation restores a water simulation from a saved configuration
func RestoreWaterSimulation(model interface{}, config water.Config) *water.Simulation {
	ws := water.NewSimulation(Eng, config.OceanSize, config.BaseAmplitude)
	ws.ApplyConfig(config)

	// Link model if provided
	if m, ok := model.(*interface{}); ok && m != nil {
		// Model linking handled by caller
	}

	// Set sky color from editor
	ws.SetSkyColor(mgl32.Vec3{skyboxSolidColor[0], skyboxSolidColor[1], skyboxSolidColor[2]})

	return ws
}
