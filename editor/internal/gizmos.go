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
	var selectedLight *renderer.Light

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
	} else if selectedType == "light" && selectedLightIndex >= 0 {
		openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
		if !ok {
			return
		}
		lights := openglRenderer.GetLights()
		if selectedLightIndex >= len(lights) {
			return
		}
		selectedLight = lights[selectedLightIndex]
		pos = selectedLight.Position
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

			// Handle drag start - check if hovering over any gizmo axis
			if !gizmoDragging && mouseDown && (xHover || yHover || zHover) {
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
				case 0: // X axis - positive screen X = positive world X
					movement = mgl.Vec3{-delta.X * sensitivity, 0, 0}
				case 1: // Y axis - negative screen Y = positive world Y (screen Y is inverted)
					movement = mgl.Vec3{0, -delta.Y * sensitivity, 0}
				case 2: // Z axis - positive screen X = negative world Z (into screen)
					movement = mgl.Vec3{0, 0, delta.X * sensitivity}
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
	} else if selectedType == "light" && selectedLightIndex >= 0 {
		openglRenderer, ok := Eng.GetRenderer().(*renderer.OpenGLRenderer)
		if ok {
			lights := openglRenderer.GetLights()
			if selectedLightIndex < len(lights) {
				light := lights[selectedLightIndex]
				light.Position = newPos
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

// RenderOrientationGizmo draws a small axis indicator in the top-right corner
// showing the current camera orientation (like Unity's scene view gizmo)
func RenderOrientationGizmo() {
	if Eng == nil || Eng.Camera == nil {
		return
	}

	// Window and gizmo dimensions
	gizmoSize := float32(80)
	margin := float32(10)
	windowSize := gizmoSize + margin*2

	// Window position in top-right corner
	windowX := float32(Eng.Width) - windowSize - margin
	windowY := margin // Small margin from top

	// Center of gizmo within the window (in screen coordinates)
	centerX := windowX + windowSize/2
	centerY := windowY + windowSize/2

	// Get camera rotation to transform axes
	view := Eng.Camera.GetViewMatrix()

	// Extract rotation from view matrix (transpose of upper 3x3)
	// The view matrix transforms world to camera space
	// We want to show where world axes point in screen space
	axisLength := gizmoSize * 0.35

	// Transform world axes by view rotation (just rotation, not translation)
	xAxis := mgl.Vec3{view[0], view[4], view[8]}  // World X in camera space
	yAxis := mgl.Vec3{view[1], view[5], view[9]}  // World Y in camera space
	zAxis := mgl.Vec3{view[2], view[6], view[10]} // World Z in camera space

	// Convert to screen space (flip Y because screen Y is down)
	xScreen := imgui.Vec2{X: centerX + xAxis.X()*axisLength, Y: centerY - xAxis.Y()*axisLength}
	yScreen := imgui.Vec2{X: centerX + yAxis.X()*axisLength, Y: centerY - yAxis.Y()*axisLength}
	zScreen := imgui.Vec2{X: centerX + zAxis.X()*axisLength, Y: centerY - zAxis.Y()*axisLength}
	center := imgui.Vec2{X: centerX, Y: centerY}

	// Create overlay window for drawing
	imgui.SetNextWindowPos(imgui.Vec2{X: windowX, Y: windowY})
	imgui.SetNextWindowSize(imgui.Vec2{X: windowSize, Y: windowSize})
	imgui.SetNextWindowBgAlpha(0.3)

	flags := imgui.WindowFlagsNoTitleBar |
		imgui.WindowFlagsNoResize |
		imgui.WindowFlagsNoMove |
		imgui.WindowFlagsNoScrollbar |
		imgui.WindowFlagsNoSavedSettings |
		imgui.WindowFlagsNoInputs |
		imgui.WindowFlagsNoBringToFrontOnFocus

	if imgui.BeginV("##OrientationGizmo", nil, flags) {
		drawList := imgui.WindowDrawList()

		// Draw background circle
		drawList.AddCircleFilled(center, gizmoSize*0.45, imgui.PackedColor(0x40000000))
		drawList.AddCircle(center, gizmoSize*0.45, imgui.PackedColor(0x80FFFFFF))

		// Determine draw order based on Z depth (draw back-to-front)
		type axisInfo struct {
			end   imgui.Vec2
			color uint32
			label string
			depth float32
		}

		axes := []axisInfo{
			{xScreen, 0xFF0000FF, "X", xAxis.Z()}, // Red for X
			{yScreen, 0xFF00FF00, "Y", yAxis.Z()}, // Green for Y
			{zScreen, 0xFFFF0000, "Z", zAxis.Z()}, // Blue for Z
		}

		// Sort by depth (furthest first)
		for i := 0; i < len(axes)-1; i++ {
			for j := i + 1; j < len(axes); j++ {
				if axes[i].depth > axes[j].depth {
					axes[i], axes[j] = axes[j], axes[i]
				}
			}
		}

		// Draw axes (back to front)
		for _, axis := range axes {
			// Draw line
			drawList.AddLineV(center, axis.end, imgui.PackedColor(axis.color), 2.0)
			// Draw endpoint circle
			drawList.AddCircleFilled(axis.end, 6, imgui.PackedColor(axis.color))
			// Draw label
			labelOffset := imgui.Vec2{X: axis.end.X - center.X, Y: axis.end.Y - center.Y}
			labelLen := float32(math.Sqrt(float64(labelOffset.X*labelOffset.X + labelOffset.Y*labelOffset.Y)))
			if labelLen > 0 {
				labelOffset.X = labelOffset.X / labelLen * 12
				labelOffset.Y = labelOffset.Y / labelLen * 12
			}
			drawList.AddText(imgui.Vec2{X: axis.end.X + labelOffset.X - 3, Y: axis.end.Y + labelOffset.Y - 6}, imgui.PackedColor(axis.color), axis.label)
		}

		// Draw center point
		drawList.AddCircleFilled(center, 4, imgui.PackedColor(0xFFFFFFFF))

		imgui.End()
	}
}
