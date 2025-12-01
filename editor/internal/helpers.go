package editor

import (
	"Gopher3D/internal/engine"
	"Gopher3D/internal/renderer"
	"fmt"
	"math"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	mgl "github.com/go-gl/mathgl/mgl32"
	"github.com/inkyblackness/imgui-go/v4"
)

func getFileNameFromPath(path string) string {
	// Extract filename from path (cross-platform)
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			name := path[i+1:]
			// Remove extension
			for j := len(name) - 1; j >= 0; j-- {
				if name[j] == '.' {
					return name[:j]
				}
			}
			return name
		}
	}
	return path
}

func logToConsole(message string, msgType string) {
	timestamp := time.Now().Format("15:04:05")
	consoleLines = append(consoleLines, ConsoleEntry{
		Message: fmt.Sprintf("[%s] %s", timestamp, message),
		Type:    msgType,
	})

	// Limit console history
	if len(consoleLines) > maxConsoleLines {
		consoleLines = consoleLines[len(consoleLines)-maxConsoleLines:]
	}
}

func executeConsoleCommand(cmd string) {
	logToConsole("> "+cmd, "command")

	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	command := strings.ToLower(parts[0])

	switch command {
	case "help":
		logToConsole("Available commands:", "info")
		logToConsole("  clear - Clear console", "info")
		logToConsole("  models - List all models in scene", "info")
		logToConsole("  inspect <name> - Show detailed material info", "info")
		logToConsole("  wireframe [on/off] - Toggle wireframe mode", "info")
		logToConsole("  culling [on/off] - Toggle frustum culling", "info")
		logToConsole("  delete <name> - Delete model by name", "info")
		logToConsole("  grid [on/off] - Toggle reference grid visibility", "info")
		logToConsole("  fix-materials - Reset all materials to defaults", "info")
		logToConsole("  sh <cmd> - Execute shell command (PowerShell/bash)", "info")
		logToConsole("  !<cmd> - Shortcut for shell command", "info")

	case "clear":
		consoleLines = []ConsoleEntry{}

	case "grid":
		if len(parts) > 1 {
			openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
			if ok {
				models := openglRenderer.GetModels()
				for _, model := range models {
					if model.Name == "Grid Floor" {
						if parts[1] == "off" {
							model.SetScale(0, 0, 0) // Hide grid
							logToConsole("Reference grid hidden", "info")
						} else if parts[1] == "on" {
							model.SetScale(500, 0.5, 500) // Show grid
							logToConsole("Reference grid visible", "info")
						}
						break
					}
				}
			}
		} else {
			logToConsole("Usage: grid [on/off]", "warning")
		}

	case "models":
		openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
		if ok {
			models := openglRenderer.GetModels()
			logToConsole(fmt.Sprintf("Total models: %d", len(models)), "info")
			for i, model := range models {
				logToConsole(fmt.Sprintf("  %d: %s (pos: %.1f, %.1f, %.1f)",
					i, model.Name, model.Position.X(), model.Position.Y(), model.Position.Z()), "info")
			}
		}

	case "wireframe":
		if len(parts) > 1 {
			if parts[1] == "on" {
				renderer.Debug = true
				logToConsole("Wireframe enabled", "info")
			} else if parts[1] == "off" {
				renderer.Debug = false
				logToConsole("Wireframe disabled", "info")
			}
		} else {
			// Toggle when no argument given
			renderer.Debug = !renderer.Debug
			if renderer.Debug {
				logToConsole("Wireframe enabled", "info")
			} else {
				logToConsole("Wireframe disabled", "info")
			}
		}

	case "culling":
		if len(parts) > 1 {
			if parts[1] == "on" {
				renderer.FrustumCullingEnabled = true
				logToConsole("Frustum culling enabled", "info")
			} else if parts[1] == "off" {
				renderer.FrustumCullingEnabled = false
				logToConsole("Frustum culling disabled", "info")
			}
		} else {
			// Toggle when no argument given
			renderer.FrustumCullingEnabled = !renderer.FrustumCullingEnabled
			if renderer.FrustumCullingEnabled {
				logToConsole("Frustum culling enabled", "info")
			} else {
				logToConsole("Frustum culling disabled", "info")
			}
		}

	case "delete":
		if len(parts) > 1 {
			modelName := strings.Join(parts[1:], " ")
			openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
			if ok {
				models := openglRenderer.GetModels()
				found := false
				for _, model := range models {
					if model.Name == modelName {
						openglRenderer.RemoveModel(model)
						logToConsole(fmt.Sprintf("Deleted model: %s", modelName), "info")
						found = true
						break
					}
				}
				if !found {
					logToConsole(fmt.Sprintf("Model not found: %s", modelName), "error")
				}
			}
		} else {
			logToConsole("Usage: delete <model_name>", "warning")
		}

	case "inspect":
		if len(parts) > 1 {
			modelName := strings.Join(parts[1:], " ")
			openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
			if !ok {
				logToConsole("ERROR: Cannot access renderer", "error")
				return
			}
			models := openglRenderer.GetModels()
			found := false
			for _, model := range models {
				if model.Name == modelName {
					found = true
					logToConsole(fmt.Sprintf("=== Model: %s ===", model.Name), "info")
					logToConsole(fmt.Sprintf("Path: %s", model.SourcePath), "info")
					logToConsole(fmt.Sprintf("Position: (%.2f, %.2f, %.2f)", model.Position.X(), model.Position.Y(), model.Position.Z()), "info")
					logToConsole(fmt.Sprintf("Scale: (%.2f, %.2f, %.2f)", model.Scale.X(), model.Scale.Y(), model.Scale.Z()), "info")
					if model.Material != nil {
						logToConsole("=== Material ===", "info")
						logToConsole(fmt.Sprintf("Diffuse: (%.2f, %.2f, %.2f)", model.Material.DiffuseColor[0], model.Material.DiffuseColor[1], model.Material.DiffuseColor[2]), "info")
						logToConsole(fmt.Sprintf("Specular: (%.2f, %.2f, %.2f)", model.Material.SpecularColor[0], model.Material.SpecularColor[1], model.Material.SpecularColor[2]), "info")
						logToConsole(fmt.Sprintf("Shininess: %.2f", model.Material.Shininess), "info")
						logToConsole(fmt.Sprintf("Metallic: %.2f, Roughness: %.2f", model.Material.Metallic, model.Material.Roughness), "info")
						logToConsole(fmt.Sprintf("Exposure: %.2f (CRITICAL)", model.Material.Exposure), "info")
						logToConsole(fmt.Sprintf("Alpha: %.2f", model.Material.Alpha), "info")
						if model.Material.TexturePath != "" {
							logToConsole(fmt.Sprintf("Texture: %s (ID: %d)", filepath.Base(model.Material.TexturePath), model.Material.TextureID), "info")
						} else {
							logToConsole("Texture: None", "info")
						}
					}
					break
				}
			}
			if !found {
				logToConsole(fmt.Sprintf("Model '%s' not found", modelName), "error")
			}
		} else {
			logToConsole("Usage: inspect <model_name>", "warning")
		}

	case "fix-materials":
		openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
		if !ok {
			logToConsole("ERROR: Cannot access renderer", "error")
			return
		}
		models := openglRenderer.GetModels()
		fixed := 0
		for _, model := range models {
			if model.Material != nil {
				needsFix := false
				if model.Material.Exposure == 0 {
					model.Material.Exposure = 1.0
					needsFix = true
				}
				if model.Material.Alpha == 0 {
					model.Material.Alpha = 1.0
					needsFix = true
				}
				if needsFix {
					model.IsDirty = true
					fixed++
				}
			}
		}
		logToConsole(fmt.Sprintf("Fixed %d models with incorrect material values", fixed), "info")

	case "sh", "shell", "exec", "!":
		// Execute shell command
		if len(parts) > 1 {
			shellCmd := strings.Join(parts[1:], " ")
			executeShellCommand(shellCmd)
		} else {
			logToConsole("Usage: sh <command> or !<command>", "warning")
		}

	default:
		// Check if command starts with ! for shell execution
		if strings.HasPrefix(cmd, "!") {
			shellCmd := strings.TrimPrefix(cmd, "!")
			executeShellCommand(shellCmd)
		} else {
		logToConsole(fmt.Sprintf("Unknown command: %s (type 'help' for commands)", command), "error")
		}
	}
}

func executeShellCommand(cmd string) {
	logToConsole(fmt.Sprintf("Executing: %s", cmd), "info")

	// Use PowerShell on Windows, sh on Unix
	var shellCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		shellCmd = exec.Command("powershell", "-Command", cmd)
	} else {
		shellCmd = exec.Command("sh", "-c", cmd)
	}

	output, err := shellCmd.CombinedOutput()
	if err != nil {
		logToConsole(fmt.Sprintf("Error: %v", err), "error")
	}

	// Log output line by line
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			logToConsole(line, "info")
		}
	}
}

func focusCameraOnModel(model *renderer.Model) {
	// Calculate distance to view the entire model
	// Use bounding sphere radius if available, otherwise estimate from scale
	distance := model.BoundingSphereRadius
	if distance <= 0 {
		// Estimate from scale (use largest dimension)
		maxScale := model.Scale.X()
		if model.Scale.Y() > maxScale {
			maxScale = model.Scale.Y()
		}
		if model.Scale.Z() > maxScale {
			maxScale = model.Scale.Z()
		}
		distance = maxScale * 5.0 // View from 5x the size for better overview
		if distance < 10 {
			distance = 10 // Minimum distance
		}
	} else {
		distance *= 4.0 // View from 4x the bounding radius for better overview
		if distance < 10 {
			distance = 10
		}
	}

	// Position camera to look at the model center
	targetPos := model.Position

	// Place camera in front and slightly above the model
	cameraPos := mgl.Vec3{
		targetPos.X(),
		targetPos.Y() + distance*0.4, // Slightly above
		targetPos.Z() + distance,     // In front
	}

	// Set camera position
	Eng.Camera.Position = cameraPos

	// Calculate direction to look at target
	direction := targetPos.Sub(cameraPos).Normalize()

	// Calculate yaw and pitch from direction vector
	// Yaw: rotation around Y axis (horizontal)
	Eng.Camera.Yaw = float32(math.Atan2(float64(direction.X()), float64(direction.Z()))) * (180.0 / math.Pi)
	// Pitch: rotation around X axis (vertical)
	Eng.Camera.Pitch = float32(math.Asin(float64(direction.Y()))) * (180.0 / math.Pi)

	// Camera vectors will be updated on next frame through the engine's update loop

	logToConsole(fmt.Sprintf("Focused camera on '%s' at distance %.1f", model.Name, distance), "info")
}

func loadTextureToSelected(path string) {
	openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
	if !ok {
		return
	}
	models := openglRenderer.GetModels()
	if selectedType != "model" || selectedModelIndex < 0 || selectedModelIndex >= len(models) {
		logToConsole("Error: No model selected for texture loading", "error")
		return
	}
	model := models[selectedModelIndex]

	logToConsole(fmt.Sprintf("Loading texture '%s' for model '%s'...", filepath.Base(path), model.Name), "info")

	textureID, err := openglRenderer.LoadTexture(path)
	if err != nil {
		logToConsole(fmt.Sprintf("Failed to load texture: %v", err), "error")
		return
	}

	// Apply to main material
	if model.Material != nil {
		model.Material.TextureID = textureID
		model.Material.TexturePath = path
		model.IsDirty = true
	}

	// Apply to all material groups to ensure the whole model gets textured
	for i := range model.MaterialGroups {
		if model.MaterialGroups[i].Material != nil {
			model.MaterialGroups[i].Material.TextureID = textureID
			model.MaterialGroups[i].Material.TexturePath = path
		}
	}

	logToConsole(fmt.Sprintf("Successfully applied texture to %s (ID: %d)", model.Name, textureID), "info")
}

func ApplyDarkTheme() {
	style := imgui.CurrentStyle()

	// Go Cyan color (#00ADD8)
	goCyan := imgui.Vec4{X: 0.0, Y: 0.678, Z: 0.847, W: 1.0}
	goCyanHover := imgui.Vec4{X: 0.0, Y: 0.678, Z: 0.847, W: 0.6}
	goCyanActive := imgui.Vec4{X: 0.0, Y: 0.678, Z: 0.847, W: 0.8}
	goCyanDim := imgui.Vec4{X: 0.0, Y: 0.678, Z: 0.847, W: 0.4}

	// Base colors
	style.SetColor(imgui.StyleColorWindowBg, imgui.Vec4{X: 0.1, Y: 0.1, Z: 0.1, W: 0.95})
	style.SetColor(imgui.StyleColorTitleBg, imgui.Vec4{X: 0.08, Y: 0.08, Z: 0.08, W: 1.0})
	style.SetColor(imgui.StyleColorTitleBgActive, goCyan) // Active window title - Go cyan
	style.SetColor(imgui.StyleColorMenuBarBg, imgui.Vec4{X: 0.14, Y: 0.14, Z: 0.14, W: 1.0})

	// Borders and separators - Go cyan
	style.SetColor(imgui.StyleColorBorder, goCyan)
	style.SetColor(imgui.StyleColorSeparator, goCyan)
	style.SetColor(imgui.StyleColorSeparatorHovered, goCyanActive)
	style.SetColor(imgui.StyleColorSeparatorActive, goCyan)

	// Headers (hierarchy selection) - Go cyan
	style.SetColor(imgui.StyleColorHeader, goCyanDim)
	style.SetColor(imgui.StyleColorHeaderHovered, goCyanHover)
	style.SetColor(imgui.StyleColorHeaderActive, goCyan)

	// Buttons - Go cyan accents
	style.SetColor(imgui.StyleColorButton, imgui.Vec4{X: 0.2, Y: 0.2, Z: 0.2, W: 1.0})
	style.SetColor(imgui.StyleColorButtonHovered, goCyanHover)
	style.SetColor(imgui.StyleColorButtonActive, goCyanActive)

	// Frame backgrounds
	style.SetColor(imgui.StyleColorFrameBg, imgui.Vec4{X: 0.2, Y: 0.2, Z: 0.2, W: 0.54})
	style.SetColor(imgui.StyleColorFrameBgHovered, imgui.Vec4{X: 0.25, Y: 0.25, Z: 0.25, W: 0.78})
	style.SetColor(imgui.StyleColorFrameBgActive, imgui.Vec4{X: 0.3, Y: 0.3, Z: 0.3, W: 0.67})

	// Sliders and grab handles - Go cyan
	style.SetColor(imgui.StyleColorSliderGrab, goCyan)
	style.SetColor(imgui.StyleColorSliderGrabActive, goCyanActive)

	// Tabs - Go cyan
	style.SetColor(imgui.StyleColorTab, imgui.Vec4{X: 0.15, Y: 0.15, Z: 0.15, W: 1.0})
	style.SetColor(imgui.StyleColorTabHovered, goCyanActive)
	style.SetColor(imgui.StyleColorTabActive, goCyan)
	style.SetColor(imgui.StyleColorTabUnfocused, imgui.Vec4{X: 0.12, Y: 0.12, Z: 0.12, W: 1.0})
	style.SetColor(imgui.StyleColorTabUnfocusedActive, goCyanDim)

	// Checkboxes and radio buttons
	style.SetColor(imgui.StyleColorCheckMark, goCyan)

	// Text selection
	style.SetColor(imgui.StyleColorTextSelectedBg, goCyanDim)

	// Increase border thickness for visibility
	style.SetWindowBorderSize(1.5)
	style.SetFrameBorderSize(1.0)
	style.SetWindowRounding(4.0)
	style.SetFrameRounding(2.0)
	style.SetGrabRounding(2.0)
}

func updateFPS() {
	frameCount++
	now := time.Now()

	if now.Sub(fpsUpdateTime) >= time.Second {
		fps = float64(frameCount) / now.Sub(fpsUpdateTime).Seconds()
		frameCount = 0
		fpsUpdateTime = now
	}
}

func getCurrentStyleColors() StyleColors {
	style := imgui.CurrentStyle()
	border := style.Color(imgui.StyleColorBorder)
	titleActive := style.Color(imgui.StyleColorTitleBgActive)
	header := style.Color(imgui.StyleColorHeader)
	buttonHover := style.Color(imgui.StyleColorButtonHovered)

	return StyleColors{
		BorderR:       border.X,
		BorderG:       border.Y,
		BorderB:       border.Z,
		TitleActiveR:  titleActive.X,
		TitleActiveG:  titleActive.Y,
		TitleActiveB:  titleActive.Z,
		HeaderR:       header.X,
		HeaderG:       header.Y,
		HeaderB:       header.Z,
		ButtonHoverR:  buttonHover.X,
		ButtonHoverG:  buttonHover.Y,
		ButtonHoverB:  buttonHover.Z,
		WindowBorderR: windowBorderR,
		WindowBorderG: windowBorderG,
		WindowBorderB: windowBorderB,
	}
}

func ApplyStyleColors(colors StyleColors) {
	if colors.BorderR == 0 && colors.BorderG == 0 && colors.BorderB == 0 &&
		colors.TitleActiveR == 0 && colors.TitleActiveG == 0 && colors.TitleActiveB == 0 {
		return
	}

	style := imgui.CurrentStyle()

	borderColor := imgui.Vec4{X: colors.BorderR, Y: colors.BorderG, Z: colors.BorderB, W: 1.0}
	style.SetColor(imgui.StyleColorBorder, borderColor)
	style.SetColor(imgui.StyleColorSeparator, borderColor)

	titleColor := imgui.Vec4{X: colors.TitleActiveR, Y: colors.TitleActiveG, Z: colors.TitleActiveB, W: 1.0}
	style.SetColor(imgui.StyleColorTitleBgActive, titleColor)

	headerColor := imgui.Vec4{X: colors.HeaderR, Y: colors.HeaderG, Z: colors.HeaderB, W: 0.4}
	style.SetColor(imgui.StyleColorHeader, headerColor)
	style.SetColor(imgui.StyleColorHeaderActive, imgui.Vec4{X: colors.HeaderR, Y: colors.HeaderG, Z: colors.HeaderB, W: 1.0})

	buttonHoverColor := imgui.Vec4{X: colors.ButtonHoverR, Y: colors.ButtonHoverG, Z: colors.ButtonHoverB, W: 0.6}
	style.SetColor(imgui.StyleColorButtonHovered, buttonHoverColor)

	windowBorderR = colors.WindowBorderR
	windowBorderG = colors.WindowBorderG
	windowBorderB = colors.WindowBorderB
	updateWindowBorderColor()
}

func updateWindowBorderColor() {
	engine.SetWindowBorderColor(windowBorderR, windowBorderG, windowBorderB)
}

// quatToEuler converts a quaternion to Euler angles (degrees)
func quatToEuler(q mgl.Quat) mgl.Vec3 {
	// Convert quaternion to euler angles
	w, x, y, z := q.W, q.V[0], q.V[1], q.V[2]

	// Roll (x-axis rotation)
	sinrCosp := 2 * (w*x + y*z)
	cosrCosp := 1 - 2*(x*x+y*y)
	roll := float32(math.Atan2(float64(sinrCosp), float64(cosrCosp)))

	// Pitch (y-axis rotation)
	sinp := 2 * (w*y - z*x)
	var pitch float32
	if math.Abs(float64(sinp)) >= 1 {
		pitch = float32(math.Copysign(math.Pi/2, float64(sinp)))
	} else {
		pitch = float32(math.Asin(float64(sinp)))
	}

	// Yaw (z-axis rotation)
	sinyCosp := 2 * (w*z + x*y)
	cosyCosp := 1 - 2*(y*y+z*z)
	yaw := float32(math.Atan2(float64(sinyCosp), float64(cosyCosp)))

	// Convert to degrees
	return mgl.Vec3{
		roll * 180 / math.Pi,
		pitch * 180 / math.Pi,
		yaw * 180 / math.Pi,
	}
}

// eulerToQuat converts Euler angles (degrees) to a quaternion
func eulerToQuat(euler mgl.Vec3) mgl.Quat {
	// Convert degrees to radians
	roll := euler.X() * math.Pi / 180
	pitch := euler.Y() * math.Pi / 180
	yaw := euler.Z() * math.Pi / 180

	// Create quaternion from euler angles
	cy := float32(math.Cos(float64(yaw) * 0.5))
	sy := float32(math.Sin(float64(yaw) * 0.5))
	cp := float32(math.Cos(float64(pitch) * 0.5))
	sp := float32(math.Sin(float64(pitch) * 0.5))
	cr := float32(math.Cos(float64(roll) * 0.5))
	sr := float32(math.Sin(float64(roll) * 0.5))

	return mgl.Quat{
		W: cr*cp*cy + sr*sp*sy,
		V: mgl.Vec3{
			sr*cp*cy - cr*sp*sy,
			cr*sp*cy + sr*cp*sy,
			cr*cp*sy - sr*sp*cy,
		},
	}
}
