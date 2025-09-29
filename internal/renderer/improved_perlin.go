package renderer

import (
	"math"
	"math/rand"
	"time"
)

// ImprovedPerlinNoise implements the improved Perlin noise from GPU Gems Chapter 5
// Based on Ken Perlin's 2002 improvements: better interpolation and gradient distribution
type ImprovedPerlinNoise struct {
	perm      [512]int      // Permutation table (doubled for wrapping)
	gradients [12][]float64 // 12 gradient vectors as specified in GPU Gems
}

// NewImprovedPerlinNoise creates a new improved Perlin noise generator
func NewImprovedPerlinNoise(seed int64) *ImprovedPerlinNoise {
	noise := &ImprovedPerlinNoise{}

	// Initialize the 12 gradient vectors from GPU Gems Chapter 5
	// These are the edge centers of a cube, providing even statistical distribution
	noise.gradients = [12][]float64{
		{1, 1, 0}, {-1, 1, 0}, {1, -1, 0}, {-1, -1, 0},
		{1, 0, 1}, {-1, 0, 1}, {1, 0, -1}, {-1, 0, -1},
		{0, 1, 1}, {0, -1, 1}, {0, 1, -1}, {0, -1, -1},
	}

	// Create permutation table
	rng := rand.New(rand.NewSource(seed))

	// Initialize with sequential values
	for i := 0; i < 256; i++ {
		noise.perm[i] = i
	}

	// Shuffle using Fisher-Yates algorithm for better randomness
	for i := 255; i > 0; i-- {
		j := rng.Intn(i + 1)
		noise.perm[i], noise.perm[j] = noise.perm[j], noise.perm[i]
	}

	// Double the permutation table to avoid wrapping
	for i := 0; i < 256; i++ {
		noise.perm[256+i] = noise.perm[i]
	}

	return noise
}

// DefaultImprovedPerlinNoise creates a noise generator with time-based seed
func DefaultImprovedPerlinNoise() *ImprovedPerlinNoise {
	return NewImprovedPerlinNoise(time.Now().UnixNano())
}

// fade implements the improved quintic interpolation from GPU Gems Chapter 5
// 6t^5 - 15t^4 + 10t^3 (removes second-derivative discontinuities)
func fade(t float64) float64 {
	return t * t * t * (t*(t*6-15) + 10)
}

// lerp performs linear interpolation
func lerp(t, a, b float64) float64 {
	return a + t*(b-a)
}

// grad computes the dot product of a gradient vector with the input vector
func (noise *ImprovedPerlinNoise) grad(hash int, x, y, z float64) float64 {
	gradient := noise.gradients[hash%12]
	return gradient[0]*x + gradient[1]*y + gradient[2]*z
}

// Noise3D generates 3D Perlin noise at the given coordinates
func (noise *ImprovedPerlinNoise) Noise3D(x, y, z float64) float64 {
	// Find unit cube that contains point
	X := int(math.Floor(x)) & 255
	Y := int(math.Floor(y)) & 255
	Z := int(math.Floor(z)) & 255

	// Find relative x,y,z of point in cube
	x -= math.Floor(x)
	y -= math.Floor(y)
	z -= math.Floor(z)

	// Compute fade curves for each of x,y,z using improved quintic
	u := fade(x)
	v := fade(y)
	w := fade(z)

	// Hash coordinates of the 8 cube corners
	A := noise.perm[X] + Y
	AA := noise.perm[A] + Z
	AB := noise.perm[A+1] + Z
	B := noise.perm[X+1] + Y
	BA := noise.perm[B] + Z
	BB := noise.perm[B+1] + Z

	// Add blended results from 8 corners of cube
	return lerp(w,
		lerp(v,
			lerp(u,
				noise.grad(noise.perm[AA], x, y, z),
				noise.grad(noise.perm[BA], x-1, y, z)),
			lerp(u,
				noise.grad(noise.perm[AB], x, y-1, z),
				noise.grad(noise.perm[BB], x-1, y-1, z))),
		lerp(v,
			lerp(u,
				noise.grad(noise.perm[AA+1], x, y, z-1),
				noise.grad(noise.perm[BA+1], x-1, y, z-1)),
			lerp(u,
				noise.grad(noise.perm[AB+1], x, y-1, z-1),
				noise.grad(noise.perm[BB+1], x-1, y-1, z-1))))
}

// Noise2D generates 2D Perlin noise (z=0)
func (noise *ImprovedPerlinNoise) Noise2D(x, y float64) float64 {
	return noise.Noise3D(x, y, 0.0)
}

// Turbulence generates turbulence using multiple octaves of noise
// Implementation based on GPU Gems Chapter 5 examples
func (noise *ImprovedPerlinNoise) Turbulence(x, y, z float64, octaves int, persistence float64) float64 {
	value := 0.0
	amplitude := 1.0
	frequency := 1.0
	maxValue := 0.0

	for i := 0; i < octaves; i++ {
		value += noise.Noise3D(x*frequency, y*frequency, z*frequency) * amplitude
		maxValue += amplitude
		amplitude *= persistence
		frequency *= 2.0
	}

	// Normalize to [-1, 1]
	return value / maxValue
}

// Turbulence2D generates 2D turbulence
func (noise *ImprovedPerlinNoise) Turbulence2D(x, y float64, octaves int, persistence float64) float64 {
	return noise.Turbulence(x, y, 0.0, octaves, persistence)
}

// Ridge generates ridged multi-fractal noise (good for mountain ridges)
func (noise *ImprovedPerlinNoise) Ridge(x, y, z float64, octaves int, persistence float64) float64 {
	value := 0.0
	amplitude := 1.0
	frequency := 1.0

	for i := 0; i < octaves; i++ {
		n := noise.Noise3D(x*frequency, y*frequency, z*frequency)
		n = math.Abs(n) // Take absolute value
		n = 1.0 - n     // Invert so valleys become ridges
		n = n * n       // Square for sharper ridges

		value += n * amplitude
		amplitude *= persistence
		frequency *= 2.0
	}

	return value
}

// Marble creates a marble-like pattern using noise distortion
// Based on GPU Gems Chapter 5 marble example
func (noise *ImprovedPerlinNoise) Marble(x, y, z float64, frequency float64) float64 {
	// Create distorted sine wave using turbulence
	distortion := noise.Turbulence(x, y, z, 3, 0.5) * 2.0
	marble := math.Sin((x + distortion) * frequency)

	// Square the result for sharper transitions (like real marble)
	return marble*marble - 0.5
}

// Wood creates a wood grain pattern
func (noise *ImprovedPerlinNoise) Wood(x, y, z float64, rings float64) float64 {
	// Distance from center in XZ plane
	distance := math.Sqrt(x*x + z*z)

	// Add noise for irregular rings
	distortion := noise.Turbulence(x, y, z, 2, 0.3) * 0.5

	// Create ring pattern
	grain := math.Sin((distance + distortion) * rings)

	return (grain + 1.0) * 0.5 // Normalize to [0, 1]
}

// Clouds generates cloud-like patterns
func (noise *ImprovedPerlinNoise) Clouds(x, y, z float64, coverage float64) float64 {
	// Multiple octaves for realistic cloud structure
	base := noise.Turbulence(x*0.5, y*0.5, z*0.5, 4, 0.6)
	detail := noise.Turbulence(x*2.0, y*2.0, z*2.0, 2, 0.3) * 0.25

	clouds := base + detail - coverage

	// Smooth step for soft cloud edges
	if clouds < 0 {
		return 0
	}
	if clouds > 1 {
		return 1
	}

	// Smooth step function for natural falloff
	return clouds * clouds * (3.0 - 2.0*clouds)
}
