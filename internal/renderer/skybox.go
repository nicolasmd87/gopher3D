package renderer

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// Public skybox variables - easily customizable in examples!
var (
	SkyboxR    float32 = 0.5     // Red component (0.0-1.0)
	SkyboxG    float32 = 0.7     // Green component (0.0-1.0)
	SkyboxB    float32 = 1.0     // Blue component (0.0-1.0) - bright blue sky!
	SkyboxSize float32 = 10000.0 // Skybox size (default 10000 units - very large)
)

type Skybox struct {
	VAO       uint32
	VBO       uint32
	TextureID uint32
	Shader    Shader
}

// CreateSkybox creates a skybox with the specified texture
func CreateSkybox(texturePath string) (*Skybox, error) {
	skybox := &Skybox{}

	// Handle special case for solid color skybox
	if texturePath == "dark_sky" || texturePath == "" {
		// Creating solid color skybox with current RGB values
		return CreateSolidColorSkybox(SkyboxR, SkyboxG, SkyboxB) // Use public variables!
	}

	// Create skybox geometry (configurable size)
	size := SkyboxSize
	vertices := []float32{
		// Positions (cube centered at origin)
		-size, size, -size,
		-size, -size, -size,
		size, -size, -size,
		size, -size, -size,
		size, size, -size,
		-size, size, -size,

		-1000.0, -1000.0, 1000.0,
		-1000.0, -1000.0, -1000.0,
		-1000.0, 1000.0, -1000.0,
		-1000.0, 1000.0, -1000.0,
		-1000.0, 1000.0, 1000.0,
		-1000.0, -1000.0, 1000.0,

		1000.0, -1000.0, -1000.0,
		1000.0, -1000.0, 1000.0,
		1000.0, 1000.0, 1000.0,
		1000.0, 1000.0, 1000.0,
		1000.0, 1000.0, -1000.0,
		1000.0, -1000.0, -1000.0,

		-1000.0, -1000.0, 1000.0,
		-1000.0, 1000.0, 1000.0,
		1000.0, 1000.0, 1000.0,
		1000.0, 1000.0, 1000.0,
		1000.0, -1000.0, 1000.0,
		-1000.0, -1000.0, 1000.0,

		-1000.0, 1000.0, -1000.0,
		1000.0, 1000.0, -1000.0,
		1000.0, 1000.0, 1000.0,
		1000.0, 1000.0, 1000.0,
		-1000.0, 1000.0, 1000.0,
		-1000.0, 1000.0, -1000.0,

		-1000.0, -1000.0, -1000.0,
		-1000.0, -1000.0, 1000.0,
		1000.0, -1000.0, -1000.0,
		1000.0, -1000.0, -1000.0,
		-1000.0, -1000.0, 1000.0,
		1000.0, -1000.0, 1000.0,
	}

	// Create VAO and VBO
	gl.GenVertexArrays(1, &skybox.VAO)
	gl.GenBuffers(1, &skybox.VBO)

	gl.BindVertexArray(skybox.VAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, skybox.VBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

	// Position attribute
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 3*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)

	gl.BindVertexArray(0)

	// Load texture directly for skybox
	textureID, err := loadSkyboxTexture(texturePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load skybox texture %s: %v", texturePath, err)
	}
	skybox.TextureID = textureID

	// Initialize skybox shader
	skybox.Shader = InitSkyboxShader()
	skybox.Shader.Compile()

	return skybox, nil
}

// Render renders the skybox
func (s *Skybox) Render(camera Camera) {
	// Skybox rendering is now handled by the renderer using clear color
	// This method is kept for compatibility but does nothing
}

// loadSkyboxTexture loads a texture specifically for skybox use
func loadSkyboxTexture(filePath string) (uint32, error) {
	imgFile, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer imgFile.Close()

	img, _, err := image.Decode(imgFile)
	if err != nil {
		return 0, err
	}

	rgba := image.NewRGBA(img.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		return 0, fmt.Errorf("unsupported stride")
	}
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)

	var textureID uint32
	gl.GenTextures(1, &textureID)
	gl.BindTexture(gl.TEXTURE_2D, textureID)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(rgba.Rect.Size().X), int32(rgba.Rect.Size().Y), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgba.Pix))

	// Skybox-specific texture parameters
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	return textureID, nil
}

// CreateSolidColorSkybox creates a skybox with a solid color (no texture needed)
func CreateSolidColorSkybox(r, g, b float32) (*Skybox, error) {
	skybox := &Skybox{}
	// Creating solid color skybox

	// Create skybox geometry (configurable size)
	size := SkyboxSize
	vertices := []float32{
		// Positions (cube centered at origin)
		-size, size, -size,
		-size, -size, -size,
		size, -size, -size,
		size, -size, -size,
		size, size, -size,
		-size, size, -size,

		-1000.0, -1000.0, 1000.0,
		-1000.0, -1000.0, -1000.0,
		-1000.0, 1000.0, -1000.0,
		-1000.0, 1000.0, -1000.0,
		-1000.0, 1000.0, 1000.0,
		-1000.0, -1000.0, 1000.0,

		1000.0, -1000.0, -1000.0,
		1000.0, -1000.0, 1000.0,
		1000.0, 1000.0, 1000.0,
		1000.0, 1000.0, 1000.0,
		1000.0, 1000.0, -1000.0,
		1000.0, -1000.0, -1000.0,

		-1000.0, -1000.0, 1000.0,
		-1000.0, 1000.0, 1000.0,
		1000.0, 1000.0, 1000.0,
		1000.0, 1000.0, 1000.0,
		1000.0, -1000.0, 1000.0,
		-1000.0, -1000.0, 1000.0,

		-1000.0, 1000.0, -1000.0,
		1000.0, 1000.0, -1000.0,
		1000.0, 1000.0, 1000.0,
		1000.0, 1000.0, 1000.0,
		-1000.0, 1000.0, 1000.0,
		-1000.0, 1000.0, -1000.0,

		-1000.0, -1000.0, -1000.0,
		-1000.0, -1000.0, 1000.0,
		1000.0, -1000.0, -1000.0,
		1000.0, -1000.0, -1000.0,
		-1000.0, -1000.0, 1000.0,
		1000.0, -1000.0, 1000.0,
	}

	// Create VAO and VBO
	gl.GenVertexArrays(1, &skybox.VAO)
	gl.GenBuffers(1, &skybox.VBO)

	gl.BindVertexArray(skybox.VAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, skybox.VBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

	// Position attribute
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 3*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)

	gl.BindVertexArray(0)

	// No texture needed for solid color
	skybox.TextureID = 0

	// Initialize solid color skybox shader
	skybox.Shader = InitSolidColorSkyboxShader(r, g, b)
	skybox.Shader.Compile()

	return skybox, nil
}

// SetSkyboxColor easily sets custom skybox color (RGB values 0.0-1.0)
// Call this BEFORE engine.SetSkybox() in your examples!
// Example: renderer.SetSkyboxColor(1.0, 0.5, 0.2) // Orange sky
func SetSkyboxColor(r, g, b float32) {
	SkyboxR = r
	SkyboxG = g
	SkyboxB = b
}

// SetSkyboxSize sets the skybox size (default is 1000.0)
// Call this BEFORE engine.SetSkybox() in your examples!
// Example: renderer.SetSkyboxSize(5000.0) // Large skybox for big worlds
func SetSkyboxSize(size float32) {
	SkyboxSize = size
}

// Preset color functions for convenience - call before SetSkybox()
func SetSkyboxDay() {
	SetSkyboxColor(0.5, 0.7, 1.0) // Bright day sky
}

func SetSkyboxSunset() {
	SetSkyboxColor(1.0, 0.6, 0.3) // Orange sunset
}

func SetSkyboxNight() {
	SetSkyboxColor(0.1, 0.1, 0.3) // Dark night
}

func SetSkyboxBrightBlue() {
	SetSkyboxColor(0.3, 0.6, 1.0) // Very bright blue
}

// UpdateCurrentSkyboxColor updates the color of an existing skybox (if solid color)
// This is useful for changing colors after the skybox is already created
func UpdateCurrentSkyboxColor(skybox *Skybox, r, g, b float32) {
	if skybox != nil {
		skybox.UpdateColor(r, g, b)
	}
}

// UpdateColor dynamically updates the skybox color (for solid color skyboxes only)
func (s *Skybox) UpdateColor(r, g, b float32) {
	if s.TextureID == 0 { // Only for solid color skyboxes
		s.Shader.skyColor = mgl32.Vec3{r, g, b}
		// fmt.Printf("DEBUG: Updated skybox color to RGB(%.2f, %.2f, %.2f)\n", r, g, b)
	}
}

// Cleanup cleans up skybox resources
func (s *Skybox) Cleanup() {
	gl.DeleteVertexArrays(1, &s.VAO)
	gl.DeleteBuffers(1, &s.VBO)
	gl.DeleteTextures(1, &s.TextureID)
}
