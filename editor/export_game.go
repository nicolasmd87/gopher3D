package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/sqweek/dialog"
)

var (
	showExportDialog   = false
	exportGameName     = "MyGame"
	exportOutputDir    = ""
	exportPlatforms    = map[string]bool{"windows": true, "linux": false, "darwin": false}
	exportIncludeScene = true
	exportProgress     = float32(0)
	exportStatus       = ""
	exportInProgress   = false
)

type ExportConfig struct {
	GameName      string
	OutputDir     string
	Platforms     []string
	ScenePath     string
	IncludeAssets bool
}

func renderExportDialog() {
	if !showExportDialog {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: float32(eng.Width)/2 - 200, Y: float32(eng.Height)/2 - 150}, imgui.ConditionFirstUseEver, imgui.Vec2{})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 400, Y: 350}, imgui.ConditionFirstUseEver)

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
		imgui.Text("Target Platforms:")

		winChecked := exportPlatforms["windows"]
		if imgui.Checkbox("Windows (x64)", &winChecked) {
			exportPlatforms["windows"] = winChecked
		}

		linuxChecked := exportPlatforms["linux"]
		if imgui.Checkbox("Linux (x64)", &linuxChecked) {
			exportPlatforms["linux"] = linuxChecked
		}

		macChecked := exportPlatforms["darwin"]
		if imgui.Checkbox("macOS (x64)", &macChecked) {
			exportPlatforms["darwin"] = macChecked
		}

		imgui.Spacing()
		imgui.Separator()
		imgui.Text("Options:")
		imgui.Checkbox("Include current scene", &exportIncludeScene)

		imgui.Spacing()
		imgui.Separator()

		if exportInProgress {
			imgui.ProgressBar(exportProgress)
			imgui.Text(exportStatus)
		} else {
			if imgui.Button("Export") {
				go startExport()
			}
			imgui.SameLine()
			if imgui.Button("Cancel") {
				showExportDialog = false
			}
		}
	}
	imgui.End()
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

	platforms := []string{}
	for platform, enabled := range exportPlatforms {
		if enabled {
			platforms = append(platforms, platform)
		}
	}

	if len(platforms) == 0 {
		logToConsole("Export failed: No platforms selected", "error")
		return
	}

	exportInProgress = true
	exportProgress = 0
	exportStatus = "Starting export..."

	config := ExportConfig{
		GameName:      exportGameName,
		OutputDir:     exportOutputDir,
		Platforms:     platforms,
		ScenePath:     currentScenePath,
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
	totalSteps := float32(len(config.Platforms) * 3)
	currentStep := float32(0)

	for _, platform := range config.Platforms {
		exportStatus = fmt.Sprintf("Building for %s...", platform)
		exportProgress = currentStep / totalSteps
		currentStep++

		outputName := config.GameName
		ext := ""
		if platform == "windows" {
			ext = ".exe"
		}
		outputName += "-" + platform + ext

		outputPath := filepath.Join(config.OutputDir, platform, outputName)
		outputDir := filepath.Dir(outputPath)

		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}

		exportStatus = fmt.Sprintf("Compiling for %s...", platform)
		exportProgress = currentStep / totalSteps
		currentStep++

		if err := buildForPlatform(platform, outputPath); err != nil {
			return fmt.Errorf("build failed for %s: %v", platform, err)
		}

		exportStatus = fmt.Sprintf("Copying assets for %s...", platform)
		exportProgress = currentStep / totalSteps
		currentStep++

		if config.IncludeAssets && config.ScenePath != "" {
			assetsDir := filepath.Join(outputDir, "assets")
			if err := os.MkdirAll(assetsDir, 0755); err != nil {
				return fmt.Errorf("failed to create assets directory: %v", err)
			}

			sceneDir := filepath.Dir(config.ScenePath)
			if err := copyDir(sceneDir, assetsDir); err != nil {
				logToConsole(fmt.Sprintf("Warning: Could not copy all assets: %v", err), "warning")
			}

			sceneDest := filepath.Join(assetsDir, filepath.Base(config.ScenePath))
			if err := copyFile(config.ScenePath, sceneDest); err != nil {
				logToConsole(fmt.Sprintf("Warning: Could not copy scene file: %v", err), "warning")
			}
		}
	}

	exportProgress = 1.0
	return nil
}

func buildForPlatform(platform, outputPath string) error {
	goarch := "amd64"
	goos := platform

	env := os.Environ()
	env = append(env, "GOOS="+goos)
	env = append(env, "GOARCH="+goarch)
	env = append(env, "CGO_ENABLED=1")

	if platform != runtime.GOOS {
		logToConsole(fmt.Sprintf("Cross-compilation to %s may require additional setup", platform), "warning")
	}

	runtimeDir := filepath.Join(filepath.Dir(os.Args[0]), "..", "runtime")
	if _, err := os.Stat(runtimeDir); os.IsNotExist(err) {
		runtimeDir = filepath.Join(".", "runtime")
	}

	cmd := exec.Command("go", "build", "-ldflags=-s -w", "-o", outputPath, runtimeDir)
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "runtime") {
			return createStandaloneRuntime(platform, outputPath)
		}
		return fmt.Errorf("%v: %s", err, string(output))
	}

	return nil
}

func createStandaloneRuntime(platform, outputPath string) error {
	runtimeCode := generateRuntimeCode()

	tempDir, err := os.MkdirTemp("", "gopher3d-export-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	mainFile := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(runtimeCode), 0644); err != nil {
		return err
	}

	modFile := filepath.Join(tempDir, "go.mod")
	modContent := `module game

go 1.21

require (
	Gopher3D v0.0.0
)

replace Gopher3D => ` + getModulePath() + `
`
	if err := os.WriteFile(modFile, []byte(modContent), 0644); err != nil {
		return err
	}

	cmd := exec.Command("go", "build", "-ldflags=-s -w", "-o", outputPath, ".")
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(),
		"GOOS="+platform,
		"GOARCH=amd64",
		"CGO_ENABLED=1",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build failed: %v\n%s", err, string(output))
	}

	return nil
}

func generateRuntimeCode() string {
	return `package main

import (
	"Gopher3D/internal/engine"
	"Gopher3D/internal/renderer"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	eng := engine.NewGopher()
	eng.Width = 1280
	eng.Height = 720
	eng.SetRenderer(renderer.NewOpenGLRenderer())

	scenePath := findScene()
	if scenePath != "" {
		loadScene(eng, scenePath)
	}

	eng.Render(0, 0)
}

func findScene() string {
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	paths := []string{
		filepath.Join(exeDir, "assets", "scene.json"),
		filepath.Join(exeDir, "scene.json"),
		"assets/scene.json",
		"scene.json",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func loadScene(eng *engine.Gopher, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Could not load scene: %v\n", err)
		return
	}

	var scene map[string]interface{}
	if err := json.Unmarshal(data, &scene); err != nil {
		fmt.Printf("Could not parse scene: %v\n", err)
		return
	}

	fmt.Printf("Loaded scene from %s\n", path)
}
`
}

func getModulePath() string {
	wd, _ := os.Getwd()
	return wd
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

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
