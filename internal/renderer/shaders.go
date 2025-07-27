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

// Enhanced Gerstner wave parameters for smoother water
uniform int waveCount;
uniform vec3 waveDirections[5];
uniform float waveAmplitudes[5];
uniform float waveFrequencies[5];
uniform float waveSpeeds[5];

vec3 calculateGerstnerWave(vec3 position, vec3 direction, float amplitude, float frequency, float speed, float time) {
    float phase = dot(direction.xz, position.xz) * frequency + time * speed;
    float sine = sin(phase);
    float cosine = cos(phase);
    
    // Gerstner wave displacement
    vec3 displacement = vec3(0.0);
    displacement.x = direction.x * amplitude * sine;
    displacement.y = amplitude * cosine;
    displacement.z = direction.z * amplitude * sine;
    
    return displacement;
}

vec3 calculateGerstnerNormal(vec3 position, vec3 direction, float amplitude, float frequency, float speed, float time) {
    float phase = dot(direction.xz, position.xz) * frequency + time * speed;
    float sine = sin(phase);
    float cosine = cos(phase);
    
    // Calculate partial derivatives for normal
    float dPhaseDx = direction.x * frequency;
    float dPhaseDz = direction.z * frequency;
    
    float dYdx = -amplitude * frequency * direction.x * sine;
    float dYdz = -amplitude * frequency * direction.z * sine;
    
    return vec3(dYdx, 1.0, dYdz);
}

void main() {
    vec3 worldPos = (model * vec4(aPos, 1.0)).xyz;
    vec3 totalDisplacement = vec3(0.0);
    vec3 totalNormal = vec3(0.0, 1.0, 0.0);
    
    // Enhanced wave calculation with more natural parameters
    for (int i = 0; i < min(waveCount, 5); i++) {
        // Calculate smoother wave displacement
        vec3 waveDisp = calculateGerstnerWave(
            worldPos, 
            waveDirections[i], 
            waveAmplitudes[i], // Use full amplitude from Go code
            waveFrequencies[i], 
            waveSpeeds[i], 
            time
        );
        
        // Calculate normal for this wave
        vec3 waveNormal = calculateGerstnerNormal(
            worldPos,
            waveDirections[i],
            waveAmplitudes[i], // Use full amplitude from Go code
            waveFrequencies[i],
            waveSpeeds[i],
            time
        );
        
        totalDisplacement += waveDisp;
        totalNormal += waveNormal;
    }
    
    // Add subtle high-frequency noise for micro-detail without sharp triangles
    float microNoise = sin(worldPos.x * 0.3 + time * 0.8) * cos(worldPos.z * 0.4 + time * 0.6) * 0.02;
    totalDisplacement.y += microNoise;
    
    // Apply displacement
    worldPos += totalDisplacement;
    
    // Normalize the accumulated normal
    totalNormal = normalize(totalNormal);
    
    fragPosition = worldPos;
    fragTexCoord = aTexCoord;
    fragNormal = totalNormal;
    
    gl_Position = viewProjection * vec4(worldPos, 1.0);
}` + "\x00"

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
    vec3 lightDir = normalize(lightPos - fragPosition);
    vec3 viewDir = normalize(viewPos - fragPosition);

    float waveHeight = fragPosition.y;
    float distanceFromCamera = length(viewPos - fragPosition);
    
    // Adaptive detail based on distance to maintain smoothness
    float detailScale = mix(1.0, 0.3, smoothstep(0.0, 300.0, distanceFromCamera));
    
    // Multi-scale coordinates with smoother scaling
    vec2 coord1 = fragPosition.xz * 0.008 * detailScale + time * 0.02;
    vec2 coord2 = fragPosition.xz * 0.025 * detailScale + time * 0.05;
    vec2 coord3 = fragPosition.xz * 0.06 * detailScale + time * 0.12;
    
    // Use rotation matrices to break grid alignment
    mat2 rot1 = mat2(cos(1.2), sin(1.2), -sin(1.2), cos(1.2));
    mat2 rot2 = mat2(cos(2.3), sin(2.3), -sin(2.3), cos(2.3));
    
    coord2 = rot1 * coord2;
    coord3 = rot2 * coord3;
    
    // Generate smoother surface patterns
    float surface1 = warpedNoise(coord1);
    float surface2 = fbm(coord2) * 0.7;
    float surface3 = noise(coord3) * 0.4;
    
    // Combine with natural weights and smooth transitions
    float combinedSurface = surface1 * 0.5 + surface2 * 0.3 + surface3 * 0.2;
    
    // Add normal perturbation to hide triangle edges
    vec3 surfaceGradient = vec3(
        dFdx(combinedSurface) * 0.1,
        1.0,
        dFdy(combinedSurface) * 0.1
    );
    norm = normalize(norm + surfaceGradient * 0.3);
    
    // Enhanced water colors with smoother transitions
    vec3 deepWaterColor = vec3(0.0, 0.02, 0.12);
    vec3 mediumWaterColor = vec3(0.0, 0.12, 0.28);
    vec3 shallowWaterColor = vec3(0.05, 0.25, 0.45);
    vec3 surfaceColor = vec3(0.1, 0.35, 0.55);
    vec3 foamColor = vec3(0.97, 0.99, 1.0);
    vec3 sunlightColor = vec3(1.0, 0.88, 0.65);
    
    // Smoother foam generation
    vec2 foamCoord1 = fragPosition.xz * 0.03 * detailScale + time * 0.08;
    vec2 foamCoord2 = fragPosition.xz * 0.08 * detailScale + time * 0.15;
    
    // Rotate foam coordinates to break alignment
    foamCoord1 = rot1 * foamCoord1;
    foamCoord2 = rot2 * foamCoord2;
    
    float foam1 = warpedNoise(foamCoord1);
    float foam2 = fbm(foamCoord2) * 0.6;
    
    float foamPattern = foam1 * 0.7 + foam2 * 0.4;
    
    // Much more subtle foam - only on the highest wave crests
    float heightFoam = smoothstep(0.6, 0.9, waveHeight);        // Only on very high waves
    float patternFoam = smoothstep(0.75, 0.95, foamPattern) * heightFoam * 0.4; // Much less intense
    float crestFoam = smoothstep(0.7, 0.95, waveHeight) * (0.2 + 0.3 * foamPattern); // Very subtle
    
    float totalFoam = clamp(heightFoam * 0.3 + patternFoam * 0.5 + crestFoam * 0.4, 0.0, 0.6); // Much less foam overall
    
    // Enhanced Fresnel effect
    float fresnel = pow(1.0 - max(dot(norm, viewDir), 0.0), 2.2);
    fresnel = mix(0.02, 0.95, fresnel);
    
    // Multiple specular highlights for smoother reflection
    vec3 reflectDir = reflect(-lightDir, norm);
    float spec1 = pow(max(dot(viewDir, reflectDir), 0.0), 200.0);
    float spec2 = pow(max(dot(viewDir, reflectDir), 0.0), 64.0) * 0.4;
    float spec3 = pow(max(dot(viewDir, reflectDir), 0.0), 16.0) * 0.2;
    
    // Smoother caustics
    vec2 causticsCoord = fragPosition.xz * 0.12 * detailScale + time * 0.04;
    causticsCoord = rot1 * causticsCoord;
    float caustics = pow(warpedNoise(causticsCoord), 2.5) * 0.35;
    caustics *= smoothstep(0.0, 0.4, waveHeight);
    
    // Enhanced subsurface scattering
    vec3 subsurface = vec3(0.0, 0.25, 0.6) * max(0.0, dot(-norm, lightDir)) * 0.5;
    
    // Smoother water color mixing
    float depth1 = smoothstep(-0.8, 0.0, waveHeight);
    float depth2 = smoothstep(-0.2, 0.4, waveHeight);
    float depth3 = smoothstep(0.1, 0.6, waveHeight);
    
    vec3 waterColor = mix(deepWaterColor, mediumWaterColor, depth1);
    waterColor = mix(waterColor, shallowWaterColor, depth2);
    waterColor = mix(waterColor, surfaceColor, depth3);
    
    // Add subtle surface variation
    waterColor += vec3(combinedSurface * 0.008, combinedSurface * 0.015, combinedSurface * 0.012);
    
    // Enhanced distance fog for smoother far appearance
    float fogDistance = smoothstep(150.0, 800.0, distanceFromCamera);
    vec3 fogColor = mix(vec3(0.6, 0.75, 0.9), vec3(0.4, 0.6, 0.8), pow(1.0 - fogDistance, 0.8));
    waterColor = mix(waterColor, fogColor, pow(fogDistance, 1.5));
    
    // Enhanced lighting
    float diffuse = max(dot(norm, lightDir), 0.0);
    vec3 ambientLight = vec3(0.12, 0.20, 0.30) * (1.0 + caustics * 0.4);
    vec3 diffuseLight = diffuse * lightColor * lightIntensity * 0.8;
    vec3 specularLight = (spec1 + spec2 + spec3) * sunlightColor * lightIntensity * fresnel;
    
    // Smoother rim lighting
    float rimIntensity = pow(1.0 - dot(norm, viewDir), 3.0) * 0.18;
    vec3 rimColor = vec3(0.1, 0.25, 0.4) * rimIntensity;
    
    // Distance-based sparkles for close detail
    float sparkles = 0.0;
    if (distanceFromCamera < 150.0) {
        sparkles = pow(fbm(fragPosition.xz * 0.4 + time * 0.2), 8.0) * 0.04;
        sparkles *= smoothstep(0.2, 0.7, waveHeight);
        sparkles *= (1.0 - smoothstep(50.0, 150.0, distanceFromCamera));
    }
    
    // Final color assembly with smoother blending
    vec3 finalColor = waterColor * (ambientLight + diffuseLight) + 
                     specularLight + 
                     subsurface + 
                     rimColor +
                     vec3(sparkles);
    
    // Smoother foam mixing
    vec3 foamWithDetail = foamColor * (0.9 + 0.1 * foamPattern);
    finalColor = mix(finalColor, foamWithDetail, totalFoam);
    
    // Enhanced transparency
    float alpha = mix(0.88, 0.96, fresnel) + totalFoam * 0.08 + sparkles;
    alpha = clamp(alpha, 0.88, 1.0);
    
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
