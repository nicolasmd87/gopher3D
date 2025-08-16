package renderer

import (
	"Gopher3D/internal/logger"
	"fmt"
	"image"
	"image/draw"
	"os"
	"strings"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"go.uber.org/zap"
)

var currentTextureID uint32 = ^uint32(0) // Initialize with an invalid value
var frustum Frustum

type OpenGLRenderer struct {
	defaultShader        Shader
	Models               []*Model
	instanceVBO          uint32  // Buffer for instance model matrices
	currentShaderProgram uint32  // Track currently bound shader to avoid unnecessary switches
	skybox               *Skybox // Optional skybox
}

func (rend *OpenGLRenderer) Init(width, height int32, _ *glfw.Window) {
	if err := gl.Init(); err != nil {
		logger.Log.Error("OpenGL initialization failed", zap.Error(err))
		return
	}

	if Debug {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
	}
	gl.GenBuffers(1, &rend.instanceVBO)
	FrustumCullingEnabled = false
	FaceCullingEnabled = false
	SetDefaultTexture(rend)
	gl.Viewport(0, 0, width, height)
	rend.InitShader()
	logger.Log.Info("OpenGL render initialized")
}

func (rend *OpenGLRenderer) InitShader() {
	rend.defaultShader = InitShader()
	rend.defaultShader.Compile()
}

func (rend *OpenGLRenderer) AddModel(model *Model) {
	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(model.InterleavedData)*4, gl.Ptr(model.InterleavedData), gl.STATIC_DRAW)

	var ebo uint32
	gl.GenBuffers(1, &ebo)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ebo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(model.Faces)*4, gl.Ptr(model.Faces), gl.STATIC_DRAW)

	stride := int32((8) * 4)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)

	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(1)

	gl.VertexAttribPointer(2, 3, gl.FLOAT, false, stride, gl.PtrOffset(5*4))
	gl.EnableVertexAttribArray(2)

	if model.IsInstanced && len(model.InstanceModelMatrices) > 0 {
		gl.BindBuffer(gl.ARRAY_BUFFER, rend.instanceVBO)
		gl.BufferData(gl.ARRAY_BUFFER, len(model.InstanceModelMatrices)*int(unsafe.Sizeof(mgl32.Mat4{})), gl.Ptr(model.InstanceModelMatrices), gl.DYNAMIC_DRAW)

		for i := 0; i < 4; i++ {
			gl.EnableVertexAttribArray(3 + uint32(i))
			gl.VertexAttribPointer(3+uint32(i), 4, gl.FLOAT, false, int32(unsafe.Sizeof(mgl32.Mat4{})), unsafe.Pointer(uintptr(i*16)))
			gl.VertexAttribDivisor(3+uint32(i), 1)
		}
	}

	model.VAO = vao
	model.VBO = vbo
	model.EBO = ebo

	// Calculate the initial model matrix based on position, rotation, and scale
	model.updateModelMatrix()

	rend.Models = append(rend.Models, model)
}

func (rend *OpenGLRenderer) RemoveModel(model *Model) {
	for i, m := range rend.Models {
		if m == model {
			rend.Models = append(rend.Models[:i], rend.Models[i+1:]...)
			break
		}
	}
}

func (model *Model) RemoveModelInstance(index int) {
	if index >= len(model.InstanceModelMatrices) {
		return
	}
	model.InstanceModelMatrices = append(model.InstanceModelMatrices[:index], model.InstanceModelMatrices[index+1:]...)
	model.InstanceCount--
}

func (rend *OpenGLRenderer) Render(camera Camera, light *Light) {
	// Use skybox color for clear color if skybox is present
	if rend.skybox != nil && rend.skybox.Shader.skyColor != (mgl32.Vec3{}) {
		// Use skybox color as background
		gl.ClearColor(rend.skybox.Shader.skyColor.X(), rend.skybox.Shader.skyColor.Y(), rend.skybox.Shader.skyColor.Z(), 1.0)
	} else {
		// Use black background
		gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	}
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	// Skybox is now handled by clear color - no rendering needed

	// Ensure depth testing is enabled for models (skybox may have disabled it)
	if DepthTestEnabled {
		gl.Enable(gl.DEPTH_TEST)
		gl.DepthMask(true) // Ensure depth writing is enabled for models
	} else {
		gl.Disable(gl.DEPTH_TEST)
	}
	viewProjection := camera.GetViewProjection()

	// Culling : https://learnopengl.com/Advanced-OpenGL/Face-culling
	if FaceCullingEnabled {
		gl.Enable(gl.CULL_FACE)
		// IF FACES OF THE MODEL ARE RENDERED IN THE WRONG ORDER, TRY SWITCHING THE FOLLOWING LINE TO gl.CCW or we need to make sure the winding of each model is consistent
		// CCW = Counter ClockWise
		gl.CullFace(gl.BACK) // Changed from FRONT to BACK
		gl.FrontFace(gl.CCW) // Changed from CW to CCW
	}

	// Calculate frustum
	// TODO: Add check to see if camera is dirty(moved)
	if FrustumCullingEnabled {
		frustum = camera.CalculateFrustum()
	}

	modLen := len(rend.Models)
	for i := 0; i < modLen; i++ {
		model := rend.Models[i]

		// Skip rendering if the model is outside the frustum
		if FrustumCullingEnabled && !frustum.IntersectsSphere(model.BoundingSphereCenter, model.BoundingSphereRadius) {
			continue
		}

		if model.IsDirty {
			// Recalculate the model matrix only if necessary
			model.calculateModelMatrix()
			model.IsDirty = false
		}

		// Determine which shader to use
		var shader *Shader
		if model.Shader.IsValid() {
			shader = &model.Shader
			// Ensure custom shader is compiled before using
			if !shader.isCompiled {
				shader.Compile()
			}
			// Using custom shader
		} else {
			shader = &rend.defaultShader
			// Using default shader
		}

		// Switch shader if needed
		if rend.currentShaderProgram != shader.program {
			shader.Use()
			rend.currentShaderProgram = shader.program
		}

		// Set common uniforms for all shaders
		rend.setCommonUniforms(shader, viewProjection, model, light, camera)

		// Set material uniforms if applicable
		if model.Material != nil {
			rend.setMaterialUniforms(shader, model)
		}

		// Set shader-specific uniforms (like water shader uniforms)
		rend.setShaderSpecificUniforms(shader, model)

		// Bind texture
		if model.Material != nil && model.Material.TextureID != currentTextureID {
			gl.BindTexture(gl.TEXTURE_2D, model.Material.TextureID)
			currentTextureID = model.Material.TextureID
		}

		// Set texture uniform (cache this location for better performance)
		textureUniform := gl.GetUniformLocation(shader.program, gl.Str("textureSampler\x00"))
		if textureUniform != -1 {
			gl.Uniform1i(textureUniform, 0)
		}

		// Render the model
		gl.BindVertexArray(model.VAO)
		if model.IsInstanced && len(model.InstanceModelMatrices) > 0 {
			rend.UpdateInstanceMatrices(model)
			gl.DrawElementsInstanced(gl.TRIANGLES, int32(len(model.Faces)), gl.UNSIGNED_INT, nil, int32(model.InstanceCount))
			shader.SetInt("isInstanced", 1)
		} else {
			// Regular draw
			gl.DrawElements(gl.TRIANGLES, int32(len(model.Faces)), gl.UNSIGNED_INT, nil)
			shader.SetInt("isInstanced", 0)
		}
		gl.BindVertexArray(0)
	}
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.CULL_FACE)
}

// setCommonUniforms sets uniforms that are common to most shaders
func (rend *OpenGLRenderer) setCommonUniforms(shader *Shader, viewProjection mgl32.Mat4, model *Model, light *Light, camera Camera) {
	// Set view projection matrix
	viewProjLoc := gl.GetUniformLocation(shader.program, gl.Str("viewProjection\x00"))
	if viewProjLoc != -1 {
		gl.UniformMatrix4fv(viewProjLoc, 1, false, &viewProjection[0])
	}

	// Set model matrix
	modelLoc := gl.GetUniformLocation(shader.program, gl.Str("model\x00"))
	if modelLoc != -1 {
		gl.UniformMatrix4fv(modelLoc, 1, false, &model.ModelMatrix[0])
	}

	// Set light uniforms
	if light != nil {
		// Standard light uniforms (for default shader)
		lightPosLoc := gl.GetUniformLocation(shader.program, gl.Str("light.position\x00"))
		if lightPosLoc != -1 {
			gl.Uniform3f(lightPosLoc, light.Position[0], light.Position[1], light.Position[2])
		}

		lightColorLoc := gl.GetUniformLocation(shader.program, gl.Str("light.color\x00"))
		if lightColorLoc != -1 {
			gl.Uniform3f(lightColorLoc, light.Color[0], light.Color[1], light.Color[2])
		}

		lightIntensityLoc := gl.GetUniformLocation(shader.program, gl.Str("light.intensity\x00"))
		if lightIntensityLoc != -1 {
			gl.Uniform1f(lightIntensityLoc, light.Intensity)
		}

		// New lighting features
		ambientStrengthLoc := gl.GetUniformLocation(shader.program, gl.Str("light.ambientStrength\x00"))
		if ambientStrengthLoc != -1 {
			gl.Uniform1f(ambientStrengthLoc, light.AmbientStrength)
		}

		temperatureLoc := gl.GetUniformLocation(shader.program, gl.Str("light.temperature\x00"))
		if temperatureLoc != -1 {
			gl.Uniform1f(temperatureLoc, light.Temperature)
		}

		isDirectionalLoc := gl.GetUniformLocation(shader.program, gl.Str("light.isDirectional\x00"))
		if isDirectionalLoc != -1 {
			isDirectional := 0
			if light.Mode == "directional" {
				isDirectional = 1
			}
			gl.Uniform1i(isDirectionalLoc, int32(isDirectional))
		}

		directionLoc := gl.GetUniformLocation(shader.program, gl.Str("light.direction\x00"))
		if directionLoc != -1 {
			gl.Uniform3f(directionLoc, light.Direction[0], light.Direction[1], light.Direction[2])
		}

		// Attenuation factors
		constantAttenLoc := gl.GetUniformLocation(shader.program, gl.Str("light.constantAtten\x00"))
		if constantAttenLoc != -1 {
			gl.Uniform1f(constantAttenLoc, light.ConstantAtten)
		}

		linearAttenLoc := gl.GetUniformLocation(shader.program, gl.Str("light.linearAtten\x00"))
		if linearAttenLoc != -1 {
			gl.Uniform1f(linearAttenLoc, light.LinearAtten)
		}

		quadraticAttenLoc := gl.GetUniformLocation(shader.program, gl.Str("light.quadraticAtten\x00"))
		if quadraticAttenLoc != -1 {
			gl.Uniform1f(quadraticAttenLoc, light.QuadraticAtten)
		}

		// Water shader specific light uniforms (keeping for backward compatibility)
		lightPosLocWater := gl.GetUniformLocation(shader.program, gl.Str("lightPos\x00"))
		if lightPosLocWater != -1 {
			gl.Uniform3f(lightPosLocWater, light.Position[0], light.Position[1], light.Position[2])
		}

		lightColorLocWater := gl.GetUniformLocation(shader.program, gl.Str("lightColor\x00"))
		if lightColorLocWater != -1 {
			gl.Uniform3f(lightColorLocWater, light.Color[0], light.Color[1], light.Color[2])
		}

		lightIntensityLocWater := gl.GetUniformLocation(shader.program, gl.Str("lightIntensity\x00"))
		if lightIntensityLocWater != -1 {
			gl.Uniform1f(lightIntensityLocWater, light.Intensity)
		}

		lightDirectionLocWater := gl.GetUniformLocation(shader.program, gl.Str("lightDirection\x00"))
		if lightDirectionLocWater != -1 {
			gl.Uniform3f(lightDirectionLocWater, light.Direction[0], light.Direction[1], light.Direction[2])
		}
	}

	// Set view position (for specular lighting)
	viewPosLoc := gl.GetUniformLocation(shader.program, gl.Str("viewPos\x00"))
	if viewPosLoc != -1 {
		gl.Uniform3f(viewPosLoc, camera.Position[0], camera.Position[1], camera.Position[2])
	}
}

// setMaterialUniforms sets material-specific uniforms
func (rend *OpenGLRenderer) setMaterialUniforms(shader *Shader, model *Model) {
	if model.Material == nil {
		// Use default material if none is set
		model.Material = DefaultMaterial
	}

	// Set diffuse color
	diffuseColorLoc := gl.GetUniformLocation(shader.program, gl.Str("diffuseColor\x00"))
	if diffuseColorLoc != -1 {
		gl.Uniform3fv(diffuseColorLoc, 1, &model.Material.DiffuseColor[0])
	}

	// Set specular color
	specularColorLoc := gl.GetUniformLocation(shader.program, gl.Str("specularColor\x00"))
	if specularColorLoc != -1 {
		gl.Uniform3fv(specularColorLoc, 1, &model.Material.SpecularColor[0])
	}

	// Set shininess
	shininessLoc := gl.GetUniformLocation(shader.program, gl.Str("shininess\x00"))
	if shininessLoc != -1 {
		gl.Uniform1f(shininessLoc, model.Material.Shininess)
	}

	// Modern PBR properties
	metallicLoc := gl.GetUniformLocation(shader.program, gl.Str("metallic\x00"))
	if metallicLoc != -1 {
		gl.Uniform1f(metallicLoc, model.Material.Metallic)
	}

	roughnessLoc := gl.GetUniformLocation(shader.program, gl.Str("roughness\x00"))
	if roughnessLoc != -1 {
		gl.Uniform1f(roughnessLoc, model.Material.Roughness)
	}

	exposureLoc := gl.GetUniformLocation(shader.program, gl.Str("exposure\x00"))
	if exposureLoc != -1 {
		gl.Uniform1f(exposureLoc, model.Material.Exposure)
	}
}

// setShaderSpecificUniforms allows models to set custom uniforms for their shaders
func (rend *OpenGLRenderer) setShaderSpecificUniforms(shader *Shader, model *Model) {
	if model.CustomUniforms == nil {
		return
	}

	// Set all custom uniforms stored in the model
	for name, value := range model.CustomUniforms {
		switch v := value.(type) {
		case float32:
			shader.SetFloat(name, v)
		case int32:
			shader.SetInt(name, v)
		case bool:
			shader.SetBool(name, v)
		case mgl32.Vec3:
			shader.SetVec3(name, v)
		case []float32:
			// Handle float arrays for wave parameters
			location := gl.GetUniformLocation(shader.program, gl.Str(name+"\x00"))
			if location != -1 {
				if name == "waveDirections" {
					// Special handling for Vec3 arrays (3 floats per element)
					gl.Uniform3fv(location, int32(len(v)/3), &v[0])
				} else {
					// Regular float arrays
					gl.Uniform1fv(location, int32(len(v)), &v[0])
				}
			}
		default:
			// Skip unknown types
		}
	}
}

func (rend *OpenGLRenderer) SetSkybox(skybox *Skybox) {
	rend.skybox = skybox
}

func (rend *OpenGLRenderer) Cleanup() {
	for _, model := range rend.Models {
		gl.DeleteVertexArrays(1, &model.VAO)
		gl.DeleteBuffers(1, &model.VBO)
		gl.DeleteBuffers(1, &model.EBO)
	}
	if rend.skybox != nil {
		rend.skybox.Cleanup()
	}
}

func (rend *OpenGLRenderer) LoadTexture(filePath string) (uint32, error) { // TODO: Consider specifying image format or handling different formats properly

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

	// Set texture parameters (optional)
	// gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_R, gl.REPEAT)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	// GL_NEAREST results in blocked patterns where we can clearly see the pixels that form the texture while GL_LINEAR produces a smoother pattern where the individual pixels are less visible.
	// GL_LINEAR produces a more realistic output, but some developers prefer a more 8-bit look and as a result pick the GL_NEAREST option
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	return textureID, nil
}

func (rend *OpenGLRenderer) CreateTextureFromImage(img image.Image) (uint32, error) {
	var textureID uint32
	gl.GenTextures(1, &textureID)
	gl.BindTexture(gl.TEXTURE_2D, textureID)

	rgba, ok := img.(*image.RGBA)
	if !ok {
		// Convert to *image.RGBA if necessary
		b := img.Bounds()
		rgba = image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
		draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
	}
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(rgba.Rect.Size().X), int32(rgba.Rect.Size().Y), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgba.Pix))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	return textureID, nil
}

func GenShader(source string, shaderType uint32) uint32 {
	shader := gl.CreateShader(shaderType)
	cSources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, cSources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		logger.Log.Error("Failed to compile", zap.Uint32("shader type:", shaderType), zap.String("log", log))
		fmt.Printf("SHADER COMPILATION ERROR: Type %d, Log: %s\n", shaderType, log)
	} else {
		shaderTypeName := "VERTEX"
		if shaderType == gl.FRAGMENT_SHADER {
			shaderTypeName = "FRAGMENT"
		}
		fmt.Printf("SHADER COMPILED SUCCESSFULLY: %s shader\n", shaderTypeName)
	}

	return shader
}

func GenShaderProgram(vertexShader, fragmentShader uint32) uint32 {
	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		logger.Log.Error("Failed to link program", zap.String("log", log))
		fmt.Printf("SHADER PROGRAM LINK ERROR: %s\n", log)
	} else {
		fmt.Printf("SHADER PROGRAM LINKED SUCCESSFULLY: Program ID %d\n", program)
	}
	gl.DetachShader(program, vertexShader)
	gl.DeleteShader(vertexShader)
	gl.DetachShader(program, fragmentShader)
	gl.DeleteShader(fragmentShader)
	return program
}

func CreateLight() *Light {
	return &Light{
		Position:  mgl32.Vec3{0.0, 1500.0, 0.0}, // Example position
		Color:     mgl32.Vec3{1.0, 1.0, 1.0},    // White light
		Intensity: 1.0,                          // Full intensity
		Mode:      "point",                      // Default to point light
		// New lighting features with sensible defaults
		AmbientStrength: 0.1,                  // 10% ambient lighting (same as before)
		Temperature:     5500.0,               // Daylight color temperature
		Direction:       mgl32.Vec3{0, -1, 0}, // Default direction pointing down
		// Point light attenuation (suitable for large scenes like voxel worlds)
		ConstantAtten:  1.0,       // No constant attenuation
		LinearAtten:    0.0001,    // Very gentle linear falloff for large scenes
		QuadraticAtten: 0.0000001, // Minimal quadratic falloff for large scenes
	}
}

// CreateDirectionalLight creates a directional light (like the sun)
func CreateDirectionalLight(direction mgl32.Vec3, color mgl32.Vec3, intensity float32) *Light {
	light := CreateLight()
	light.Mode = "directional"
	light.Direction = direction.Normalize()
	light.Color = color
	light.Intensity = intensity
	light.AmbientStrength = 0.15 // Slightly higher ambient for outdoor scenes
	light.Temperature = 5500.0   // Daylight temperature
	return light
}

// CreatePointLight creates a point light with specified attenuation
func CreatePointLight(position mgl32.Vec3, color mgl32.Vec3, intensity float32, range_ float32) *Light {
	light := CreateLight()
	light.Mode = "point"
	light.Position = position
	light.Color = color
	light.Intensity = intensity

	// Calculate attenuation based on desired range
	// At range distance, light should be ~1% intensity (more reasonable than 5%)
	light.ConstantAtten = 1.0
	light.LinearAtten = 2.0 / range_
	light.QuadraticAtten = 1.0 / (range_ * range_)

	return light
}

// CreateWarmLight creates a warm-colored light (like incandescent bulb)
func CreateWarmLight(position mgl32.Vec3, intensity float32) *Light {
	light := CreatePointLight(position, mgl32.Vec3{1.0, 1.0, 1.0}, intensity, 100.0)
	light.Temperature = 2700.0 // Warm incandescent
	return light
}

// CreateCoolLight creates a cool-colored light (like fluorescent)
func CreateCoolLight(position mgl32.Vec3, intensity float32) *Light {
	light := CreatePointLight(position, mgl32.Vec3{1.0, 1.0, 1.0}, intensity, 100.0)
	light.Temperature = 6500.0 // Cool fluorescent
	return light
}

// CreateSunlight creates a realistic sun light
func CreateSunlight(direction mgl32.Vec3) *Light {
	light := CreateDirectionalLight(direction, mgl32.Vec3{1.0, 0.95, 0.8}, 1.2)
	light.Temperature = 5800.0  // Sun's actual temperature
	light.AmbientStrength = 0.2 // Higher ambient for outdoor scenes
	return light
}

func (rend *OpenGLRenderer) UpdateInstanceMatrices(model *Model) {
	if len(model.InstanceModelMatrices) > 0 {
		gl.BindBuffer(gl.ARRAY_BUFFER, rend.instanceVBO)
		gl.BufferData(gl.ARRAY_BUFFER, len(model.InstanceModelMatrices)*int(unsafe.Sizeof(mgl32.Mat4{})), gl.Ptr(model.InstanceModelMatrices), gl.DYNAMIC_DRAW)
	}
}

// GetDefaultShader returns a copy of the default shader for models that need it
func (rend *OpenGLRenderer) GetDefaultShader() Shader {
	return rend.defaultShader
}

// UpdateViewport updates the OpenGL viewport to match the current window size
func (rend *OpenGLRenderer) UpdateViewport(width, height int32) {
	gl.Viewport(0, 0, width, height)
}
