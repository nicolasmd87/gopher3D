package renderer

import (
	"Gopher3D/internal/logger"
	"fmt"
	"image"
	"sort"
	"strings"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"go.uber.org/zap"
)

var currentTextureID uint32 = ^uint32(0) // Initialize with an invalid value
var frustum Frustum
var frustumDirty bool = true // Track if frustum needs recalculation

// SetFrustumDirty marks frustum as needing recalculation
func SetFrustumDirty() {
	frustumDirty = true
}

type OpenGLRenderer struct {
	defaultShader        Shader
	defaultUniformCache  *UniformCache // Cache for default shader uniforms
	Models               []*Model
	instanceVBO          uint32  // Buffer for instance model matrices
	currentShaderProgram uint32  // Track currently bound shader to avoid unnecessary switches
	skybox               *Skybox // Optional skybox
	shaderCaches         map[uint32]*UniformCache // Cache per shader program
	textureManager       *TextureManager // Central texture cache and lifecycle management
	
	// GL state tracking to avoid redundant state changes
	faceCullingState     bool   // Current face culling state
	depthTestState       bool   // Current depth test state
}

func (rend *OpenGLRenderer) Init(width, height int32, _ *glfw.Window) {
	if err := gl.Init(); err != nil {
		logger.Log.Error("OpenGL initialization failed", zap.Error(err))
		return
	}

	if Debug {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
	} else {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
	}
	gl.GenBuffers(1, &rend.instanceVBO)
	FrustumCullingEnabled = false
	FaceCullingEnabled = false
	
	// Initialize texture manager
	rend.textureManager = NewTextureManager()
	logger.Log.Info("TextureManager initialized")
	
	SetDefaultTexture(rend)
	gl.Viewport(0, 0, width, height)
	rend.InitShader()
	
	// Initialize GL state tracking - set initial states
	gl.ClearDepth(1.0) // Ensure depth buffer clears to maximum depth
	rend.setFaceCulling(false)
	rend.setDepthTest(true)
	
	// Initialize shader cache map
	rend.shaderCaches = make(map[uint32]*UniformCache)
	
	logger.Log.Info("OpenGL render initialized")
}

// setFaceCulling only changes OpenGL face culling state if needed
func (rend *OpenGLRenderer) setFaceCulling(enabled bool) {
	if rend.faceCullingState != enabled {
		if enabled {
			gl.Enable(gl.CULL_FACE)
			gl.CullFace(gl.BACK)
			gl.FrontFace(gl.CCW)
		} else {
			gl.Disable(gl.CULL_FACE)
		}
		rend.faceCullingState = enabled
	}
}

// setDepthTest only changes OpenGL depth test state if needed
func (rend *OpenGLRenderer) setDepthTest(enabled bool) {
	if rend.depthTestState != enabled {
		if enabled {
			gl.Enable(gl.DEPTH_TEST)
			gl.DepthFunc(gl.LESS) // Standard depth test
			gl.DepthMask(true)
		} else {
			gl.Disable(gl.DEPTH_TEST)
		}
		rend.depthTestState = enabled
	}
}

func (rend *OpenGLRenderer) InitShader() {
	rend.defaultShader = InitShader()
	rend.defaultShader.Compile()
	// Initialize uniform cache for default shader
	rend.defaultUniformCache = NewUniformCache(rend.defaultShader.program)
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
		// Create a dedicated instance VBO for this model
		var instanceVBO uint32
		gl.GenBuffers(1, &instanceVBO)
		gl.BindBuffer(gl.ARRAY_BUFFER, instanceVBO)
		
		// Calculate buffer size - allocate exact size first, growth handled by UpdateInstanceMatrices
		matrixSize := int(unsafe.Sizeof(mgl32.Mat4{}))
		initialSize := len(model.InstanceModelMatrices) * matrixSize
		
		// Upload actual data with exact size
		gl.BufferData(gl.ARRAY_BUFFER, initialSize, gl.Ptr(model.InstanceModelMatrices), gl.DYNAMIC_DRAW)

		// Store the instance VBO and current capacity for buffer reuse optimization
		model.InstanceVBO = instanceVBO
		model.InstanceVBOCapacity = initialSize

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

	// Load textures for materials (now that OpenGL is initialized)
	rend.loadModelTextures(model)
	
	// Sort material groups by texture ID to minimize state changes
	rend.sortMaterialGroupsByTexture(model)
	
	// Log texture manager stats after loading
	rend.textureManager.LogStats()

	rend.Models = append(rend.Models, model)
}

// sortMaterialGroupsByTexture sorts material groups by texture ID to minimize GPU state changes
func (rend *OpenGLRenderer) sortMaterialGroupsByTexture(model *Model) {
	if len(model.MaterialGroups) <= 1 {
		return // No need to sort if 0 or 1 group
	}
	
	// Use stable sort to preserve order for groups with same texture
	sort.SliceStable(model.MaterialGroups, func(i, j int) bool {
		texI := uint32(0)
		texJ := uint32(0)
		if model.MaterialGroups[i].Material != nil {
			texI = model.MaterialGroups[i].Material.TextureID
		}
		if model.MaterialGroups[j].Material != nil {
			texJ = model.MaterialGroups[j].Material.TextureID
		}
		return texI < texJ
	})
	
	logger.Log.Debug("Material groups sorted by texture ID",
		zap.Int("groupCount", len(model.MaterialGroups)))
}

// loadModelTextures loads textures for all materials in the model
func (rend *OpenGLRenderer) loadModelTextures(model *Model) {
	// Load textures for material groups
	if len(model.MaterialGroups) > 0 {
		for i := range model.MaterialGroups {
			material := model.MaterialGroups[i].Material
			if material != nil {
				if material.TexturePath != "" && material.TextureID == 0 {
					// Texture path is set but not loaded yet - use texture manager
					textureID, err := rend.textureManager.LoadTexture(material.TexturePath)
					if err != nil {
						logger.Log.Warn("Failed to load texture for material, using default",
							zap.String("material", material.Name),
							zap.String("path", material.TexturePath),
							zap.Error(err))
						// Ensure we have a valid default texture ID
						if DefaultMaterial.TextureID == 0 {
							logger.Log.Error("DefaultMaterial.TextureID is 0! This will cause rendering issues.")
						}
						material.TextureID = DefaultMaterial.TextureID
						rend.textureManager.AddReference(DefaultMaterial.TextureID)
					} else {
						material.TextureID = textureID
						logger.Log.Debug("Loaded texture for material via TextureManager",
							zap.String("material", material.Name),
							zap.String("path", material.TexturePath))
					}
				} else if material.TextureID == 0 {
					// No texture path - don't assign any texture, let shader use diffuse color
					// This allows materials to show their proper diffuse colors from MTL
					logger.Log.Debug("Material without texture will use diffuse color",
						zap.String("material", material.Name),
						zap.Float32s("diffuseColor", material.DiffuseColor[:]))
				}
			}
		}
	} else if model.Material != nil {
		if model.Material.TexturePath != "" && model.Material.TextureID == 0 {
			// Single material model with texture path - use texture manager
			textureID, err := rend.textureManager.LoadTexture(model.Material.TexturePath)
			if err != nil {
				logger.Log.Warn("Failed to load texture for material, using default",
					zap.String("material", model.Material.Name),
					zap.String("path", model.Material.TexturePath),
					zap.Error(err))
				// Ensure we have a valid default texture ID
				if DefaultMaterial.TextureID == 0 {
					logger.Log.Error("DefaultMaterial.TextureID is 0! This will cause rendering issues.")
				}
				model.Material.TextureID = DefaultMaterial.TextureID
				rend.textureManager.AddReference(DefaultMaterial.TextureID)
			} else {
				model.Material.TextureID = textureID
				logger.Log.Debug("Loaded texture for material via TextureManager",
					zap.String("material", model.Material.Name),
					zap.String("path", model.Material.TexturePath))
			}
		} else if model.Material.TextureID == 0 {
			// Single material model without texture path - don't assign texture
			// This allows the material to show its proper diffuse color from MTL
			logger.Log.Debug("Single material without texture will use diffuse color",
				zap.String("material", model.Material.Name),
				zap.Float32s("diffuseColor", model.Material.DiffuseColor[:]))
		}
	}
}

func (rend *OpenGLRenderer) RemoveModel(model *Model) {
	// Release all material group textures
	for _, group := range model.MaterialGroups {
		if group.Material != nil && group.Material.TextureID != 0 {
			rend.textureManager.ReleaseTexture(group.Material.TextureID)
		}
	}
	// Release main material texture
	if model.Material != nil && model.Material.TextureID != 0 {
		rend.textureManager.ReleaseTexture(model.Material.TextureID)
	}
	
	// Remove from models list
	for i, m := range rend.Models {
		if m == model {
			rend.Models = append(rend.Models[:i], rend.Models[i+1:]...)
			break
		}
	}
	
	logger.Log.Debug("Model removed and textures released",
		zap.Int("materialGroups", len(model.MaterialGroups)))
}

func (model *Model) RemoveModelInstance(index int) {
	if index >= len(model.InstanceModelMatrices) {
		return
	}
	model.InstanceModelMatrices = append(model.InstanceModelMatrices[:index], model.InstanceModelMatrices[index+1:]...)
	model.InstanceCount--
}

func (rend *OpenGLRenderer) Render(camera Camera, light *Light) {
	// Force fill mode
	gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)

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

	// Set depth test state (only changes if different from current state)
	rend.setDepthTest(DepthTestEnabled)

	// Enable alpha blending for transparency
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	// For transparent objects, disable depth writing but keep depth testing
	// This will be handled per-model based on alpha value
	viewProjection := camera.GetViewProjection()

	// Set face culling state (only changes if different from current state)
	rend.setFaceCulling(FaceCullingEnabled)

	// Calculate frustum only if camera moved
	if FrustumCullingEnabled && frustumDirty {
		frustum = camera.CalculateFrustum()
		frustumDirty = false
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
		var uniformCache *UniformCache
		
		if model.Shader.IsValid() {
			shader = &model.Shader
			// Ensure custom shader is compiled before using
			if !shader.isCompiled {
				shader.Compile()
			}
			// Get or create cache for this shader
			if cache, exists := rend.shaderCaches[shader.program]; exists {
				uniformCache = cache
			} else {
				uniformCache = NewUniformCache(shader.program)
				rend.shaderCaches[shader.program] = uniformCache
			}
		} else {
			shader = &rend.defaultShader
			uniformCache = rend.defaultUniformCache
		}

		// Switch shader if needed
		if rend.currentShaderProgram != shader.program {
			shader.Use()
			rend.currentShaderProgram = shader.program
		}

		// Set common uniforms for all shaders using cache
		rend.setCommonUniformsCached(uniformCache, viewProjection, model, light, camera)

		// Set shader-specific uniforms (like water shader uniforms)
		rend.setShaderSpecificUniforms(shader, model)

		// Bind vertex array
		gl.BindVertexArray(model.VAO)

		// Check if model has multiple material groups
		if len(model.MaterialGroups) > 0 {
			// Multi-material rendering - optimized for texture batching
			currentTextureID := uint32(0)
			
			// Cache texture sampler uniform location (only look up once per shader)
			textureSamplerLoc := uniformCache.GetLocation("textureSampler")
			
			for _, group := range model.MaterialGroups {
				// Set material uniforms for this group
				rend.setMaterialUniforms(shader, group.Material)

				// Handle transparency depth writing
				if group.Material != nil && group.Material.Alpha < 0.99 {
					gl.DepthMask(false) // Disable depth writing for transparent objects
					gl.Enable(gl.BLEND)
					gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
				} else {
					gl.DepthMask(true) // Enable depth writing for opaque objects
					gl.Enable(gl.BLEND)
					gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
				}

				// Bind texture only when it changes (optimized for sorted material groups)
				if group.Material != nil && group.Material.TextureID != 0 {
					if group.Material.TextureID != currentTextureID {
						gl.BindTexture(gl.TEXTURE_2D, group.Material.TextureID)
						gl.Uniform1i(textureSamplerLoc, 0)
						currentTextureID = group.Material.TextureID
					}
				} else if group.Material != nil && group.Material.TextureID == 0 {
					// Material without texture - bind default white texture
					if DefaultMaterial.TextureID != 0 && DefaultMaterial.TextureID != currentTextureID {
						gl.BindTexture(gl.TEXTURE_2D, DefaultMaterial.TextureID)
						gl.Uniform1i(textureSamplerLoc, 0)
						currentTextureID = DefaultMaterial.TextureID
					}
				}

				// Draw this material group
				if model.IsInstanced && len(model.InstanceModelMatrices) > 0 {
					if model.InstanceMatricesUpdated {
						rend.UpdateInstanceMatrices(model)
						model.InstanceMatricesUpdated = false
					}
					shader.SetInt("isInstanced", 1)
					gl.DrawElementsInstanced(gl.TRIANGLES, group.IndexCount, gl.UNSIGNED_INT, 
						gl.PtrOffset(int(group.IndexStart)*4), int32(model.InstanceCount))
				} else {
					shader.SetInt("isInstanced", 0)
					gl.DrawElements(gl.TRIANGLES, group.IndexCount, gl.UNSIGNED_INT, 
						gl.PtrOffset(int(group.IndexStart)*4))
				}
			}
		} else {
			// Single material rendering (backward compatible)
			if model.Material != nil {
				rend.setMaterialUniforms(shader, model.Material)

				// Handle transparency depth writing
				if model.Material.Alpha < 0.99 {
					gl.DepthMask(false) // Disable depth writing for transparent objects
					gl.Enable(gl.BLEND)
					gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
				} else {
					gl.DepthMask(true) // Enable depth writing for opaque objects
					gl.Enable(gl.BLEND)
					gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
				}
			}

			// Bind texture (only if it has a valid texture) - optimized
			textureSamplerLoc := uniformCache.GetLocation("textureSampler")
			
			if model.Material != nil && model.Material.TextureID != 0 && model.Material.TextureID != currentTextureID {
				gl.BindTexture(gl.TEXTURE_2D, model.Material.TextureID)
				gl.Uniform1i(textureSamplerLoc, 0)
				currentTextureID = model.Material.TextureID
			} else if model.Material != nil && model.Material.TextureID == 0 {
				// Material without texture - bind default white texture
				if DefaultMaterial.TextureID != 0 && DefaultMaterial.TextureID != currentTextureID {
					gl.BindTexture(gl.TEXTURE_2D, DefaultMaterial.TextureID)
					gl.Uniform1i(textureSamplerLoc, 0)
					currentTextureID = DefaultMaterial.TextureID
				}
			}

			// Render the model
			if model.IsInstanced && len(model.InstanceModelMatrices) > 0 {
				if model.InstanceMatricesUpdated {
					rend.UpdateInstanceMatrices(model)
					model.InstanceMatricesUpdated = false
				}
				shader.SetInt("isInstanced", 1)
				gl.DrawElementsInstanced(gl.TRIANGLES, int32(len(model.Faces)), gl.UNSIGNED_INT, nil, int32(model.InstanceCount))
			} else {
				shader.SetInt("isInstanced", 0)
				gl.DrawElements(gl.TRIANGLES, int32(len(model.Faces)), gl.UNSIGNED_INT, nil)
			}
		}
		gl.BindVertexArray(0)
	}
	// GL state is now managed through setFaceCulling() and setDepthTest()
	// No need to disable at end of frame
}

// setCommonUniformsCached sets uniforms using cached locations for better performance
func (rend *OpenGLRenderer) setCommonUniformsCached(cache *UniformCache, viewProjection mgl32.Mat4, model *Model, light *Light, camera Camera) {
	// Set view projection matrix
	viewProjLoc := cache.GetLocation("viewProjection")
	if viewProjLoc != -1 {
		gl.UniformMatrix4fv(viewProjLoc, 1, false, &viewProjection[0])
	}

	// Set model matrix
	modelLoc := cache.GetLocation("model")
	if modelLoc != -1 {
		gl.UniformMatrix4fv(modelLoc, 1, false, &model.ModelMatrix[0])
	}

	// Set light uniforms
	if light != nil {
		cache.SetVec3("light.position", light.Position[0], light.Position[1], light.Position[2])
		cache.SetVec3("light.color", light.Color[0], light.Color[1], light.Color[2])
		cache.SetFloat("light.intensity", light.Intensity)
		cache.SetFloat("light.ambientStrength", light.AmbientStrength)
		cache.SetFloat("light.temperature", light.Temperature)
		
		isDirectional := int32(0)
		if light.Mode == "directional" {
			isDirectional = 1
		}
		cache.SetInt("light.isDirectional", isDirectional)
		
		cache.SetVec3("light.direction", light.Direction[0], light.Direction[1], light.Direction[2])
		cache.SetFloat("light.constantAtten", light.ConstantAtten)
		cache.SetFloat("light.linearAtten", light.LinearAtten)
		cache.SetFloat("light.quadraticAtten", light.QuadraticAtten)
		
		// Water shader specific light uniforms (backward compatibility)
		cache.SetVec3("lightPos", light.Position[0], light.Position[1], light.Position[2])
		cache.SetVec3("lightColor", light.Color[0], light.Color[1], light.Color[2])
		cache.SetFloat("lightIntensity", light.Intensity)
		cache.SetVec3("lightDirection", light.Direction[0], light.Direction[1], light.Direction[2])
	}

	// Set view position
	cache.SetVec3("viewPos", camera.Position[0], camera.Position[1], camera.Position[2])
}

// setMaterialUniforms sets material-specific uniforms
func (rend *OpenGLRenderer) setMaterialUniforms(shader *Shader, material *Material) {
	if material == nil {
		// Use default material if none is set
		material = DefaultMaterial
	}

	// Set diffuse color
	diffuseColorLoc := gl.GetUniformLocation(shader.program, gl.Str("diffuseColor\x00"))
	if diffuseColorLoc != -1 {
		gl.Uniform3fv(diffuseColorLoc, 1, &material.DiffuseColor[0])
	}

	// Set specular color
	specularColorLoc := gl.GetUniformLocation(shader.program, gl.Str("specularColor\x00"))
	if specularColorLoc != -1 {
		gl.Uniform3fv(specularColorLoc, 1, &material.SpecularColor[0])
	}

	// Set shininess
	shininessLoc := gl.GetUniformLocation(shader.program, gl.Str("shininess\x00"))
	if shininessLoc != -1 {
		gl.Uniform1f(shininessLoc, material.Shininess)
	}

	// Modern PBR properties
	metallicLoc := gl.GetUniformLocation(shader.program, gl.Str("metallic\x00"))
	if metallicLoc != -1 {
		gl.Uniform1f(metallicLoc, material.Metallic)
	}

	roughnessLoc := gl.GetUniformLocation(shader.program, gl.Str("roughness\x00"))
	if roughnessLoc != -1 {
		gl.Uniform1f(roughnessLoc, material.Roughness)
	}

	exposureLoc := gl.GetUniformLocation(shader.program, gl.Str("exposure\x00"))
	if exposureLoc != -1 {
		gl.Uniform1f(exposureLoc, material.Exposure)
	}

	alphaLoc := gl.GetUniformLocation(shader.program, gl.Str("materialAlpha\x00"))
	if alphaLoc != -1 {
		gl.Uniform1f(alphaLoc, material.Alpha)
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

// LoadTexture loads a texture from file (delegates to TextureManager for caching)
// Kept for backward compatibility
func (rend *OpenGLRenderer) LoadTexture(filePath string) (uint32, error) {
	return rend.textureManager.LoadTexture(filePath)
}

// CreateTextureFromImage creates a texture from an image.Image (delegates to TextureManager)
// Used for embedded textures like default texture
func (rend *OpenGLRenderer) CreateTextureFromImage(img image.Image) (uint32, error) {
	return rend.textureManager.CreateTextureFromImage(img, "embedded_texture")
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

// UpdateInstanceMatrices updates instance matrices with buffer reuse optimization
// Only reallocates GPU buffer if capacity needs to grow, otherwise uses faster BufferSubData
func (rend *OpenGLRenderer) UpdateInstanceMatrices(model *Model) {
	if len(model.InstanceModelMatrices) == 0 || model.InstanceVBO == 0 {
		return
	}
	
	gl.BindBuffer(gl.ARRAY_BUFFER, model.InstanceVBO)
	
	matrixSize := int(unsafe.Sizeof(mgl32.Mat4{}))
	currentSize := len(model.InstanceModelMatrices) * matrixSize
	
	// Check if buffer needs to grow (reallocation required)
	if currentSize > model.InstanceVBOCapacity {
		// Buffer too small - need to reallocate with gl.BufferData
		// Allocate 50% more than needed to reduce future reallocations
		newCapacity := int(float32(currentSize) * 1.5)
		gl.BufferData(gl.ARRAY_BUFFER, newCapacity, gl.Ptr(model.InstanceModelMatrices), gl.DYNAMIC_DRAW)
		model.InstanceVBOCapacity = newCapacity
	} else {
		// Buffer is large enough - use faster BufferSubData (no reallocation)
		// This is 2-5x faster than BufferData for updates
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, currentSize, gl.Ptr(model.InstanceModelMatrices))
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
