package renderer

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

// Ray represents a ray in 3D space
type Ray struct {
	Origin    mgl32.Vec3
	Direction mgl32.Vec3
}

// RayIntersectSphere tests if a ray intersects a sphere
// Returns: (intersected, distance, intersection point)
func RayIntersectSphere(ray Ray, sphereCenter mgl32.Vec3, radius float32) (bool, float32, mgl32.Vec3) {
	// Calculate vector from ray origin to sphere center
	oc := ray.Origin.Sub(sphereCenter)
	
	// Calculate coefficients for quadratic equation
	a := ray.Direction.Dot(ray.Direction)
	b := 2.0 * oc.Dot(ray.Direction)
	c := oc.Dot(oc) - radius*radius
	
	// Calculate discriminant
	discriminant := b*b - 4*a*c
	
	// No intersection if discriminant is negative
	if discriminant < 0 {
		return false, 0, mgl32.Vec3{}
	}
	
	// Calculate intersection distance
	sqrtDisc := float32(math.Sqrt(float64(discriminant)))
	t1 := (-b - sqrtDisc) / (2 * a)
	t2 := (-b + sqrtDisc) / (2 * a)
	
	// Return the closest intersection (smallest positive t)
	var t float32
	if t1 > 0 && t2 > 0 {
		if t1 < t2 {
			t = t1
		} else {
			t = t2
		}
	} else if t1 > 0 {
		t = t1
	} else if t2 > 0 {
		t = t2
	} else {
		// Both intersections are behind the ray origin
		return false, 0, mgl32.Vec3{}
	}
	
	// Calculate intersection point
	intersectionPoint := ray.Origin.Add(ray.Direction.Mul(t))
	
	return true, t, intersectionPoint
}

// RayIntersectModel tests if a ray intersects a model using bounding sphere
// Returns: (intersected, distance, intersection point)
func RayIntersectModel(ray Ray, model *Model) (bool, float32, mgl32.Vec3) {
	// Use bounding sphere for quick intersection test
	return RayIntersectSphere(ray, model.BoundingSphereCenter, model.BoundingSphereRadius)
}

// RayIntersectTriangle tests if a ray intersects a triangle
// Returns: (intersected, distance, intersection point)
// Uses Möller-Trumbore algorithm
func RayIntersectTriangle(ray Ray, v0, v1, v2 mgl32.Vec3) (bool, float32, mgl32.Vec3) {
	const epsilon = 0.0000001
	
	edge1 := v1.Sub(v0)
	edge2 := v2.Sub(v0)
	h := ray.Direction.Cross(edge2)
	a := edge1.Dot(h)
	
	if a > -epsilon && a < epsilon {
		return false, 0, mgl32.Vec3{} // Ray is parallel to triangle
	}
	
	f := 1.0 / a
	s := ray.Origin.Sub(v0)
	u := f * s.Dot(h)
	
	if u < 0.0 || u > 1.0 {
		return false, 0, mgl32.Vec3{}
	}
	
	q := s.Cross(edge1)
	v := f * ray.Direction.Dot(q)
	
	if v < 0.0 || u+v > 1.0 {
		return false, 0, mgl32.Vec3{}
	}
	
	// Calculate t to find intersection point
	t := f * edge2.Dot(q)
	
	if t > epsilon {
		// Intersection found
		intersectionPoint := ray.Origin.Add(ray.Direction.Mul(t))
		return true, t, intersectionPoint
	}
	
	return false, 0, mgl32.Vec3{} // Line intersection but not ray intersection
}

// ScreenToRay converts a screen position to a world space ray
func ScreenToRay(camera Camera, screenX, screenY float32, windowWidth, windowHeight int) Ray {
	// Normalize screen coordinates to NDC (-1 to 1)
	ndcX := (2.0*screenX/float32(windowWidth) - 1.0) * camera.AspectRatio
	ndcY := 1.0 - 2.0*screenY/float32(windowHeight)
	
	// Create ray direction in clip space
	clipCoords := mgl32.Vec4{ndcX, ndcY, -1.0, 1.0}
	
	// Transform from clip space to eye space
	invProjection := camera.Projection.Inv()
	eyeCoords := invProjection.Mul4x1(clipCoords)
	eyeCoords = mgl32.Vec4{eyeCoords.X(), eyeCoords.Y(), -1.0, 0.0}
	
	// Transform from eye space to world space
	view := mgl32.LookAtV(camera.Position, camera.Position.Add(camera.Front), camera.Up)
	invView := view.Inv()
	worldDir := invView.Mul4x1(eyeCoords).Vec3().Normalize()
	
	return Ray{
		Origin:    camera.Position,
		Direction: worldDir,
	}
}

