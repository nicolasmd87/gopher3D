package renderer

import (
	"fmt"
	"image"
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

		-size, -size, size,
		-size, -size, -size,
		-size, size, -size,
		-size, size, -size,
		-size, size, size,
		-size, -size, size,

		size, -size, -size,
		size, -size, size,
		size, size, size,
		size, size, size,
		size, size, -size,
		size, -size, -size,

		-size, -size, size,
		-size, size, size,
		size, size, size,
		size, size, size,
		size, -size, size,
		-size, -size, size,

		-size, size, -size,
		size, size, -size,
		size, size, size,
		size, size, size,
		-size, size, size,
		-size, size, -size,

		-size, -size, -size,
		-size, -size, size,
		size, -size, -size,
		size, -size, -size,
		-size, -size, size,
		size, -size, size,
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
	// Only render if it's a textured skybox (TextureID != 0)
	// Solid color skyboxes are handled by gl.ClearColor in the renderer
	if s.TextureID == 0 {
		return
	}

	// Use shader
	s.Shader.Use()

	// Remove translation from view matrix (skybox should appear infinite)
	view := camera.GetViewMatrix()
	// Zero out the translation components
	view[12] = 0
	view[13] = 0
	view[14] = 0

	projection := camera.GetProjectionMatrix()

	// Set uniforms
	s.Shader.SetMat4("view", view)
	s.Shader.SetMat4("projection", projection)

	gl.DepthMask(false)
	gl.DepthFunc(gl.LEQUAL)

	// Bind texture
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, s.TextureID)

	// Draw skybox
	gl.BindVertexArray(s.VAO)
	gl.DrawArrays(gl.TRIANGLES, 0, 36)
	gl.BindVertexArray(0)

	// Restore OpenGL state
	gl.DepthMask(true)
	gl.DepthFunc(gl.LESS)
}

func loadSkyboxTexture(filePath string) (uint32, error) {
	fmt.Printf("Loading skybox texture: %s\n", filePath)

	imgFile, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer imgFile.Close()

	img, format, err := image.Decode(imgFile)
	if err != nil {
		return 0, err
	}

	fmt.Printf("Decoded %s image successfully\n", format)

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	maxSize := 512
	if width > maxSize || height > maxSize {
		fmt.Printf("Skybox %dx%d exceeds max %d, rejecting\n", width, height, maxSize)
		return 0, fmt.Errorf("skybox image too large (max %dx%d)", maxSize, maxSize)
	}

	var srcPix []uint8
	var srcStride int

	switch src := img.(type) {
	case *image.RGBA:
		srcPix = src.Pix
		srcStride = src.Stride
	case *image.NRGBA:
		srcPix = src.Pix
		srcStride = src.Stride
	default:
		rgba := image.NewRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rgba.Set(x, y, img.At(x, y))
			}
		}
		srcPix = rgba.Pix
		srcStride = rgba.Stride
	}

	result := make([]byte, width*height*4)
	rowSize := width * 4

	for y := 0; y < height; y++ {
		srcStart := y * srcStride
		dstStart := (height - 1 - y) * rowSize
		copy(result[dstStart:dstStart+rowSize], srcPix[srcStart:srcStart+rowSize])
	}

	fmt.Printf("Uploading texture to GPU (%dx%d)...\n", width, height)

	var textureID uint32
	gl.GenTextures(1, &textureID)
	gl.BindTexture(gl.TEXTURE_2D, textureID)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(result))

	fmt.Printf("Skybox texture loaded successfully! (ID: %d)\n", textureID)
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

		-size, -size, size,
		-size, -size, -size,
		-size, size, -size,
		-size, size, -size,
		-size, size, size,
		-size, -size, size,

		size, -size, -size,
		size, -size, size,
		size, size, size,
		size, size, size,
		size, size, -size,
		size, -size, -size,

		-size, -size, size,
		-size, size, size,
		size, size, size,
		size, size, size,
		size, -size, size,
		-size, -size, size,

		-size, size, -size,
		size, size, -size,
		size, size, size,
		size, size, size,
		-size, size, size,
		-size, size, -size,

		-size, -size, -size,
		-size, -size, size,
		size, -size, -size,
		size, -size, -size,
		-size, -size, size,
		size, -size, size,
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

func SetSkyboxColor(r, g, b float32) {
	SkyboxR = r
	SkyboxG = g
	SkyboxB = b
}

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

func UpdateCurrentSkyboxColor(skybox *Skybox, r, g, b float32) {
	if skybox != nil {
		skybox.UpdateColor(r, g, b)
	}
}

// UpdateColor dynamically updates the skybox color (for solid color skyboxes only)
func (s *Skybox) UpdateColor(r, g, b float32) {
	if s.TextureID == 0 { // Only for solid color skyboxes
		s.Shader.skyColor = mgl32.Vec3{r, g, b}
	}
}

// Cleanup cleans up skybox resources
func (s *Skybox) Cleanup() {
	gl.DeleteVertexArrays(1, &s.VAO)
	gl.DeleteBuffers(1, &s.VBO)
	gl.DeleteTextures(1, &s.TextureID)
}
