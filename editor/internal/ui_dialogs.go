package editor

import (
	"Gopher3D/internal/behaviour"
	"Gopher3D/internal/renderer"
	"fmt"

	mgl "github.com/go-gl/mathgl/mgl32"
	"github.com/inkyblackness/imgui-go/v4"
)

// State variables for dialogs
var (
	addModelScale = [3]float32{1, 1, 1}
	addModelPos   = [3]float32{0, 0, 0}

	addLightType      = 0 // 0=Directional, 1=Point
	addLightColor     = [3]float32{1, 1, 1}
	addLightIntensity = float32(1.0)
	addLightRange     = float32(100.0) // For point light

	addWaterSize      = float32(1000.0) // Reduced default size for better editor usability
	addWaterAmplitude = float32(5.0)    // Reduced amplitude for scale
)

func renderAddModelDialog() {
	if Eng == nil {
		return
	}
	imgui.OpenPopup("Add Mesh")

	centerX := float32(Eng.Width) / 2
	centerY := float32(Eng.Height) / 2
	imgui.SetNextWindowPosV(imgui.Vec2{X: centerX - 200, Y: centerY - 250}, imgui.ConditionAppearing, imgui.Vec2{})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 400, Y: 500}, imgui.ConditionAppearing)

	if imgui.BeginPopupModalV("Add Mesh", nil, imgui.WindowFlagsNoResize) {
		imgui.Text("Select a mesh to create a GameObject:")
		imgui.Separator()
		imgui.Spacing()

		imgui.Text("Initial Transform:")
		imgui.DragFloat3("Position", &addModelPos)
		imgui.DragFloat3("Scale", &addModelScale)

		imgui.Spacing()
		imgui.Separator()
		imgui.Text("Select Mesh:")

		// Model list with preview info
		imgui.BeginChildV("ModelList", imgui.Vec2{X: 0, Y: 300}, true, 0)
		for _, modelInfo := range availableModels {
			if imgui.SelectableV(modelInfo.Name, false, 0, imgui.Vec2{X: 0, Y: 30}) {
				// Create GameObject with MeshComponent
				createMeshGameObject(modelInfo.Path, modelInfo.Name, addModelPos, addModelScale)
				ShowAddModel = false
				imgui.CloseCurrentPopup()
			}
		}
		imgui.EndChild()

		imgui.Separator()
		imgui.Spacing()
		if imgui.Button("Cancel") {
			ShowAddModel = false
			imgui.CloseCurrentPopup()
		}
		imgui.EndPopup()
	}
}

// createMeshGameObject creates a GameObject with a MeshComponent
func createMeshGameObject(meshPath, name string, pos, scale [3]float32) *behaviour.GameObject {
	// Load the model
	model := addModelToScene(meshPath, name)
	if model == nil {
		return nil
	}

	// Apply transform
	model.SetPosition(pos[0], pos[1], pos[2])
	model.SetScale(scale[0], scale[1], scale[2])

	// Create MeshComponent
	meshComp := behaviour.NewMeshComponent()
	meshComp.MeshPath = meshPath
	meshComp.Model = model
	meshComp.Loaded = true

	// Copy material properties from loaded model
	if model.Material != nil {
		meshComp.DiffuseColor = model.Material.DiffuseColor
		meshComp.SpecularColor = model.Material.SpecularColor
		meshComp.Metallic = model.Material.Metallic
		meshComp.Roughness = model.Material.Roughness
		meshComp.Alpha = model.Material.Alpha
	}

	// Create GameObject
	obj := behaviour.NewGameObject(name)
	obj.Transform.SetPosition(mgl.Vec3{pos[0], pos[1], pos[2]})
	obj.Transform.SetScale(mgl.Vec3{scale[0], scale[1], scale[2]})
	obj.AddComponent(meshComp)
	obj.SetModel(model)

	// Register
	behaviour.GlobalComponentManager.RegisterGameObject(obj)

	logToConsole(fmt.Sprintf("Created GameObject '%s' with MeshComponent", name), "info")
	return obj
}

func renderAddLightDialog() {
	if Eng == nil {
		return
	}
	imgui.OpenPopup("Add Light")

	centerX := float32(Eng.Width) / 2
	centerY := float32(Eng.Height) / 2
	imgui.SetNextWindowPosV(imgui.Vec2{X: centerX - 200, Y: centerY - 200}, imgui.ConditionAppearing, imgui.Vec2{})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 400, Y: 400}, imgui.ConditionAppearing)

	if imgui.BeginPopupModalV("Add Light", nil, imgui.WindowFlagsNoResize) {
		imgui.Text("Configure Light:")
		imgui.Separator()

		imgui.RadioButtonInt("Directional Light", &addLightType, 0)
		imgui.SameLine()
		imgui.RadioButtonInt("Point Light", &addLightType, 1)

		imgui.Spacing()
		imgui.ColorEdit3("Color", &addLightColor)
		imgui.DragFloatV("Intensity", &addLightIntensity, 0.1, 0.0, 100.0, "%.1f", 1.0)

		if addLightType == 1 {
			imgui.DragFloatV("Range", &addLightRange, 1.0, 0.0, 10000.0, "%.1f", 1.0)
		}

		imgui.Separator()
		imgui.Spacing()

		if imgui.Button("Add Light") {
			var light *renderer.Light
			colorVec := mgl.Vec3{addLightColor[0], addLightColor[1], addLightColor[2]}

			if addLightType == 0 {
				// Directional
				light = renderer.CreateDirectionalLight(
					mgl.Vec3{-0.2, -1.0, -0.3}.Normalize(),
					colorVec,
					addLightIntensity,
				)
				light.Name = fmt.Sprintf("Directional Light %d", len(Eng.GetRenderer().(*renderer.OpenGLRenderer).GetLights())+1)
			} else {
				// Point
				light = renderer.CreatePointLight(
					Eng.Camera.Position,
					colorVec,
					addLightIntensity,
					addLightRange,
				)
				light.Name = fmt.Sprintf("Point Light %d", len(Eng.GetRenderer().(*renderer.OpenGLRenderer).GetLights())+1)
			}

			Eng.GetRenderer().(*renderer.OpenGLRenderer).AddLight(light)
			lights := Eng.GetRenderer().(*renderer.OpenGLRenderer).GetLights()
			if len(lights) > 0 {
				Eng.Light = lights[0] // Always use the first light as the main scene light
			}

			logToConsole(fmt.Sprintf("Added %s", light.Name), "info")

			ShowAddLight = false
			imgui.CloseCurrentPopup()
		}

		imgui.SameLine()
		if imgui.Button("Cancel") {
			ShowAddLight = false
			imgui.CloseCurrentPopup()
		}
		imgui.EndPopup()
	}
}

func renderAddWaterDialog() {
	if Eng == nil {
		return
	}
	imgui.OpenPopup("Add Water")

	centerX := float32(Eng.Width) / 2
	centerY := float32(Eng.Height) / 2
	imgui.SetNextWindowPosV(imgui.Vec2{X: centerX - 200, Y: centerY - 150}, imgui.ConditionAppearing, imgui.Vec2{})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 400, Y: 300}, imgui.ConditionAppearing)

	if imgui.BeginPopupModalV("Add Water", nil, imgui.WindowFlagsNoResize) {
		imgui.Text("Configure Ocean Simulation:")
		imgui.Separator()

		imgui.DragFloatV("Size (Units)", &addWaterSize, 1000.0, 1000.0, 10000000.0, "%.0f", 1.0)
		imgui.DragFloatV("Wave Amplitude", &addWaterAmplitude, 10.0, 0.0, 5000.0, "%.1f", 1.0)

		imgui.Separator()
		imgui.Spacing()

		if imgui.Button("Create Ocean") {
			if activeWaterSim == nil {
				// Use createWaterGameObject to properly create a GameObject with WaterComponent
				obj := createWaterGameObject()
				if obj != nil {
					logToConsole(fmt.Sprintf("Ocean created (Size: %.0f, Amp: %.1f)", addWaterSize, addWaterAmplitude), "info")
				}
			} else {
				logToConsole("Ocean already exists!", "warning")
			}
			ShowAddWater = false
			imgui.CloseCurrentPopup()
		}
		imgui.SameLine()
		if imgui.Button("Cancel") {
			ShowAddWater = false
			imgui.CloseCurrentPopup()
		}
		imgui.EndPopup()
	}
}
