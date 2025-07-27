package renderer

import (
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// =============================================================
//
//	Shaders
//
// =============================================================
type Shader struct {
	vertexSource   string
	fragmentSource string
	program        uint32
	Name           string "default"
	isCompiled     bool
}

func (shader *Shader) Use() {
	if !shader.isCompiled {
		shader.Compile()
	}
	gl.UseProgram(shader.program)
}

// Compile compiles the shader sources into an OpenGL program
func (shader *Shader) Compile() error {
	if shader.isCompiled {
		return nil
	}

	vertexShader := GenShader(shader.vertexSource, gl.VERTEX_SHADER)
	fragmentShader := GenShader(shader.fragmentSource, gl.FRAGMENT_SHADER)
	shader.program = GenShaderProgram(vertexShader, fragmentShader)
	shader.isCompiled = true
	return nil
}

func (shader *Shader) SetVec3(name string, value mgl32.Vec3) {
	location := gl.GetUniformLocation(shader.program, gl.Str(name+"\x00"))
	gl.Uniform3f(location, value.X(), value.Y(), value.Z())
}

func (shader *Shader) SetFloat(name string, value float32) {
	location := gl.GetUniformLocation(shader.program, gl.Str(name+"\x00"))
	gl.Uniform1f(location, value)
}

func (shader *Shader) SetInt(name string, value int32) {
	location := gl.GetUniformLocation(shader.program, gl.Str(name+"\x00"))
	gl.Uniform1i(location, value)
}

// IsValid returns true if this shader has source code (not default empty shader)
func (shader *Shader) IsValid() bool {
	return shader.vertexSource != "" && shader.fragmentSource != ""
}

var vertexShaderSource = `#version 330 core

layout(location = 0) in vec3 inPosition; // Vertex position
layout(location = 1) in vec2 inTexCoord; // Texture Coordinate
layout(location = 2) in vec3 inNormal;   // Vertex normal
layout(location = 3) in mat4 instanceModel; // Instanced model matrix

uniform bool isInstanced; // Flag to differentiate instanced vs non-instanced rendering
uniform mat4 model;       // Regular model matrix
uniform mat4 viewProjection;

out vec2 fragTexCoord;    // Pass to fragment shader
out vec3 Normal;          // Pass normal to fragment shader
out vec3 FragPos;         // Pass position to fragment shader

void main() {
    // Decide whether to use instanced or regular model matrix
    mat4 modelMatrix = isInstanced ? instanceModel : model;

    FragPos = vec3(modelMatrix * vec4(inPosition, 1.0));
    Normal = mat3(modelMatrix) * inNormal; // Use this if the model matrix has no non-uniform scaling
    fragTexCoord = inTexCoord;

    // Final vertex position
    gl_Position = viewProjection * modelMatrix * vec4(inPosition, 1.0);
}

` + "\x00"

var fragmentShaderSource = `// Fragment Shader
#version 330 core
in vec2 fragTexCoord;
in vec3 Normal;
in vec3 FragPos;

uniform sampler2D textureSampler;
uniform struct Light {
    vec3 position;
    vec3 color;
    float intensity;
} light;
uniform vec3 viewPos;
uniform vec3 diffuseColor;
uniform vec3 specularColor;
uniform float shininess;

out vec4 FragColor;

void main() {
    vec4 texColor = texture(textureSampler, fragTexCoord);

    float ambientStrength = 0.1;
    vec3 ambient = ambientStrength * light.color * diffuseColor;

    vec3 norm = normalize(Normal);
    vec3 lightDir = normalize(light.position - FragPos);
    float diff = max(dot(norm, lightDir), 0.0);
    vec3 diffuse = diff * light.color * diffuseColor;

    vec3 viewDir = normalize(viewPos - FragPos);
    vec3 reflectDir = reflect(-lightDir, norm);
    float spec = pow(max(dot(viewDir, reflectDir), 0.0), shininess);
    vec3 specular = spec * light.color * specularColor;

    vec3 result = (ambient + diffuse + specular) * light.intensity;
    FragColor = vec4(result, 1.0) * texColor;
}
` + "\x00"

var waterVertexShaderSource = `#version 330 core

layout (location = 0) in vec3 aPos;
layout (location = 1) in vec2 aTexCoord;
layout (location = 2) in vec3 aNormal;

out vec2 fragTexCoord;
out vec3 fragNormal;
out vec3 fragPosition;

uniform mat4 model;
uniform mat4 viewProjection;
uniform float time;

// Enhanced wave uniforms
uniform vec3 waveDirections[4];  // Back to 4 waves for debugging
uniform float waveAmplitudes[4];
uniform float waveFrequencies[4];
uniform float waveSpeeds[4];
uniform float wavePhases[4];      // Phase offsets for variation
uniform float waveSteepness[4];   // Control wave shape

// Enhanced Gerstner wave with steepness control and phase offset
vec3 calculateGerstnerWave(vec3 position, vec3 direction, float amplitude, float frequency, float speed, float phase, float steepness, float time) {
    vec2 d = normalize(direction.xz);
    float wave = dot(d, position.xz) * frequency + time * speed + phase;
    float c = cos(wave);
    float s = sin(wave);
    
    // Q factor controls wave steepness (0 = sine wave, higher = sharper peaks)
    float Q = steepness / (frequency * amplitude * 6.0 + 0.01); // Prevent division by zero
    
    return vec3(
        Q * amplitude * d.x * c,  // Horizontal displacement X
        amplitude * s,             // Vertical displacement
        Q * amplitude * d.y * c   // Horizontal displacement Z
    );
}

// Enhanced normal calculation with steepness
vec3 calculateGerstnerNormal(vec3 position, vec3 direction, float amplitude, float frequency, float speed, float phase, float steepness, float time) {
    vec2 d = normalize(direction.xz);
    float wave = dot(d, position.xz) * frequency + time * speed + phase;
    float c = cos(wave);
    float s = sin(wave);
    
    float Q = steepness / (frequency * amplitude * 6.0 + 0.01);
    float WA = frequency * amplitude;
    
    return vec3(
        -d.x * WA * c,               // Normal X component
        1.0 - Q * WA * s,           // Normal Y component (reduced by horizontal displacement)
        -d.y * WA * c               // Normal Z component
    );
}

void main() {
    vec3 worldPos = vec3(model * vec4(aPos, 1.0));
    
    // Calculate displacement and normals from all waves
    vec3 totalDisplacement = vec3(0.0);
    vec3 totalNormal = vec3(0.0, 1.0, 0.0);
    
    // Process 4 waves for debugging
    for (int i = 0; i < 4; i++) {
        vec3 waveDisp = calculateGerstnerWave(
            worldPos,
            waveDirections[i],
            waveAmplitudes[i],
            waveFrequencies[i],
            waveSpeeds[i],
            wavePhases[i],
            waveSteepness[i],
            time
        );
        
        vec3 waveNormal = calculateGerstnerNormal(
            worldPos,
            waveDirections[i],
            waveAmplitudes[i],
            waveFrequencies[i],
            waveSpeeds[i],
            wavePhases[i],
            waveSteepness[i],
            time
        );
        
        totalDisplacement += waveDisp;
        totalNormal += waveNormal;
    }
    
    // Ensure displacement is visible
    totalDisplacement.y *= 2.0; // Double the vertical displacement for visibility
    
    // Apply displacement
    worldPos += totalDisplacement;
    
    // Normalize the accumulated normal
    totalNormal = normalize(totalNormal);
    
    fragPosition = worldPos;
    fragTexCoord = aTexCoord;
    fragNormal = totalNormal;
    
    gl_Position = viewProjection * vec4(worldPos, 1.0);
}
` + "\x00"

var waterFragmentShaderSource = `#version 330 core

in vec2 fragTexCoord;
in vec3 fragNormal;
in vec3 fragPosition;

// Enhanced water shader uniforms
uniform vec3 lightPos;
uniform vec3 lightColor;
uniform float lightIntensity;
uniform vec3 viewPos;
uniform float time;

out vec4 FragColor;

// High-quality noise functions with pattern elimination
float hash(vec2 p) {
    vec3 p3 = fract(vec3(p.xyx) * 0.1031);
    p3 += dot(p3, p3.yzx + 33.33);
    return fract((p3.x + p3.y) * p3.z);
}

float noise(vec2 st) {
    vec2 i = floor(st);
    vec2 f = fract(st);
    
    float a = hash(i);
    float b = hash(i + vec2(1.0, 0.0));
    float c = hash(i + vec2(0.0, 1.0));
    float d = hash(i + vec2(1.0, 1.0));
    
    // Use smoother interpolation (quintic instead of cubic)
    vec2 u = f * f * f * (f * (f * 6.0 - 15.0) + 10.0);
    
    return mix(a, b, u.x) + (c - a) * u.y * (1.0 - u.x) + (d - b) * u.x * u.y;
}

// Better FBM with smoother octaves
float fbm(vec2 st) {
    float value = 0.0;
    float amplitude = 0.5;
    
    // Use prime numbers and irrational numbers to break patterns
    mat2 rotation = mat2(cos(0.5), sin(0.5), -sin(0.5), cos(0.5));
    
    for (int i = 0; i < 6; i++) { // More octaves for smoother result
        value += amplitude * noise(st);
        st = rotation * st * 2.31; // Prime-like number to avoid alignment
        amplitude *= 0.53; // Non-power-of-2 decay
    }
    return value;
}

// Domain warped noise for complex natural patterns
float warpedNoise(vec2 st) {
    vec2 warp = vec2(
        fbm(st + vec2(0.0, 0.0)),
        fbm(st + vec2(5.2, 1.3))
    );
    return fbm(st + warp * 0.3); // Reduced warp strength for smoother result
}

// Multi-octave ridged noise for foam patterns
float ridgedNoise(vec2 st) {
    float n = fbm(st);
    return 1.0 - abs(n * 2.0 - 1.0);
}

// Warped noise for complex patterns
float warpedNoise(vec2 st, float warpStrength) {
    vec2 warp = vec2(fbm(st + vec2(0.0, 0.0)), fbm(st + vec2(5.2, 1.3)));
    return fbm(st + warp * warpStrength);
}

void main() {
    vec3 norm = normalize(fragNormal);
    
    // More natural directional lighting - use light position but normalize distance effects
    vec3 lightDir = normalize(lightPos - fragPosition * 0.001);  // Reduce position influence
    
    vec3 viewDir = normalize(viewPos - fragPosition);

    float waveHeight = fragPosition.y;
    float distanceFromCamera = length(viewPos - fragPosition);
    
    // Enhanced temporal coherence with better distance handling
    float temporalPhase = time * 0.08;  // Slower for more natural movement
    
    // Much more aggressive detail scaling for distance to eliminate patterns
    float detailScale = mix(1.5, 0.15, smoothstep(30.0, 300.0, distanceFromCamera));
    
    // Multi-scale coordinates with better distance-based variation
    vec2 coord1 = fragPosition.xz * 0.004 * detailScale + vec2(cos(temporalPhase), sin(temporalPhase * 1.1)) * 0.2;
    vec2 coord2 = fragPosition.xz * 0.015 * detailScale + vec2(sin(temporalPhase * 1.4), cos(temporalPhase * 0.7)) * 0.4;
    vec2 coord3 = fragPosition.xz * 0.045 * detailScale + vec2(cos(temporalPhase * 0.6), sin(temporalPhase * 1.3)) * 0.6;
    
    // More varied rotation angles that change with distance
    float distanceFactor = smoothstep(0.0, 200.0, distanceFromCamera);
    float angle1 = temporalPhase * 0.15 + 1.2 + distanceFactor * 2.0;
    float angle2 = temporalPhase * 0.12 + 2.3 + distanceFactor * 1.5;
    
    mat2 rot1 = mat2(cos(angle1), sin(angle1), -sin(angle1), cos(angle1));
    mat2 rot2 = mat2(cos(angle2), sin(angle2), -sin(angle2), cos(angle2));
    
    coord2 = rot1 * coord2;
    coord3 = rot2 * coord3;
    
    // Moderate surface patterns for natural ocean
    float surface1 = warpedNoise(coord1) * 0.2;
    float surface2 = fbm(coord2) * 0.15;
    float surface3 = ridgedNoise(coord3) * 0.1;
    
    // Fade surface detail with distance to eliminate far patterns
    float surfaceDetailFade = 1.0 - smoothstep(120.0, 350.0, distanceFromCamera);
    surface1 *= surfaceDetailFade;
    surface2 *= surfaceDetailFade;
    surface3 *= surfaceDetailFade;
    
    // Much more subtle wave-dependent surface detail
    float waveInfluence = smoothstep(-0.6, 0.6, waveHeight);  // Wider range for smoother blending
    float combinedSurface = surface1 + surface2 * waveInfluence * 0.3 + surface3 * (1.0 - waveInfluence) * 0.3;
    
    // Moderate normal perturbation
    float normalStrength = mix(0.2, 0.05, smoothstep(0.0, 180.0, distanceFromCamera));
    vec3 surfaceGradient = vec3(
        dFdx(combinedSurface) * normalStrength,
        1.0,
        dFdy(combinedSurface) * normalStrength
    );
    norm = normalize(norm + surfaceGradient * 0.4);
    
    // Multi-depth water colors for natural variation
    vec3 deepWaterColor = vec3(0.0, 0.15, 0.35);
    vec3 mediumWaterColor = vec3(0.0, 0.18, 0.4);
    vec3 shallowWaterColor = vec3(0.05, 0.25, 0.45);
    vec3 surfaceColor = vec3(0.1, 0.3, 0.5);
    
    // No foam calculations at all - pure water only (foam was causing rendering issues)
    float totalFoam = 0.0;
    
    // Enhanced Fresnel effect for realistic reflection
    float fresnel = pow(1.0 - max(dot(norm, viewDir), 0.0), 2.0);
    fresnel = mix(0.02, 0.92, fresnel);
    
    // Multiple specular highlights for smoother reflection
    vec3 reflectDir = reflect(-lightDir, norm);
    float spec1 = pow(max(dot(viewDir, reflectDir), 0.0), 180.0);
    float spec2 = pow(max(dot(viewDir, reflectDir), 0.0), 45.0) * 0.4;
    float spec3 = pow(max(dot(viewDir, reflectDir), 0.0), 12.0) * 0.2;
    
    // Enhanced caustics for underwater lighting effects
    vec2 causticsCoord = fragPosition.xz * 0.15 * detailScale + temporalPhase * 0.05;
    causticsCoord = rot1 * causticsCoord;
    float caustics = pow(warpedNoise(causticsCoord), 2.2) * 0.4;
    caustics *= smoothstep(0.0, 0.5, waveHeight);
    
    // Enhanced subsurface scattering for depth and translucency
    vec3 subsurface = vec3(0.0, 0.3, 0.7) * max(0.0, dot(-norm, lightDir)) * 0.6;
    subsurface *= (1.0 - smoothstep(0.0, 0.8, waveHeight)); // Less on high waves
    
    // Dynamic water color mixing based on wave height and surface patterns
    // Much more uniform water color mixing - less height-dependent variation
    // Nearly uniform water color - minimal wave height influence
    float depth1 = smoothstep(-2.0, 2.0, waveHeight);  // Very wide range, almost no effect
    float depth2 = smoothstep(-1.5, 1.5, waveHeight);  // Very smooth transitions
    float depth3 = smoothstep(-1.0, 1.0, waveHeight);   // Minimal surface color change
    
    vec3 waterColor = mix(deepWaterColor, mediumWaterColor, depth1 * 0.2);  // Very subtle
    waterColor = mix(waterColor, shallowWaterColor, depth2 * 0.1);  // Almost no effect
    waterColor = mix(waterColor, surfaceColor, depth3 * 0.05);  // Barely visible
    
    // No surface pattern color injection at all
    // waterColor += vec3(combinedSurface * 0.0);
    
    // Minimal surface pattern variation for uniform color
    waterColor += vec3(combinedSurface * 0.002, combinedSurface * 0.003, combinedSurface * 0.002);
    
    // Much more subtle atmospheric perspective - no more white washing
    float fogDistance = smoothstep(300.0, 1000.0, distanceFromCamera);  // Much further distances
    vec3 fogColor = mix(vec3(0.4, 0.55, 0.7), vec3(0.35, 0.5, 0.65), pow(1.0 - fogDistance, 0.8));  // Darker fog
    waterColor = mix(waterColor, fogColor, pow(fogDistance, 2.5) * 0.3);  // Much less fog influence
    
    // Much more subtle and uniform lighting
    vec3 sunlightColor = vec3(1.0, 0.98, 0.95);  // Very subtle warm tint
    float diffuse = max(dot(norm, lightDir), 0.0);
    vec3 ambientLight = vec3(0.4, 0.45, 0.55);  // Higher ambient for more uniform look
    vec3 diffuseLight = diffuse * sunlightColor * 0.5;  // Much more subtle diffuse
    vec3 specularLight = (spec1 + spec2 + spec3) * sunlightColor * fresnel * 0.3;  // Reduce specular
    
    // Rim lighting for water surface highlights
    float rimIntensity = pow(1.0 - dot(norm, viewDir), 2.5) * 0.2;
    vec3 rimColor = vec3(0.15, 0.3, 0.5) * rimIntensity;
    
    // Distance-based surface sparkles
    float sparkles = 0.0;
    if (distanceFromCamera < 180.0) {
        sparkles = pow(fbm(fragPosition.xz * 0.5 + temporalPhase * 0.3), 6.0) * 0.06;
        sparkles *= smoothstep(0.3, 0.8, waveHeight);
        sparkles *= (1.0 - smoothstep(60.0, 180.0, distanceFromCamera));
    }
    
    // Final color assembly with no foam mixing
    vec3 finalColor = waterColor * (ambientLight + diffuseLight) + 
                     specularLight + 
                     subsurface + 
                     rimColor +
                     vec3(sparkles);
    
    // No foam mixing - pure water only
    // vec3 foamWithDetail = vec3(0.8, 0.85, 0.9) * (0.5 + 0.05 * totalFoam);
    // finalColor = mix(finalColor, foamWithDetail, totalFoam * 0.15);
    
    // Consistent transparency
    float alpha = mix(0.85, 0.95, fresnel) + sparkles;
    alpha = clamp(alpha, 0.85, 1.0);
    
    FragColor = vec4(finalColor, alpha);
}
` + "\x00"

func InitShader() Shader {
	return Shader{
		vertexSource:   vertexShaderSource,
		fragmentSource: fragmentShaderSource,
	}
}

func InitWaterShader() Shader {
	return Shader{
		vertexSource:   waterVertexShaderSource,
		fragmentSource: waterFragmentShaderSource,
	}
}
