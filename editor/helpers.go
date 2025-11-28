package main

import (
	"Gopher3D/internal/engine"
	"Gopher3D/internal/renderer"
	"fmt"
	"math"
	"path/filepath"
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

	case "clear":
		consoleLines = []ConsoleEntry{}

	case "grid":
		if len(parts) > 1 {
			openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
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
		openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
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
			logToConsole("Usage: wireframe [on/off]", "warning")
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
			logToConsole("Usage: culling [on/off]", "warning")
		}

	case "delete":
		if len(parts) > 1 {
			modelName := strings.Join(parts[1:], " ")
			openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
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
			openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
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
		openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
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

	default:
		logToConsole(fmt.Sprintf("Unknown command: %s (type 'help' for commands)", command), "error")
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
		distance = maxScale * 3.0 // View from 3x the size
	} else {
		distance *= 2.5 // View from 2.5x the bounding radius
	}

	// Position camera to look at the model
	// Place camera in front and slightly above the model
	targetPos := model.Position
	cameraPos := mgl.Vec3{
		targetPos.X(),
		targetPos.Y() + distance*0.3, // Slightly above
		targetPos.Z() + distance,     // In front
	}

	// Set camera position
	eng.Camera.Position = cameraPos

	// Calculate direction to look at target
	direction := targetPos.Sub(cameraPos).Normalize()

	// Calculate yaw and pitch from direction vector
	eng.Camera.Yaw = float32(math.Atan2(float64(direction.X()), float64(direction.Z()))) * 180.0 / 3.14159
	eng.Camera.Pitch = float32(math.Asin(float64(direction.Y()))) * 180.0 / 3.14159

	// Camera vectors will update automatically on next frame

	logToConsole(fmt.Sprintf("Focused camera on '%s'", model.Name), "info")
}

func loadTextureToSelected(path string) {
	openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
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

func applyDarkTheme() {
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

func applyStyleColors(colors StyleColors) {
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
