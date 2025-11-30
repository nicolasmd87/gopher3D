package editor

import (
	"Gopher3D/internal/renderer"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/sqweek/dialog"
)

var (
	showExportDialog   = false
	exportGameName     = "MyGame"
	exportOutputDir    = ""
	exportScenePaths   = []string{} // Multiple scenes
	exportIncludeScene = true
	exportProgress     = float32(0)
	exportStatus       = ""
	exportInProgress   = false
)

type ExportConfig struct {
	GameName      string
	OutputDir     string
	Platforms     []string
	ScenePaths    []string
	ProjectPath   string
	IncludeAssets bool
}

func renderExportDialog() {
	if !showExportDialog {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: float32(Eng.Width)/2 - 200, Y: float32(Eng.Height)/2 - 150}, imgui.ConditionFirstUseEver, imgui.Vec2{})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 400, Y: 380}, imgui.ConditionFirstUseEver)

	if imgui.BeginV("Export Game", &showExportDialog, 0) {
		imgui.Text("Export your game as a standalone executable")
		imgui.Separator()
		imgui.Spacing()

		imgui.Text("Game Name:")
		imgui.InputText("##gamename", &exportGameName)

		imgui.Spacing()
		imgui.Text("Output Directory:")
		imgui.InputText("##outputdir", &exportOutputDir)
		imgui.SameLine()
		if imgui.Button("Browse...") {
			dir, err := dialog.Directory().Title("Select Output Directory").Browse()
			if err == nil {
				exportOutputDir = dir
			}
		}

		imgui.Spacing()
		imgui.Separator()
		imgui.Text("Target Platform:")
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.5, Y: 0.5, Z: 0.5, W: 1})
		imgui.Text("(Builds for current platform only)")
		imgui.PopStyleColor()
		imgui.Spacing()
		imgui.Text("  • Windows (current)")

		imgui.Spacing()
		imgui.Separator()
		imgui.Text("Scenes to Export:")

		// Show current scene if available
		if currentScenePath != "" {
			imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.5, Y: 0.8, Z: 0.5, W: 1})
			imgui.Text("Current: " + filepath.Base(currentScenePath))
			imgui.PopStyleColor()
			imgui.SameLine()
			if imgui.Button("Add Current") {
				addSceneToExport(currentScenePath)
			}
		}

		// Add scene button
		if imgui.Button("Add Scene...") {
			file, err := dialog.File().Filter("Scene files", "json").Title("Select Scene File").Load()
			if err == nil {
				addSceneToExport(file)
			}
		}

		// List of scenes to export
		imgui.BeginChildV("SceneList", imgui.Vec2{X: 0, Y: 80}, true, 0)
		for i, scenePath := range exportScenePaths {
			imgui.PushID(fmt.Sprintf("scene_%d", i))
			imgui.Text(filepath.Base(scenePath))
			imgui.SameLine()
			if imgui.Button("X") {
				exportScenePaths = append(exportScenePaths[:i], exportScenePaths[i+1:]...)
			}
			imgui.PopID()
		}
		if len(exportScenePaths) == 0 {
			imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.6, Y: 0.6, Z: 0.6, W: 1})
			imgui.Text("No scenes added")
			imgui.PopStyleColor()
		}
		imgui.EndChild()

		imgui.Checkbox("Include scene assets", &exportIncludeScene)

		imgui.Separator()

		if exportInProgress {
			imgui.ProgressBar(exportProgress)
			imgui.Text(exportStatus)
		} else {
			canExport := len(exportScenePaths) > 0 || !exportIncludeScene
			if imgui.Button("Export") && canExport {
				go startExport()
			}
			if !canExport {
				imgui.SameLine()
				imgui.Text("(Add at least one scene)")
			}
			imgui.SameLine()
			if imgui.Button("Cancel") {
				showExportDialog = false
			}
		}
	}
	imgui.End()
}

func addSceneToExport(path string) {
	// Check if already added
	for _, p := range exportScenePaths {
		if p == path {
			return
		}
	}
	exportScenePaths = append(exportScenePaths, path)
}

func startExport() {
	if exportOutputDir == "" {
		logToConsole("Export failed: No output directory selected", "error")
		return
	}

	if exportGameName == "" {
		logToConsole("Export failed: No game name specified", "error")
		return
	}

	// Only build for current platform (Windows) due to CGO
	platforms := []string{"windows"}

	exportInProgress = true
	exportProgress = 0
	exportStatus = "Analyzing scenes..."

	projectPath := ""
	if CurrentProject != nil {
		projectPath = CurrentProject.Path
	}

	// Use selected scenes or current scene
	scenePaths := exportScenePaths
	if len(scenePaths) == 0 && currentScenePath != "" {
		scenePaths = []string{currentScenePath}
	}

	config := ExportConfig{
		GameName:      exportGameName,
		OutputDir:     exportOutputDir,
		Platforms:     platforms,
		ScenePaths:    scenePaths,
		ProjectPath:   projectPath,
		IncludeAssets: exportIncludeScene,
	}

	err := exportGame(config)
	if err != nil {
		logToConsole(fmt.Sprintf("Export failed: %v", err), "error")
		exportStatus = fmt.Sprintf("Export failed: %v", err)
	} else {
		logToConsole(fmt.Sprintf("Game exported successfully to %s", exportOutputDir), "info")
		exportStatus = "Export complete!"
	}

	exportInProgress = false
}

func exportGame(config ExportConfig) error {
	totalSteps := float32(len(config.Platforms) * 4)
	currentStep := float32(0)

	// Read and analyze all scenes to understand what's needed
	var allSceneData []*SceneData
	var primaryScene *SceneData

	if len(config.ScenePaths) > 0 && config.IncludeAssets {
		exportStatus = "Reading scene files..."
		for i, scenePath := range config.ScenePaths {
			data, err := os.ReadFile(scenePath)
			if err != nil {
				return fmt.Errorf("failed to read scene %s: %v", scenePath, err)
			}
			sceneData := &SceneData{}
			if err := json.Unmarshal(data, sceneData); err != nil {
				return fmt.Errorf("failed to parse scene %s: %v", scenePath, err)
			}
			allSceneData = append(allSceneData, sceneData)
			if i == 0 {
				primaryScene = sceneData
			}
			logToConsole(fmt.Sprintf("Scene %s: %d models, %d lights", filepath.Base(scenePath), len(sceneData.Models), len(sceneData.Lights)), "info")
		}
	}

	for _, platform := range config.Platforms {
		exportStatus = fmt.Sprintf("Preparing %s build...", platform)
		exportProgress = currentStep / totalSteps
		currentStep++

		outputName := config.GameName
		ext := ""
		if platform == "windows" {
			ext = ".exe"
		}
		outputName += ext

		outputPath := filepath.Join(config.OutputDir, platform, outputName)
		outputDir := filepath.Dir(outputPath)

		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}

		// Generate runtime code based on scene
		exportStatus = fmt.Sprintf("Generating runtime for %s...", platform)
		exportProgress = currentStep / totalSteps
		currentStep++

		if err := generateAndBuildRuntime(platform, outputPath, primaryScene); err != nil {
			return fmt.Errorf("build failed for %s: %v", platform, err)
		}

		// Copy assets for all scenes
		exportStatus = fmt.Sprintf("Copying assets for %s...", platform)
		exportProgress = currentStep / totalSteps
		currentStep++

		if config.IncludeAssets && len(allSceneData) > 0 {
			for i, sceneData := range allSceneData {
				scenePath := config.ScenePaths[i]
				if err := copySceneAssets(scenePath, outputDir, sceneData); err != nil {
					logToConsole(fmt.Sprintf("Warning: Asset copy issues for %s: %v", filepath.Base(scenePath), err), "warning")
				}
			}
		}

		currentStep++
	}

	exportProgress = 1.0
	return nil
}

func generateAndBuildRuntime(platform, outputPath string, scene *SceneData) error {
	modulePath := getModulePath()
	logToConsole(fmt.Sprintf("Module path: %s", modulePath), "info")

	// Create runtime directory
	runtimeDir := filepath.Join(modulePath, "runtime")
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return fmt.Errorf("failed to create runtime dir: %v", err)
	}

	// Copy project scripts to runtime directory if they exist
	projectScripts := copyProjectScriptsToRuntime(runtimeDir)

	// Generate runtime code based on scene data
	runtimeCode := generateRuntimeCode(scene)

	// If we have project scripts, add import for them
	if len(projectScripts) > 0 {
		runtimeCode = addScriptImportToRuntime(runtimeCode, projectScripts)
	}

	mainFile := filepath.Join(runtimeDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(runtimeCode), 0644); err != nil {
		return fmt.Errorf("failed to write main.go: %v", err)
	}

	logToConsole("Generated runtime/main.go", "info")
	logToConsole(fmt.Sprintf("Building for %s...", platform), "info")

	// Build using simple go build - no cross-compilation for now
	// CGO requires native compilation
	args := []string{"build", "-o", outputPath, "./runtime"}

	cmd := exec.Command("go", args...)
	cmd.Dir = modulePath

	// Use current environment, don't try to cross-compile with CGO
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		logToConsole(fmt.Sprintf("Build command: go build -o %s ./runtime", outputPath), "info")
		logToConsole(fmt.Sprintf("Working dir: %s", modulePath), "info")
		logToConsole(fmt.Sprintf("Build error: %s", string(output)), "error")
		return fmt.Errorf("build failed: %v\n%s", err, string(output))
	}

	logToConsole(fmt.Sprintf("Build successful: %s", outputPath), "info")
	return nil
}

// copyProjectScriptsToRuntime copies project scripts to the runtime directory
func copyProjectScriptsToRuntime(runtimeDir string) []string {
	var copiedScripts []string

	if CurrentProject == nil {
		return copiedScripts
	}

	scriptsDir := filepath.Join(CurrentProject.Path, "resources", "scripts")
	if _, err := os.Stat(scriptsDir); os.IsNotExist(err) {
		return copiedScripts
	}

	entries, err := os.ReadDir(scriptsDir)
	if err != nil {
		logToConsole(fmt.Sprintf("Warning: Could not read scripts directory: %v", err), "warning")
		return copiedScripts
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".go") {
			continue
		}

		srcPath := filepath.Join(scriptsDir, name)
		dstPath := filepath.Join(runtimeDir, name)

		// Read and modify the script to change package to main
		content, err := os.ReadFile(srcPath)
		if err != nil {
			logToConsole(fmt.Sprintf("Warning: Could not read script %s: %v", name, err), "warning")
			continue
		}

		// Replace package declaration with package main
		modifiedContent := strings.Replace(string(content), "package scripts", "package main", 1)

		if err := os.WriteFile(dstPath, []byte(modifiedContent), 0644); err != nil {
			logToConsole(fmt.Sprintf("Warning: Could not copy script %s: %v", name, err), "warning")
			continue
		}

		copiedScripts = append(copiedScripts, strings.TrimSuffix(name, ".go"))
		logToConsole(fmt.Sprintf("Included script: %s", name), "info")
	}

	return copiedScripts
}

// addScriptImportToRuntime modifies runtime code to include script initialization
func addScriptImportToRuntime(code string, scripts []string) string {
	// Scripts are now in the same package (main), so they auto-register via init()
	// We just need to ensure behaviour package is imported
	if !strings.Contains(code, `"Gopher3D/internal/behaviour"`) {
		code = strings.Replace(code,
			`"Gopher3D/internal/renderer"`,
			`"Gopher3D/internal/behaviour"
	"Gopher3D/internal/renderer"`, 1)
	}

	logToConsole(fmt.Sprintf("Scripts will auto-register: %v", scripts), "info")
	return code
}

func copySceneAssets(scenePath string, outputDir string, scene *SceneData) error {
	assetsDir := filepath.Join(outputDir, "assets")
	meshesDir := filepath.Join(assetsDir, "meshes")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(meshesDir, 0755); err != nil {
		return err
	}

	sceneDir := filepath.Dir(scenePath)

	// Get renderer to access actual model data
	openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
	if !ok {
		return fmt.Errorf("could not get renderer")
	}

	// Create enhanced scene data with mesh references
	exportScene := *scene // Copy the scene
	exportScene.Models = make([]SceneModel, 0, len(scene.Models))

	// Process each model
	for i, sceneModel := range scene.Models {
		exportModel := sceneModel

		if sceneModel.Path != "" {
			// File-based model - copy the file
			srcPath := sceneModel.Path
			if !filepath.IsAbs(srcPath) {
				srcPath = filepath.Join(sceneDir, srcPath)
			}

			relPath := filepath.Base(sceneModel.Path)
			dstPath := filepath.Join(assetsDir, relPath)

			if err := copyFile(srcPath, dstPath); err != nil {
				logToConsole(fmt.Sprintf("Warning: Could not copy model %s: %v", sceneModel.Name, err), "warning")
			} else {
				copyAssociatedFiles(srcPath, assetsDir)
			}
		} else {
			// Procedural model (voxel, primitive) - serialize mesh data
			// Find the actual model in the renderer
			models := openglRenderer.GetModels()
			var actualModel *renderer.Model
			for _, m := range models {
				if m.Name == sceneModel.Name {
					actualModel = m
					break
				}
			}

			if actualModel != nil {
				// Serialize mesh to binary file
				meshFilename := fmt.Sprintf("mesh_%d_%s.gmesh", i, sanitizeFilename(sceneModel.Name))
				meshPath := filepath.Join(meshesDir, meshFilename)

				mesh := renderer.SerializeMesh(actualModel)
				meshData, err := renderer.EncodeMeshBinary(mesh)
				if err != nil {
					logToConsole(fmt.Sprintf("Warning: Could not serialize mesh %s: %v", sceneModel.Name, err), "warning")
				} else {
					if err := os.WriteFile(meshPath, meshData, 0644); err != nil {
						logToConsole(fmt.Sprintf("Warning: Could not write mesh %s: %v", sceneModel.Name, err), "warning")
					} else {
						// Update scene model to reference the mesh file
						exportModel.MeshDataFile = filepath.Join("meshes", meshFilename)
						logToConsole(fmt.Sprintf("Serialized mesh: %s (%d bytes)", meshFilename, len(meshData)), "info")
					}
				}
			}
		}

		exportScene.Models = append(exportScene.Models, exportModel)
	}

	// Serialize water if present
	if scene.Water != nil && activeWaterSim != nil && activeWaterSim.model != nil {
		meshFilename := "water_mesh.gmesh"
		meshPath := filepath.Join(meshesDir, meshFilename)

		mesh := renderer.SerializeMesh(activeWaterSim.model)
		meshData, err := renderer.EncodeMeshBinary(mesh)
		if err != nil {
			logToConsole(fmt.Sprintf("Warning: Could not serialize water mesh: %v", err), "warning")
		} else {
			if err := os.WriteFile(meshPath, meshData, 0644); err != nil {
				logToConsole(fmt.Sprintf("Warning: Could not write water mesh: %v", err), "warning")
			} else {
				exportScene.Water.MeshDataFile = filepath.Join("meshes", meshFilename)
				logToConsole(fmt.Sprintf("Serialized water mesh (%d bytes)", len(meshData)), "info")
			}
		}
	}

	// Write the updated scene file
	sceneDest := filepath.Join(assetsDir, "scene.json")
	sceneJSON, err := json.MarshalIndent(exportScene, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize scene: %v", err)
	}
	if err := os.WriteFile(sceneDest, sceneJSON, 0644); err != nil {
		return fmt.Errorf("failed to write scene: %v", err)
	}

	// Copy skybox if present
	if scene.Skybox != nil && scene.Skybox.ImagePath != "" {
		srcPath := scene.Skybox.ImagePath
		if !filepath.IsAbs(srcPath) {
			srcPath = filepath.Join(sceneDir, srcPath)
		}
		dstPath := filepath.Join(assetsDir, filepath.Base(scene.Skybox.ImagePath))
		copyFile(srcPath, dstPath)
	}

	logToConsole(fmt.Sprintf("Assets copied to %s", assetsDir), "info")
	return nil
}

// sanitizeFilename removes invalid characters from filename
func sanitizeFilename(name string) string {
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	return result
}

func copyAssociatedFiles(modelPath, assetsDir string) {
	// Look for common associated files (textures, materials)
	dir := filepath.Dir(modelPath)
	base := strings.TrimSuffix(filepath.Base(modelPath), filepath.Ext(modelPath))

	// Common texture extensions
	texExts := []string{".png", ".jpg", ".jpeg", ".tga", ".bmp"}
	// Common material extensions
	matExts := []string{".mtl"}

	for _, ext := range append(texExts, matExts...) {
		srcPath := filepath.Join(dir, base+ext)
		if _, err := os.Stat(srcPath); err == nil {
			dstPath := filepath.Join(assetsDir, base+ext)
			copyFile(srcPath, dstPath)
		}
	}

	// Also check for textures directory
	texDir := filepath.Join(dir, "textures")
	if info, err := os.Stat(texDir); err == nil && info.IsDir() {
		copyDir(texDir, filepath.Join(assetsDir, "textures"))
	}
}

func generateRuntimeCode(scene *SceneData) string {
	// Determine what features are needed based on scene
	hasSkybox := scene != nil && scene.Skybox != nil
	hasComponents := false

	if scene != nil {
		for _, m := range scene.Models {
			if len(m.Components) > 0 {
				hasComponents = true
				break
			}
		}
	}

	// Build imports based on what's needed
	imports := []string{
		`"Gopher3D/internal/engine"`,
		`"Gopher3D/internal/loader"`,
		`"Gopher3D/internal/renderer"`,
		`"encoding/json"`,
		`"fmt"`,
		`"os"`,
		`"path/filepath"`,
		`"runtime"`,
		`mgl "github.com/go-gl/mathgl/mgl32"`,
	}

	if hasComponents {
		imports = append(imports, `"Gopher3D/internal/behaviour"`)
	}

	code := `package main

import (
` + strings.Join(imports, "\n\t") + `
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
`
	if hasComponents {
		code += `		if sceneReady {
			behaviour.GlobalBehaviourManager.UpdateAll()
		}
`
	}
	code += `	})

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
`
	if hasComponents {
		code += `
		// Load components
		if len(m.Components) > 0 {
			gameObj := behaviour.NewGameObject(m.Name)
			gameObj.SetModel(model)
			for _, comp := range m.Components {
				script := behaviour.CreateScript(comp.Type)
				if script != nil {
					gameObj.AddComponent(script)
				}
			}
			behaviour.GlobalComponentManager.RegisterGameObject(gameObj)
		}
`
	}
	code += `
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
`
	if hasSkybox {
		code += `
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
`
	}
	code += `
	// Apply rendering configuration with safe defaults
	// Always ensure depth testing is enabled for proper 3D rendering
	renderer.DepthTestEnabled = true
	renderer.FrustumCullingEnabled = false
	renderer.FaceCullingEnabled = false
	renderer.Debug = false
	
	if scene.Rendering != nil {
		r.EnableBloom = scene.Rendering.Bloom
		r.EnableFXAA = scene.Rendering.FXAA
		// Only override depth test if explicitly set in scene
		if scene.Rendering.DepthTest {
			renderer.DepthTestEnabled = true
		}
		renderer.FaceCullingEnabled = scene.Rendering.FaceCulling
		renderer.Debug = scene.Rendering.Wireframe
	}

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
	GameObjects []SceneGameObject     ` + "`json:\"game_objects,omitempty\"`" + `
	Models      []SceneModel          ` + "`json:\"models,omitempty\"`" + `
	Lights      []SceneLight          ` + "`json:\"lights,omitempty\"`" + `
	Camera      *SceneCamera          ` + "`json:\"camera,omitempty\"`" + `
	Cameras     []SceneCamera         ` + "`json:\"cameras,omitempty\"`" + `
	Water       *SceneWater           ` + "`json:\"water,omitempty\"`" + `
	Skybox      *SceneSkybox          ` + "`json:\"skybox,omitempty\"`" + `
	Rendering   *SceneRenderingConfig ` + "`json:\"rendering,omitempty\"`" + `
}

type SceneGameObject struct {
	Name       string           ` + "`json:\"name\"`" + `
	Tag        string           ` + "`json:\"tag,omitempty\"`" + `
	Active     bool             ` + "`json:\"active\"`" + `
	Position   [3]float32       ` + "`json:\"position\"`" + `
	Rotation   [3]float32       ` + "`json:\"rotation\"`" + `
	Scale      [3]float32       ` + "`json:\"scale\"`" + `
	Components []SceneComponent ` + "`json:\"components,omitempty\"`" + `
}

type SceneCamera struct {
	Name        string     ` + "`json:\"name\"`" + `
	Position    [3]float32 ` + "`json:\"position\"`" + `
	Rotation    [3]float32 ` + "`json:\"rotation\"`" + `
	Speed       float32    ` + "`json:\"speed\"`" + `
	FOV         float32    ` + "`json:\"fov\"`" + `
	Near        float32    ` + "`json:\"near,omitempty\"`" + `
	Far         float32    ` + "`json:\"far,omitempty\"`" + `
	InvertMouse bool       ` + "`json:\"invert_mouse\"`" + `
	IsActive    bool       ` + "`json:\"is_active\"`" + `
}

type SceneModel struct {
	Name          string           ` + "`json:\"name\"`" + `
	Path          string           ` + "`json:\"path,omitempty\"`" + `
	MeshDataFile  string           ` + "`json:\"mesh_data_file,omitempty\"`" + `
	Position      [3]float32       ` + "`json:\"position\"`" + `
	Scale         [3]float32       ` + "`json:\"scale\"`" + `
	Rotation      [3]float32       ` + "`json:\"rotation\"`" + `
	DiffuseColor  [3]float32       ` + "`json:\"diffuse_color\"`" + `
	SpecularColor [3]float32       ` + "`json:\"specular_color\"`" + `
	Shininess     float32          ` + "`json:\"shininess\"`" + `
	Metallic      float32          ` + "`json:\"metallic\"`" + `
	Roughness     float32          ` + "`json:\"roughness\"`" + `
	Alpha         float32          ` + "`json:\"alpha\"`" + `
	Components    []SceneComponent ` + "`json:\"components,omitempty\"`" + `
}

type SceneComponent struct {
	Type       string                 ` + "`json:\"type\"`" + `
	Category   string                 ` + "`json:\"category\"`" + `
	Properties map[string]interface{} ` + "`json:\"properties,omitempty\"`" + `
}

type SceneLight struct {
	Name            string     ` + "`json:\"name\"`" + `
	Mode            string     ` + "`json:\"mode\"`" + `
	Position        [3]float32 ` + "`json:\"position\"`" + `
	Direction       [3]float32 ` + "`json:\"direction\"`" + `
	Color           [3]float32 ` + "`json:\"color\"`" + `
	Intensity       float32    ` + "`json:\"intensity\"`" + `
	AmbientStrength float32    ` + "`json:\"ambient_strength\"`" + `
}

type SceneWater struct {
	OceanSize           float32    ` + "`json:\"ocean_size\"`" + `
	BaseAmplitude       float32    ` + "`json:\"base_amplitude\"`" + `
	WaterColor          [3]float32 ` + "`json:\"water_color\"`" + `
	Transparency        float32    ` + "`json:\"transparency\"`" + `
	WaveSpeedMultiplier float32    ` + "`json:\"wave_speed_multiplier\"`" + `
	Position            [3]float32 ` + "`json:\"position\"`" + `
	FoamEnabled         bool       ` + "`json:\"foam_enabled\"`" + `
	FoamIntensity       float32    ` + "`json:\"foam_intensity\"`" + `
	CausticsEnabled     bool       ` + "`json:\"caustics_enabled\"`" + `
	CausticsIntensity   float32    ` + "`json:\"caustics_intensity\"`" + `
	CausticsScale       float32    ` + "`json:\"caustics_scale\"`" + `
	SpecularIntensity   float32    ` + "`json:\"specular_intensity\"`" + `
	NormalStrength      float32    ` + "`json:\"normal_strength\"`" + `
	DistortionStrength  float32    ` + "`json:\"distortion_strength\"`" + `
	ShadowStrength      float32    ` + "`json:\"shadow_strength\"`" + `
	MeshDataFile        string     ` + "`json:\"mesh_data_file,omitempty\"`" + `
}

type SceneSkybox struct {
	Type      string     ` + "`json:\"type\"`" + `
	ImagePath string     ` + "`json:\"image_path\"`" + `
	Color     [3]float32 ` + "`json:\"color\"`" + `
}

type SceneRenderingConfig struct {
	Bloom       bool       ` + "`json:\"bloom\"`" + `
	FXAA        bool       ` + "`json:\"fxaa\"`" + `
	DepthTest   bool       ` + "`json:\"depth_test\"`" + `
	FaceCulling bool       ` + "`json:\"face_culling\"`" + `
	Wireframe   bool       ` + "`json:\"wireframe\"`" + `
	SkyboxColor [3]float32 ` + "`json:\"skybox_color\"`" + `
}
`
	return code
}

func getModulePath() string {
	// Try to find module root by looking for go.mod
	wd, _ := os.Getwd()

	// Walk up to find go.mod
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return wd
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}
