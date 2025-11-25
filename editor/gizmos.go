package main

import (
	"Gopher3D/internal/renderer"
	
	mgl "github.com/go-gl/mathgl/mgl32"
	"github.com/inkyblackness/imgui-go/v4"
)

func renderGizmos() {
	if selectedType != "model" || selectedModelIndex < 0 || eng.Camera == nil {
		return
	}
	
	openglRenderer, ok := eng.GetRenderer().(*renderer.OpenGLRenderer)
	if !ok { return }
	
	models := openglRenderer.GetModels()
	if selectedModelIndex >= len(models) { return }
	model := models[selectedModelIndex]
	
	// Get model position
	pos := model.Position
	
	// Create a transparent full-screen overlay for gizmos
	// This ensures we can draw anywhere on screen on top of everything
	imgui.SetNextWindowPos(imgui.Vec2{X: 0, Y: 0})
	imgui.SetNextWindowSize(imgui.Vec2{X: float32(eng.Width), Y: float32(eng.Height)})
	imgui.SetNextWindowBgAlpha(0.0)
	
	flags := imgui.WindowFlagsNoTitleBar | 
			 imgui.WindowFlagsNoResize | 
			 imgui.WindowFlagsNoMove | 
			 imgui.WindowFlagsNoScrollbar | 
			 imgui.WindowFlagsNoInputs | 
			 imgui.WindowFlagsNoSavedSettings | 
			 imgui.WindowFlagsNoBringToFrontOnFocus

	if imgui.BeginV("##GizmoOverlay", nil, flags) {
		drawList := imgui.WindowDrawList()
		
		// Length of gizmo lines (in world units)
		length := float32(20.0) // Increased size
		
		// Calculate screen points
		origin := worldToScreen(pos)
		xAxis := worldToScreen(pos.Add(mgl.Vec3{length, 0, 0}))
		yAxis := worldToScreen(pos.Add(mgl.Vec3{0, length, 0}))
		zAxis := worldToScreen(pos.Add(mgl.Vec3{0, 0, length}))
		
		// Only draw if origin is in front of camera
		if origin.Z() > 0 {
			// X Axis (Red)
			if xAxis.Z() > 0 {
				drawList.AddLine(imgui.Vec2{X: origin.X(), Y: origin.Y()}, imgui.Vec2{X: xAxis.X(), Y: xAxis.Y()}, imgui.PackedColor(0xFF0000FF)) // Red
				drawList.AddText(imgui.Vec2{X: xAxis.X(), Y: xAxis.Y()}, imgui.PackedColor(0xFF0000FF), "X")
			}
			// Y Axis (Green)
			if yAxis.Z() > 0 {
				drawList.AddLine(imgui.Vec2{X: origin.X(), Y: origin.Y()}, imgui.Vec2{X: yAxis.X(), Y: yAxis.Y()}, imgui.PackedColor(0xFF00FF00)) // Green
				drawList.AddText(imgui.Vec2{X: yAxis.X(), Y: yAxis.Y()}, imgui.PackedColor(0xFF00FF00), "Y")
			}
			// Z Axis (Blue)
			if zAxis.Z() > 0 {
				drawList.AddLine(imgui.Vec2{X: origin.X(), Y: origin.Y()}, imgui.Vec2{X: zAxis.X(), Y: zAxis.Y()}, imgui.PackedColor(0xFFFF0000)) // Blue
				drawList.AddText(imgui.Vec2{X: zAxis.X(), Y: zAxis.Y()}, imgui.PackedColor(0xFFFF0000), "Z")
			}
			
			// Center point
			drawList.AddCircleFilled(imgui.Vec2{X: origin.X(), Y: origin.Y()}, 4.0, imgui.PackedColor(0xFFFFFFFF))
		}
		
		imgui.End()
	}
}

// Project world position to screen coordinates
// Returns Vec3 where X,Y are screen coords and Z is depth (positive = in front)
func worldToScreen(worldPos mgl.Vec3) mgl.Vec3 {
	vp := eng.Camera.GetViewProjection()
	
	// Project
	pos4 := mgl.Vec4{worldPos.X(), worldPos.Y(), worldPos.Z(), 1.0}
	clipPos := vp.Mul4x1(pos4)
	
	// Perspective divide
	if clipPos.W() == 0 {
		return mgl.Vec3{0, 0, -1}
	}
	
	ndc := mgl.Vec3{clipPos.X() / clipPos.W(), clipPos.Y() / clipPos.W(), clipPos.Z() / clipPos.W()}
	
	// Viewport transform
	width := float32(eng.Width)
	height := float32(eng.Height)
	
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
