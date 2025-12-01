// camera.go
package renderer

import (
	"math"

	"github.com/xlab/linmath"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

type Camera struct {
	// HOT DATA - Accessed every frame for view/projection calculations
	Position   mgl32.Vec3 // Camera position in world space
	Front      mgl32.Vec3 // Forward direction vector
	Up         mgl32.Vec3 // Up direction vector
	Right      mgl32.Vec3 // Right direction vector
	Projection mgl32.Mat4 // Projection matrix
	Pitch      float32    // Pitch angle (vertical rotation)
	Yaw        float32    // Yaw angle (horizontal rotation)

	// COLD DATA - Configuration and input handling, accessed less frequently
	WorldUp      mgl32.Vec3 // World up vector (usually (0,1,0))
	Speed        float32    // Movement speed
	Sensitivity  float32    // Mouse sensitivity
	Fov          float32    // Field of view
	Near         float32    // Near clipping plane
	Far          float32    // Far clipping plane
	AspectRatio  float32    // Screen aspect ratio
	LastX, LastY float32    // Last mouse position
	InvertMouse  bool       // Invert mouse Y axis
	firstMouse   bool       // First mouse movement flag

	// Identification
	Name     string // Camera name for editor identification
	IsActive bool   // Whether this is the active camera
}

type Plane struct {
	Normal   mgl32.Vec3
	Distance float32
}

type Frustum struct {
	Planes [6]Plane
}

func NewDefaultCamera(height int32, width int32) *Camera {
	camera := Camera{
		Position:    mgl32.Vec3{1, 0, 100},
		Front:       mgl32.Vec3{0, 0, -1},
		Up:          mgl32.Vec3{0, 1, 0},
		WorldUp:     mgl32.Vec3{0, 1, 0},
		Pitch:       0.0,
		Yaw:         -90.0,
		Speed:       70,
		Sensitivity: 0.1,
		Fov:         45.0,
		Near:        0.1,
		// TODO: Setting it form an example doesn't seem to work
		Far:         10000.0,
		LastX:       float32(width) / 2,
		LastY:       float32(height) / 2,
		AspectRatio: float32(height) / float32(width),
		firstMouse:  true,
		InvertMouse: true,
	}
	camera.updateCameraVectors()
	camera.UpdateProjection()
	return &camera
}

func (c *Camera) UpdateProjection() {
	c.Projection = mgl32.Perspective(mgl32.DegToRad(c.Fov), c.AspectRatio, c.Near, c.Far)
}

// Setter methods that automatically update projection
func (c *Camera) SetNear(near float32) {
	c.Near = near
	c.UpdateProjection()
}

func (c *Camera) SetFar(far float32) {
	c.Far = far
	c.UpdateProjection()
}

func (c *Camera) SetFov(fov float32) {
	c.Fov = fov
	c.UpdateProjection()
}

func (c *Camera) SetAspectRatio(aspectRatio float32) {
	c.AspectRatio = aspectRatio
	c.UpdateProjection()
}

func (c *Camera) GetViewProjection() mgl32.Mat4 {
	view := mgl32.LookAtV(c.Position, c.Position.Add(c.Front), c.Up)
	return c.Projection.Mul4(view)
}

func (c *Camera) GetViewMatrix() mgl32.Mat4 {
	return mgl32.LookAtV(c.Position, c.Position.Add(c.Front), c.Up)
}

func (c *Camera) GetProjectionMatrix() mgl32.Mat4 {
	return c.Projection
}

// TODO: THIS IS JUST WHILE I TEST VULKAN INTEGRATION, WE SHOULD NOT BE CONVERTING ANYTHING
//
//	WE NEED TO WORK ON A LINMATH CAMERA IMPLEMENTATION FOR VULKAN
//
// Assume linmath.Mat4x4 is a struct with a flat array of 16 float32s
func convertMGL32Mat4ToLinMathMat4x4(m mgl32.Mat4) linmath.Mat4x4 {
	var viewProjection linmath.Mat4x4
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			viewProjection[i][j] = m[i*4+j]
		}
	}
	return viewProjection
}

func (c *Camera) GetViewProjectionVulkan() linmath.Mat4x4 {
	return convertMGL32Mat4ToLinMathMat4x4(c.GetViewProjection())
}

func (c *Camera) GetViewMatrixVulkan() linmath.Mat4x4 {
	view := mgl32.LookAtV(c.Position, c.Position.Add(c.Front), c.Up)
	return convertMGL32Mat4ToLinMathMat4x4(view)
}

func (c *Camera) ProcessKeyboard(window *glfw.Window, deltaTime float32) {
	// Compute the right vector
	c.Right = c.Front.Cross(c.WorldUp).Normalize()
	baseVelocity := c.Speed * deltaTime

	// If Shift is pressed, multiply the velocity by a factor (e.g., 2.5)
	if window.GetKey(glfw.KeyLeftShift) == glfw.Press || window.GetKey(glfw.KeyRightShift) == glfw.Press {
		baseVelocity *= 2.5
	}

	cameraMoved := false
	if window.GetKey(glfw.KeyW) == glfw.Press {
		c.Position = c.Position.Add(c.Front.Mul(baseVelocity))
		cameraMoved = true
	}
	if window.GetKey(glfw.KeyS) == glfw.Press {
		c.Position = c.Position.Sub(c.Front.Mul(baseVelocity))
		cameraMoved = true
	}
	if window.GetKey(glfw.KeyA) == glfw.Press {
		c.Position = c.Position.Sub(c.Right.Mul(baseVelocity))
		cameraMoved = true
	}
	if window.GetKey(glfw.KeyD) == glfw.Press {
		c.Position = c.Position.Add(c.Right.Mul(baseVelocity))
		cameraMoved = true
	}

	if cameraMoved {
		MarkFrustumDirty()
	}
}

func (c *Camera) ProcessMouseMovement(xoffset, yoffset float32, constrainPitch bool) {
	xoffset *= c.Sensitivity
	yoffset *= c.Sensitivity

	c.Yaw += xoffset

	if c.InvertMouse {
		c.Pitch -= yoffset
	} else {
		c.Pitch += yoffset
	}
	if constrainPitch {
		c.Pitch = mgl32.Clamp(c.Pitch, -89.0, 89.0) // Prevent extreme pitch values
	}
	c.updateCameraVectors()
	MarkFrustumDirty()
}

func (c *Camera) LookAt(target mgl32.Vec3) {
	direction := c.Position.Sub(target).Normalize()
	c.Yaw = float32(math.Atan2(float64(direction.Z()), float64(direction.X())))
	c.Pitch = float32(math.Atan2(float64(direction.Y()), math.Sqrt(float64(direction.X()*direction.X()+direction.Z()*direction.Z()))))
	c.updateCameraVectors()
}

func (c *Camera) updateCameraVectors() {
	yawRad := mgl32.DegToRad(c.Yaw)
	pitchRad := mgl32.DegToRad(c.Pitch)

	front := mgl32.Vec3{
		float32(math.Cos(float64(yawRad)) * math.Cos(float64(pitchRad))),
		float32(math.Sin(float64(pitchRad))),
		float32(math.Sin(float64(yawRad)) * math.Cos(float64(pitchRad))),
	}

	c.Front = front.Normalize()
	c.Right = c.WorldUp.Cross(c.Front).Normalize()
	c.Up = c.Front.Cross(c.Right).Normalize()
}

func (c *Camera) CalculateFrustum() Frustum {
	var frustum Frustum
	vp := c.GetViewProjection()

	// Left Plane
	frustum.Planes[0] = Plane{
		Normal:   mgl32.Vec3{vp[3] + vp[0], vp[7] + vp[4], vp[11] + vp[8]},
		Distance: vp[15] + vp[12],
	}

	// Right Plane
	frustum.Planes[1] = Plane{
		Normal:   mgl32.Vec3{vp[3] - vp[0], vp[7] - vp[4], vp[11] - vp[8]},
		Distance: vp[15] - vp[12],
	}

	// Bottom Plane
	frustum.Planes[2] = Plane{
		Normal:   mgl32.Vec3{vp[3] + vp[1], vp[7] + vp[5], vp[11] + vp[9]},
		Distance: vp[15] + vp[13],
	}

	// Top Plane
	frustum.Planes[3] = Plane{
		Normal:   mgl32.Vec3{vp[3] - vp[1], vp[7] - vp[5], vp[11] - vp[9]},
		Distance: vp[15] - vp[13],
	}

	// Near Plane
	frustum.Planes[4] = Plane{
		Normal:   mgl32.Vec3{vp[3] + vp[2], vp[7] + vp[6], vp[11] + vp[10]},
		Distance: vp[15] + vp[14],
	}

	// Far Plane
	frustum.Planes[5] = Plane{
		Normal:   mgl32.Vec3{vp[3] - vp[2], vp[7] - vp[6], vp[11] - vp[10]},
		Distance: vp[15] - vp[14],
	}

	// Normalize the planes
	for i := 0; i < 6; i++ {
		length := frustum.Planes[i].Normal.Len()
		frustum.Planes[i].Normal = frustum.Planes[i].Normal.Mul(1.0 / length)
		frustum.Planes[i].Distance /= length
	}

	return frustum
}

func (p *Plane) DistanceToPoint(point mgl32.Vec3) float32 {
	return p.Normal.Dot(point) + p.Distance
}

func (f *Frustum) IntersectsSphere(center mgl32.Vec3, radius float32) bool {
	// Check if the sphere intersects with the frustum
	for _, plane := range f.Planes {
		if plane.DistanceToPoint(center) < -radius {
			return false // Sphere is outside the frustum
		}
	}
	return true
}

// MarkFrustumDirty marks the frustum as needing recalculation
func MarkFrustumDirty() {
	// Set the package-level flag in opengl_renderer.go
	SetFrustumDirty()
}

func (c *Camera) ScreenToWorld(screenPos mgl32.Vec2, windowWidth, windowHeight int) mgl32.Vec3 {
	// Step 1: Normalize screen position to NDC (-1 to 1 range)
	ndcX := (2.0*screenPos.X()/float32(windowWidth) - 1.0) * c.AspectRatio
	ndcY := 1.0 - 2.0*screenPos.Y()/float32(windowHeight)

	// Step 2: Create a 4D point in clip space (NDC)
	clipCoords := mgl32.Vec4{ndcX, ndcY, -1.0, 1.0}

	// Step 3: Transform clip space to eye space by inverting the projection matrix
	invProjection := c.Projection.Inv()
	eyeCoords := invProjection.Mul4x1(clipCoords)
	eyeCoords = mgl32.Vec4{eyeCoords.X(), eyeCoords.Y(), -1.0, 0.0}

	// Step 4: Transform from eye space to world space using the view matrix
	view := mgl32.LookAtV(c.Position, c.Position.Add(c.Front), c.Up)
	invView := view.Inv()
	worldCoords := invView.Mul4x1(eyeCoords).Vec3().Normalize()

	// Step 5: Return the world position
	return c.Position.Add(worldCoords.Mul(10.0)) // Adjust scaling if necessary
}
