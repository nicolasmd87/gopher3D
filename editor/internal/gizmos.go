package editor

import (
	"Gopher3D/internal/behaviour"
	"Gopher3D/internal/renderer"
	"math"

	mgl "github.com/go-gl/mathgl/mgl32"
	"github.com/inkyblackness/imgui-go/v4"
)

// Gizmo state
var (
	gizmoDragging     = false
	gizmoDragAxis     = -1 // 0=X, 1=Y, 2=Z
	gizmoDragStartPos mgl.Vec3
	gizmoLastMousePos imgui.Vec2
)

func RenderGizmos() {
	if Eng.Camera == nil {
		return
	}

	// Get the selected object's position
	var pos mgl.Vec3
	var model *renderer.Model

	if selectedType == "gameobject" && selectedGameObjectIndex >= 0 {
		allGameObjects := behaviour.GlobalComponentManager.GetAllGameObjects()
		if selectedGameObjectIndex < len(allGameObjects) {
			obj := allGameObjects[selectedGameObjectIndex]
			pos = obj.Transform.Position
			if m, ok := obj.GetModel().(*renderer.Model); ok {
				model = m
			}
		} else {
			return
		}
	} else if selectedType == "model" && selectedModelIndex >= 0 {
		openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
	if !ok {
		return
	}
	models := openglRenderer.GetModels()
	if selectedModelIndex >= len(models) {
		return
	}
		model = models[selectedModelIndex]
		pos = model.Position
	} else {
		return
	}

	// Create overlay for gizmo drawing
	imgui.SetNextWindowPos(imgui.Vec2{X: 0, Y: 0})
	imgui.SetNextWindowSize(imgui.Vec2{X: float32(Eng.Width), Y: float32(Eng.Height)})
	imgui.SetNextWindowBgAlpha(0.0)

	flags := imgui.WindowFlagsNoTitleBar |
		imgui.WindowFlagsNoResize |
		imgui.WindowFlagsNoMove |
		imgui.WindowFlagsNoScrollbar |
		imgui.WindowFlagsNoSavedSettings |
		imgui.WindowFlagsNoBringToFrontOnFocus

	if imgui.BeginV("##GizmoOverlay", nil, flags) {
		drawList := imgui.WindowDrawList()

		// Length of gizmo lines (in world units)
		length := float32(20.0)

		// Calculate screen points
		origin := worldToScreen(pos)
		xAxis := worldToScreen(pos.Add(mgl.Vec3{length, 0, 0}))
		yAxis := worldToScreen(pos.Add(mgl.Vec3{0, length, 0}))
		zAxis := worldToScreen(pos.Add(mgl.Vec3{0, 0, length}))

		// Only draw if origin is in front of camera
		if origin.Z() > 0 {
			mousePos := imgui.MousePos()
			mouseDown := imgui.IsMouseDown(0)

			// Check for axis hover/click
			xHover := isNearLine(mousePos, origin, xAxis, 8.0)
			yHover := isNearLine(mousePos, origin, yAxis, 8.0)
			zHover := isNearLine(mousePos, origin, zAxis, 8.0)

			// Handle drag start
			if !gizmoDragging && mouseDown && !imgui.IsWindowHovered() {
				if xHover {
					gizmoDragging = true
					gizmoDragAxis = 0
					gizmoDragStartPos = pos
					gizmoLastMousePos = mousePos
				} else if yHover {
					gizmoDragging = true
					gizmoDragAxis = 1
					gizmoDragStartPos = pos
					gizmoLastMousePos = mousePos
				} else if zHover {
					gizmoDragging = true
					gizmoDragAxis = 2
					gizmoDragStartPos = pos
					gizmoLastMousePos = mousePos
				}
			}

			// Handle dragging
			if gizmoDragging {
				if mouseDown {
					// Calculate movement
					delta := imgui.Vec2{
						X: mousePos.X - gizmoLastMousePos.X,
						Y: mousePos.Y - gizmoLastMousePos.Y,
					}

					// Movement sensitivity based on camera distance
					sensitivity := float32(0.05)
					if model != nil {
						dist := Eng.Camera.Position.Sub(model.Position).Len()
						sensitivity = dist * 0.002
					}

					var movement mgl.Vec3
					switch gizmoDragAxis {
					case 0: // X axis
						movement = mgl.Vec3{delta.X * sensitivity, 0, 0}
					case 1: // Y axis
						movement = mgl.Vec3{0, -delta.Y * sensitivity, 0}
					case 2: // Z axis
						movement = mgl.Vec3{0, 0, -delta.X * sensitivity}
					}

					// Apply movement
					newPos := pos.Add(movement)
					applyGizmoMovement(newPos)

					gizmoLastMousePos = mousePos
				} else {
					// Mouse released
					gizmoDragging = false
					gizmoDragAxis = -1
				}
			}

			// Draw colors (highlight if hovered or dragging)
			xColor := uint32(0xFF0000FF) // Red
			yColor := uint32(0xFF00FF00) // Green
			zColor := uint32(0xFFFF0000) // Blue

			if xHover || gizmoDragAxis == 0 {
				xColor = 0xFF8080FF // Bright red
			}
			if yHover || gizmoDragAxis == 1 {
				yColor = 0xFF80FF80 // Bright green
			}
			if zHover || gizmoDragAxis == 2 {
				zColor = 0xFFFF8080 // Bright blue
			}

			// X Axis (Red)
			if xAxis.Z() > 0 {
				drawList.AddLineV(imgui.Vec2{X: origin.X(), Y: origin.Y()}, imgui.Vec2{X: xAxis.X(), Y: xAxis.Y()}, imgui.PackedColor(xColor), 3.0)
				drawList.AddText(imgui.Vec2{X: xAxis.X() + 5, Y: xAxis.Y()}, imgui.PackedColor(xColor), "X")
			}
			// Y Axis (Green)
			if yAxis.Z() > 0 {
				drawList.AddLineV(imgui.Vec2{X: origin.X(), Y: origin.Y()}, imgui.Vec2{X: yAxis.X(), Y: yAxis.Y()}, imgui.PackedColor(yColor), 3.0)
				drawList.AddText(imgui.Vec2{X: yAxis.X() + 5, Y: yAxis.Y()}, imgui.PackedColor(yColor), "Y")
			}
			// Z Axis (Blue)
			if zAxis.Z() > 0 {
				drawList.AddLineV(imgui.Vec2{X: origin.X(), Y: origin.Y()}, imgui.Vec2{X: zAxis.X(), Y: zAxis.Y()}, imgui.PackedColor(zColor), 3.0)
				drawList.AddText(imgui.Vec2{X: zAxis.X() + 5, Y: zAxis.Y()}, imgui.PackedColor(zColor), "Z")
			}

			// Center point
			drawList.AddCircleFilled(imgui.Vec2{X: origin.X(), Y: origin.Y()}, 5.0, imgui.PackedColor(0xFFFFFFFF))
		}

		imgui.End()
	}
}

// isNearLine checks if a point is near a line segment
func isNearLine(point imgui.Vec2, start, end mgl.Vec3, threshold float32) bool {
	if start.Z() <= 0 || end.Z() <= 0 {
		return false
	}

	// Calculate distance from point to line segment
	lineStart := imgui.Vec2{X: start.X(), Y: start.Y()}
	lineEnd := imgui.Vec2{X: end.X(), Y: end.Y()}

	// Vector from start to end
	dx := lineEnd.X - lineStart.X
	dy := lineEnd.Y - lineStart.Y

	// Length squared
	lenSq := dx*dx + dy*dy
	if lenSq == 0 {
		// Start and end are the same point
		dist := float32(math.Sqrt(float64((point.X-lineStart.X)*(point.X-lineStart.X) + (point.Y-lineStart.Y)*(point.Y-lineStart.Y))))
		return dist < threshold
	}

	// Project point onto line
	t := ((point.X-lineStart.X)*dx + (point.Y-lineStart.Y)*dy) / lenSq
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}

	// Closest point on line
	closestX := lineStart.X + t*dx
	closestY := lineStart.Y + t*dy

	// Distance from point to closest point
	dist := float32(math.Sqrt(float64((point.X-closestX)*(point.X-closestX) + (point.Y-closestY)*(point.Y-closestY))))
	return dist < threshold
}

// applyGizmoMovement applies the new position to the selected object
func applyGizmoMovement(newPos mgl.Vec3) {
	if selectedType == "gameobject" && selectedGameObjectIndex >= 0 {
		allGameObjects := behaviour.GlobalComponentManager.GetAllGameObjects()
		if selectedGameObjectIndex < len(allGameObjects) {
			obj := allGameObjects[selectedGameObjectIndex]
			obj.Transform.SetPosition(newPos)

			// Also update the model if present
			if model, ok := obj.GetModel().(*renderer.Model); ok && model != nil {
				model.SetPosition(newPos.X(), newPos.Y(), newPos.Z())
				model.IsDirty = true
			}
		}
	} else if selectedType == "model" && selectedModelIndex >= 0 {
		openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
		if ok {
			models := openglRenderer.GetModels()
			if selectedModelIndex < len(models) {
				model := models[selectedModelIndex]
				model.SetPosition(newPos.X(), newPos.Y(), newPos.Z())
				model.IsDirty = true
			}
		}
	}
}

// Project world position to screen coordinates
// Returns Vec3 where X,Y are screen coords and Z is depth (positive = in front)
func worldToScreen(worldPos mgl.Vec3) mgl.Vec3 {
	vp := Eng.Camera.GetViewProjection()

	// Project
	pos4 := mgl.Vec4{worldPos.X(), worldPos.Y(), worldPos.Z(), 1.0}
	clipPos := vp.Mul4x1(pos4)

	// Perspective divide
	if clipPos.W() == 0 {
		return mgl.Vec3{0, 0, -1}
	}

	ndc := mgl.Vec3{clipPos.X() / clipPos.W(), clipPos.Y() / clipPos.W(), clipPos.Z() / clipPos.W()}

	// Viewport transform
	width := float32(Eng.Width)
	height := float32(Eng.Height)

	// Map NDC [-1, 1] to Screen [0, Width/Height]
	// ImGui (0,0) is Top-Left. OpenGL NDC (0,0) is Center.
	screenX := (ndc.X() + 1) * 0.5 * width
	screenY := (1 - ndc.Y()) * 0.5 * height // Flip Y: NDC Y+ is Up, Screen Y+ is Down

	// Check if behind camera
	if clipPos.W() < 0 {
		return mgl.Vec3{screenX, screenY, -1}
	}

	return mgl.Vec3{screenX, screenY, 1}
}
