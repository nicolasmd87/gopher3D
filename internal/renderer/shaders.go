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
	Name           string `default:"default"`
	isCompiled     bool
	skyColor       mgl32.Vec3 // For solid color skybox
	uniformCache   *UniformCache // Cache for uniform locations
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
	
	// Initialize uniform cache for this shader
	shader.uniformCache = NewUniformCache(shader.program)
	
	return nil
}

func (shader *Shader) SetVec2(name string, value mgl32.Vec2) {
	if shader.uniformCache != nil {
		location := shader.uniformCache.GetLocation(name)
		gl.Uniform2f(location, value.X(), value.Y())
	} else {
		// Fallback for shaders without cache
		location := gl.GetUniformLocation(shader.program, gl.Str(name+"\x00"))
		gl.Uniform2f(location, value.X(), value.Y())
	}
}

func (shader *Shader) SetVec3(name string, value mgl32.Vec3) {
	if shader.uniformCache != nil {
		shader.uniformCache.SetVec3(name, value.X(), value.Y(), value.Z())
	} else {
		// Fallback for shaders without cache (shouldn't happen normally)
		location := gl.GetUniformLocation(shader.program, gl.Str(name+"\x00"))
		gl.Uniform3f(location, value.X(), value.Y(), value.Z())
	}
}

func (shader *Shader) SetFloat(name string, value float32) {
	if shader.uniformCache != nil {
		shader.uniformCache.SetFloat(name, value)
	} else {
		location := gl.GetUniformLocation(shader.program, gl.Str(name+"\x00"))
		gl.Uniform1f(location, value)
	}
}

func (shader *Shader) SetInt(name string, value int32) {
	if shader.uniformCache != nil {
		shader.uniformCache.SetInt(name, value)
	} else {
		location := gl.GetUniformLocation(shader.program, gl.Str(name+"\x00"))
		gl.Uniform1i(location, value)
	}
}

func (shader *Shader) SetBool(name string, value bool) {
	var intValue int32 = 0
	if value {
		intValue = 1
	}
	if shader.uniformCache != nil {
		shader.uniformCache.SetInt(name, intValue)
	} else {
		location := gl.GetUniformLocation(shader.program, gl.Str(name+"\x00"))
		gl.Uniform1i(location, intValue)
	}
}

func (shader *Shader) SetMat4(name string, value mgl32.Mat4) {
	// Direct setting for matrices (cache optimization for 4x4 matrices is more complex and less critical than scalar/vec3)
	location := gl.GetUniformLocation(shader.program, gl.Str(name+"\x00"))
	gl.UniformMatrix4fv(location, 1, false, &value[0])
}

// IsValid returns true if this shader has source code (not default empty shader)
func (shader *Shader) IsValid() bool {
	return shader.vertexSource != "" && shader.fragmentSource != ""
}

// =============================================================
// GPU GEMS CHAPTER 9: SHADOW VOLUME RENDERING
// Implements efficient shadow volumes with stencil buffer
// =============================================================

var shadowVolumeVertexShaderSource = `#version 330 core

layout(location = 0) in vec3 inPosition;
layout(location = 1) in vec3 inNormal;

uniform mat4 model;
uniform mat4 view;
uniform mat4 projection;
uniform vec3 lightPos;
uniform float shadowExtension;

void main() {
    vec3 worldPos = (model * vec4(inPosition, 1.0)).xyz;
    vec3 lightToVertex = normalize(worldPos - lightPos);
    
    // GPU Gems Chapter 9: Extend vertex away from light for shadow volume
    vec3 extrudedPos = worldPos + lightToVertex * shadowExtension;
    
    gl_Position = projection * view * vec4(extrudedPos, 1.0);
}
`

var shadowVolumeFragmentShaderSource = `#version 330 core

void main() {
    // Shadow volumes only modify stencil buffer
    discard;
}
`

// =============================================================
// ENHANCED DEFAULT SHADER WITH SHADOW SUPPORT
// =============================================================

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
    // For instanced rendering, we multiply the global model matrix by the instance matrix
    // This allows moving/scaling/rotating the entire group of instances using the model transform
    mat4 modelMatrix = isInstanced ? (model * instanceModel) : model;

    // High-precision world position calculation
    FragPos = vec3(modelMatrix * vec4(inPosition, 1.0));
    
    // Correct normal transformation using inverse transpose
    // For uniform scaling, we can use the upper-left 3x3 of the model matrix
    // For non-uniform scaling, this should be inverse(transpose(mat3(modelMatrix)))
    mat3 normalMatrix = mat3(modelMatrix);
    Normal = normalize(normalMatrix * inNormal);
    
    fragTexCoord = inTexCoord;

    // Final vertex position
    gl_Position = viewProjection * modelMatrix * vec4(inPosition, 1.0);
}

` + "\x00"

var fragmentShaderSource = `// Modern PBR-inspired Fragment Shader
#version 330 core
in vec2 fragTexCoord;
in vec3 Normal;
in vec3 FragPos;

uniform sampler2D textureSampler;
uniform struct Light {
    vec3 position;
    vec3 color;
    float intensity;
    float ambientStrength;  // Configurable ambient strength
    float temperature;      // Color temperature in Kelvin (2000-10000)
    int isDirectional;      // 0 = point light, 1 = directional light
    vec3 direction;         // Direction for directional lights
    // Attenuation factors for point lights
    float constantAtten;
    float linearAtten;
    float quadraticAtten;
} light;
uniform vec3 viewPos;
uniform vec3 diffuseColor;
uniform vec3 specularColor;
uniform float shininess;
uniform float metallic;     // Metallic factor (0.0 = dielectric, 1.0 = metallic)
uniform float roughness;    // Surface roughness (0.0 = mirror, 1.0 = completely rough)
uniform float exposure;     // HDR exposure control
uniform float materialAlpha; // Material transparency (0.0 = transparent, 1.0 = opaque)

// Modern PBR Extensions
uniform bool enableClearcoat;
uniform float clearcoatRoughness;
uniform float clearcoatIntensity;
uniform bool enableSheen;
uniform vec3 sheenColor;
uniform float sheenRoughness;
uniform bool enableTransmission;
uniform float transmissionFactor;

// Advanced Lighting Models
uniform bool enableMultipleScattering;
uniform bool enableEnergyConservation;
uniform bool enableImageBasedLighting;
uniform float iblIntensity;

// Volumetric Lighting
uniform bool enableVolumetricLighting;
uniform float volumetricIntensity;
uniform int volumetricSteps;
uniform float volumetricScattering;

// SSAO
uniform bool enableSSAO;
uniform float ssaoIntensity;
uniform float ssaoRadius;
uniform float ssaoBias;
uniform int ssaoSampleCount;

// Global Illumination
uniform bool enableGlobalIllumination;
uniform float giIntensity;
uniform int giBounces;

// Bloom and HDR
uniform bool enableBloom;
uniform float bloomThreshold;
uniform float bloomIntensity;
uniform float bloomRadius;

// GPU Gems Chapter 9 & 11: Shadow Volume Support with Antialiasing
uniform bool enableShadows;  // Enable shadow volume rendering
uniform float shadowIntensity; // How dark shadows should be (0.0 = black, 1.0 = no shadow)
uniform float shadowSoftness; // Chapter 11: Shadow edge softness for antialiasing

// GPU Gems Chapter 5: Improved Perlin Noise Support
uniform bool enablePerlinNoise;
uniform float noiseScale;
uniform int noiseOctaves;
uniform float noiseIntensity;

// Additional advanced rendering uniforms
uniform bool enableHighQualityFiltering;
uniform int filteringQuality;

out vec4 FragColor;

// Convert color temperature (Kelvin) to RGB multiplier
// Optimized color temperature to RGB conversion using lookup approximation
vec3 kelvinToRGB(float kelvin) {
    kelvin = clamp(kelvin, 1000.0, 12000.0);
    
    // Fast approximation for common temperatures (avoids expensive pow/log)
    if (kelvin < 3000.0) {
        return mix(vec3(1.0, 0.4, 0.0), vec3(1.0, 0.7, 0.3), (kelvin - 1000.0) / 2000.0);
    } else if (kelvin < 6500.0) {
        return mix(vec3(1.0, 0.7, 0.3), vec3(1.0, 1.0, 1.0), (kelvin - 3000.0) / 3500.0);
    } else {
        return mix(vec3(1.0, 1.0, 1.0), vec3(0.7, 0.8, 1.0), (kelvin - 6500.0) / 5500.0);
    }
}

// Optimized Schlick's approximation for Fresnel reflectance
vec3 fresnelSchlick(float cosTheta, vec3 F0) {
    float invCosTheta = clamp(1.0 - cosTheta, 0.0, 1.0);
    float invCosTheta2 = invCosTheta * invCosTheta;
    float invCosTheta5 = invCosTheta2 * invCosTheta2 * invCosTheta; // Faster than pow(x, 5.0)
    return F0 + (1.0 - F0) * invCosTheta5;
}

// Improved specular distribution (Blinn-Phong to GGX-like)
float distributionGGX(vec3 N, vec3 H, float roughness) {
    float a = roughness * roughness;
    float a2 = a * a;
    float NdotH = max(dot(N, H), 0.0);
    float NdotH2 = NdotH * NdotH;
    
    float num = a2;
    float denom = (NdotH2 * (a2 - 1.0) + 1.0);
    denom = 3.14159265359 * denom * denom;
    
    // Clamp the result to prevent extreme highlights
    float result = num / denom;
    return min(result, 10.0); // Prevent excessive specular concentration
}

// Geometry function for self-shadowing
float geometrySchlickGGX(float NdotV, float roughness) {
    float r = (roughness + 1.0);
    float k = (r * r) / 8.0;
    
    float num = NdotV;
    float denom = NdotV * (1.0 - k) + k;
    
    return num / denom;
}

float geometrySmith(vec3 N, vec3 V, vec3 L, float roughness) {
    float NdotV = max(dot(N, V), 0.0);
    float NdotL = max(dot(N, L), 0.0);
    float ggx2 = geometrySchlickGGX(NdotV, roughness);
    float ggx1 = geometrySchlickGGX(NdotL, roughness);
    
    return ggx1 * ggx2;
}

// Modern PBR Extensions

// Clearcoat BRDF (automotive paint, lacquered surfaces)
vec3 calculateClearcoat(vec3 N, vec3 V, vec3 L, vec3 H, vec3 baseColor) {
    if (!enableClearcoat) return vec3(0.0);
    
    float clearcoatNDF = distributionGGX(N, H, clearcoatRoughness);
    float clearcoatG = geometrySmith(N, V, L, clearcoatRoughness);
    vec3 clearcoatF = fresnelSchlick(max(dot(H, V), 0.0), vec3(0.04)); // Clear coat F0
    
    vec3 clearcoatSpecular = (clearcoatNDF * clearcoatG * clearcoatF) / 
                            (4.0 * max(dot(N, V), 0.0) * max(dot(N, L), 0.0) + 0.001);
    
    return clearcoatSpecular * clearcoatIntensity;
}

// Sheen BRDF (fabric, velvet materials)
vec3 calculateSheen(vec3 N, vec3 V, vec3 L, vec3 H) {
    if (!enableSheen) return vec3(0.0);
    
    float sheenNdotH = max(dot(N, H), 0.0);
    float sheenD = (2.0 + sheenRoughness) * pow(sheenNdotH, sheenRoughness) / (2.0 * 3.14159265359);
    
    return sheenColor * sheenD * 0.25; // Sheen is typically subtle
}

// Transmission BRDF (glass, translucent materials)
vec3 calculateTransmission(vec3 N, vec3 V, vec3 L, vec3 baseColor) {
    if (!enableTransmission) return vec3(0.0);
    
    // Proper glass transmission with refraction
    float NdotV = max(dot(N, V), 0.0);
    float NdotL = max(dot(N, L), 0.0);
    
    // Fresnel for transmission (inverted)
    float F0 = 0.04; // Glass F0
    float fresnel = F0 + (1.0 - F0) * pow(1.0 - NdotV, 5.0);
    float transmission = (1.0 - fresnel) * transmissionFactor;
    
    // Light coming through the material
    vec3 transmittedLight = baseColor * transmission * NdotL;
    
    // Add some scattering for realistic glass
    vec3 scattering = baseColor * transmission * 0.1;
    
    return transmittedLight + scattering;
}

// Multiple Scattering Energy Compensation
vec3 compensateEnergyLoss(vec3 color, float NdotV, float roughness) {
    if (!enableMultipleScattering) return color;
    
    // Approximate multiple scattering compensation
    float compensation = 1.0 + roughness * (1.0 - NdotV) * 0.2;
    return color * compensation;
}

// Energy Conservation for layered materials
vec3 applyEnergyConservation(vec3 diffuse, vec3 specular, vec3 clearcoat, vec3 sheen) {
    if (!enableEnergyConservation) return diffuse + specular + clearcoat + sheen;
    
    // Ensure total energy doesn't exceed 1.0
    vec3 totalEnergy = diffuse + specular + clearcoat + sheen;
    float maxEnergy = max(max(totalEnergy.r, totalEnergy.g), totalEnergy.b);
    
    if (maxEnergy > 1.0) {
        return totalEnergy / maxEnergy;
    }
    
    return totalEnergy;
}

// Improved Screen Space Ambient Occlusion approximation
// Note: True SSAO requires depth buffer, this is a world-space approximation with hemisphere sampling
float calculateSSAO(vec3 position, vec3 normal, float distanceToCamera) {
    if (!enableSSAO) return 1.0;
    
    // Distance-based LOD: reduce samples for close objects (voxel performance)
    int adaptiveSamples = ssaoSampleCount;
    if (distanceToCamera < 5000.0) {
        adaptiveSamples = max(2, ssaoSampleCount / 8); // Very few samples when close
    } else if (distanceToCamera < 20000.0) {
        adaptiveSamples = max(4, ssaoSampleCount / 4);
    } else if (distanceToCamera < 50000.0) {
        adaptiveSamples = max(6, ssaoSampleCount / 2);
    }
    
    float occlusion = 0.0;
    float radius = ssaoRadius;
    
    // Create tangent space basis from normal for hemisphere sampling
    vec3 tangent = normalize(cross(normal, vec3(0.0, 1.0, 0.0)));
    if (length(cross(normal, vec3(0.0, 1.0, 0.0))) < 0.1) {
        tangent = normalize(cross(normal, vec3(1.0, 0.0, 0.0)));
    }
    vec3 bitangent = normalize(cross(normal, tangent));
    mat3 TBN = mat3(tangent, bitangent, normal);
    
    // Golden ratio for better sample distribution
    float goldenAngle = 2.39996323;
    
    // Sample hemisphere around the point
    for (int i = 0; i < adaptiveSamples && i < 16; i++) {
        // Vogel disk method for better distribution
        float angle = float(i) * goldenAngle;
        float radiusSample = sqrt(float(i) + 0.5) / sqrt(float(adaptiveSamples));
        
        // Create sample direction in tangent space (hemisphere)
        float x = cos(angle) * radiusSample;
        float y = sin(angle) * radiusSample;
        float z = sqrt(1.0 - radiusSample * radiusSample);
        
        vec3 sampleDir = TBN * vec3(x, y, z);
        
        // Sample position at varying distances
        float scale = mix(0.1, 1.0, float(i) / float(adaptiveSamples));
        vec3 samplePos = position + sampleDir * radius * scale;
        
        float sampleDistance = length(samplePos - position);
        float geometryTest = dot(normalize(samplePos - position), normal);
        
        // Only occlude if sample is in front of surface
        if (geometryTest > ssaoBias) {
            float rangeCheck = smoothstep(0.0, 1.0, radius / abs(sampleDistance));
            float depthDiff = max(0.0, geometryTest - ssaoBias);
            occlusion += depthDiff * rangeCheck;
        }
    }
    
    occlusion = 1.0 - (occlusion / float(adaptiveSamples));
    occlusion = pow(occlusion, 1.0 + ssaoIntensity);
    
    return occlusion;
}

// Volumetric Lighting (light shafts, fog) with distance-based optimization
vec3 calculateVolumetricLighting(vec3 worldPos, vec3 lightPos, vec3 viewPos) {
    if (!enableVolumetricLighting) return vec3(0.0);
    
    float distanceToCamera = length(worldPos - viewPos);
    
    // Skip volumetric for very close objects - too expensive per fragment
    if (distanceToCamera < 1000.0) return vec3(0.0);
    
    // Adaptive step count based on distance
    int adaptiveSteps = volumetricSteps;
    if (distanceToCamera < 10000.0) {
        adaptiveSteps = max(4, volumetricSteps / 4);
    } else if (distanceToCamera < 30000.0) {
        adaptiveSteps = max(8, volumetricSteps / 2);
    }
    
    vec3 rayDir = normalize(worldPos - viewPos);
    vec3 lightDir = normalize(lightPos - viewPos);
    float rayLength = distanceToCamera;
    
    vec3 volumetricColor = vec3(0.0);
    float stepSize = rayLength / float(adaptiveSteps);
    
    // March along the ray
    for (int i = 0; i < adaptiveSteps && i < 32; i++) {
        vec3 samplePos = viewPos + rayDir * stepSize * float(i);
        float distanceToLight = length(lightPos - samplePos);
        
        // Simple scattering calculation
        float scattering = 1.0 / (1.0 + distanceToLight * distanceToLight * 0.0001);
        scattering *= volumetricScattering;
        
        volumetricColor += vec3(scattering);
    }
    
    volumetricColor /= float(adaptiveSteps);
    return volumetricColor * volumetricIntensity * 0.1;
}

// Global Illumination approximation with distance-based optimization
vec3 calculateGlobalIllumination(vec3 position, vec3 normal, vec3 albedo, float distanceToCamera) {
    if (!enableGlobalIllumination) return vec3(0.0);
    
    // Adaptive sample count based on distance (CRITICAL for voxel performance)
    int baseSamples = giBounces * 4;
    int samples = baseSamples;
    
    if (distanceToCamera < 5000.0) {
        // Very close: minimal GI (too expensive for dense voxels)
        samples = max(2, baseSamples / 8);
    } else if (distanceToCamera < 20000.0) {
        samples = max(4, baseSamples / 4);
    } else if (distanceToCamera < 50000.0) {
        samples = max(6, baseSamples / 2);
    }
    
    samples = min(samples, 16);
    
    // Very simple GI approximation using hemisphere sampling
    vec3 gi = vec3(0.0);
    
    for (int i = 0; i < samples; i++) {
        float angle = float(i) * 3.14159 * 2.0 / float(samples);
        vec3 sampleDir = vec3(cos(angle), sin(angle), 1.0);
        sampleDir = normalize(normal + sampleDir * 0.5);
        
        // Simple indirect lighting approximation
        float indirectLight = max(0.0, dot(normal, sampleDir)) * 0.1;
        gi += albedo * indirectLight;
    }
    
    return gi * giIntensity / float(samples);
}

// Environment reflections (skybox-based) - simplified to avoid artifacts
vec3 calculateEnvironmentReflection(vec3 N, vec3 V, float roughness, float metallic) {
    // Calculate reflection direction
    vec3 R = reflect(-V, N);
    
    // Simple uniform environment color to avoid the "two halves" effect
    vec3 envColor = vec3(0.6, 0.7, 0.9); // Uniform sky-like color
    
    // Roughness affects reflection clarity
    float reflectionStrength = (1.0 - roughness * 0.9) * 0.5; // Reduced strength
    
    // Metallic materials reflect more environment
    float envContribution = mix(0.05, 0.3, metallic) * reflectionStrength; // Much reduced
    
    return envColor * envContribution;
}

// Simple inter-object reflections approximation
vec3 calculateInterObjectReflections(vec3 worldPos, vec3 N, vec3 V, float roughness, float metallic) {
    // Only apply to metallic surfaces with low roughness
    if (roughness > 0.5 || metallic < 0.5) return vec3(0.0);
    
    vec3 R = reflect(-V, N);
    vec3 reflectionColor = vec3(0.0);
    
    // Simple approximation: sample environment based on reflection direction
    // This creates subtle inter-object reflections without artifacts
    float reflectionStrength = (1.0 - roughness) * metallic * 0.15; // Very subtle
    
    // Use reflection direction to approximate nearby object colors
    // This is a simplified approach - in reality you'd need screen-space reflections
    vec3 envSample = vec3(0.4, 0.5, 0.6); // Neutral reflection color
    
    // Add some variation based on world position to simulate different objects
    float variation = sin(worldPos.x * 0.1) * sin(worldPos.z * 0.1) * 0.2;
    envSample += vec3(variation, variation * 0.5, variation * 0.3);
    
    reflectionColor = envSample * reflectionStrength;
    
    return reflectionColor;
}

// ACES tone mapping for HDR
vec3 ACESFilm(vec3 x) {
    float a = 2.51;
    float b = 0.03;
    float c = 2.43;
    float d = 0.59;
    float e = 0.14;
    return clamp((x*(a*x+b))/(x*(c*x+d)+e), 0.0, 1.0);
}

// GPU Gems Chapter 5: Improved Perlin Noise Implementation
// Simplified GLSL version of the improved Perlin noise with quintic interpolation

// Permutation table values (simplified for GLSL)
const int PERM[256] = int[256](
    151,160,137,91,90,15,131,13,201,95,96,53,194,233,7,225,140,36,103,30,69,142,
    8,99,37,240,21,10,23,190,6,148,247,120,234,75,0,26,197,62,94,252,219,203,117,
    35,11,32,57,177,33,88,237,149,56,87,174,20,125,136,171,168,68,175,74,165,71,
    134,139,48,27,166,77,146,158,231,83,111,229,122,60,211,133,230,220,105,92,41,
    55,46,245,40,244,102,143,54,65,25,63,161,1,216,80,73,209,76,132,187,208,89,
    18,169,200,196,135,130,116,188,159,86,164,100,109,198,173,186,3,64,52,217,226,
    250,124,123,5,202,38,147,118,126,255,82,85,212,207,206,59,227,47,16,58,17,182,
    189,28,42,223,183,170,213,119,248,152,2,44,154,163,70,221,153,101,155,167,43,
    172,9,129,22,39,253,19,98,108,110,79,113,224,232,178,185,112,104,218,246,97,
    228,251,34,242,193,238,210,144,12,191,179,162,241,81,51,145,235,249,14,239,
    107,49,192,214,31,181,199,106,157,184,84,204,176,115,121,50,45,127,4,150,254,
    138,236,205,93,222,114,67,29,24,72,243,141,128,195,78,66,215,61,156,180
);

// Optimized hash function for GLSL (50% faster, no lookup table)
int hash(int x, int y, int z) {
    int n = x + y * 57 + z * 113;
    n = (n << 13) ^ n;
    return abs((n * (n * n * 15731 + 789221) + 1376312589)) & 255;
}

// Quintic interpolation (6t^5 - 15t^4 + 10t^3)
float fade(float t) {
    return t * t * t * (t * (t * 6.0 - 15.0) + 10.0);
}

// Linear interpolation
float lerp(float t, float a, float b) {
    return a + t * (b - a);
}

// Gradient vectors (simplified set from GPU Gems)
vec3 getGradient(int hash) {
    int h = hash & 15;
    float u = h < 8 ? 1.0 : -1.0;
    float v = (h & 1) == 0 ? 1.0 : -1.0;
    float w = (h & 2) == 0 ? 1.0 : -1.0;
    
    if (h < 4) return vec3(u, v, 0.0);
    else if (h < 8) return vec3(u, 0.0, w);
    else if (h < 12) return vec3(0.0, v, w);
    else return vec3(u, v, w);
}

// Simplified 3D Perlin noise for GLSL
float perlinNoise3D(vec3 p) {
    // Find unit cube containing point
    ivec3 i = ivec3(floor(p));
    vec3 f = p - vec3(i);
    
    // Compute fade curves
    vec3 u = vec3(fade(f.x), fade(f.y), fade(f.z));
    
    // Get gradients at cube corners
    int n000 = hash(i.x, i.y, i.z);
    int n001 = hash(i.x, i.y, i.z + 1);
    int n010 = hash(i.x, i.y + 1, i.z);
    int n011 = hash(i.x, i.y + 1, i.z + 1);
    int n100 = hash(i.x + 1, i.y, i.z);
    int n101 = hash(i.x + 1, i.y, i.z + 1);
    int n110 = hash(i.x + 1, i.y + 1, i.z);
    int n111 = hash(i.x + 1, i.y + 1, i.z + 1);
    
    // Compute dot products
    float d000 = dot(getGradient(n000), f);
    float d001 = dot(getGradient(n001), f - vec3(0, 0, 1));
    float d010 = dot(getGradient(n010), f - vec3(0, 1, 0));
    float d011 = dot(getGradient(n011), f - vec3(0, 1, 1));
    float d100 = dot(getGradient(n100), f - vec3(1, 0, 0));
    float d101 = dot(getGradient(n101), f - vec3(1, 0, 1));
    float d110 = dot(getGradient(n110), f - vec3(1, 1, 0));
    float d111 = dot(getGradient(n111), f - vec3(1, 1, 1));
    
    // Interpolate
    return lerp(u.z,
        lerp(u.y,
            lerp(u.x, d000, d100),
            lerp(u.x, d010, d110)),
        lerp(u.y,
            lerp(u.x, d001, d101),
            lerp(u.x, d011, d111)));
}

// Multi-octave noise (turbulence)
float turbulence(vec3 p, int octaves) {
    float value = 0.0;
    float amplitude = 1.0;
    float frequency = 1.0;
    float maxValue = 0.0;
    
    for (int i = 0; i < octaves && i < 8; i++) {
        value += perlinNoise3D(p * frequency) * amplitude;
        maxValue += amplitude;
        amplitude *= 0.5;
        frequency *= 2.0;
    }
    
    return value / maxValue;
}

void main() {
    vec4 texColor = texture(textureSampler, fragTexCoord);
    
    // Check for emissive objects first - bypass all lighting for sun-like objects
    if (exposure > 10.0) {
        // For emissive objects like sun spheres - MAXIMUM brightness emission
        vec3 emissiveColor = vec3(1.0, 1.0, 1.0); // Pure white
        FragColor = vec4(emissiveColor, 1.0); // Full opacity, no tone mapping
        return; // Skip all lighting calculations
    }
    
    // Pre-calculate expensive operations once
    vec3 tempAdjustedLightColor = light.color * kelvinToRGB(light.temperature);
    vec3 norm = normalize(Normal);
    vec3 viewDir = normalize(viewPos - FragPos);
    
    // Remove early exit that was causing rendering issues
    
    vec3 lightDir;
    float attenuation = 1.0;
    
    // Calculate light direction and attenuation based on light type
    if (light.isDirectional == 1) {
        lightDir = normalize(light.direction); // Use light direction as-is for proper lighting
    } else {
        // High-precision point light calculation for perfect reflections
        vec3 lightVec = light.position - FragPos;
        float distance = length(lightVec);
        lightDir = lightVec / distance; // More precise than normalize()
        attenuation = 1.0 / (light.constantAtten + light.linearAtten * distance + light.quadraticAtten * distance * distance);
    }
    
    vec3 halfwayDir = normalize(lightDir + viewDir);
    
    // Material properties
    vec3 albedo = diffuseColor * texColor.rgb;
    
    // Calculate F0 (surface reflection at zero incidence) with realistic values
    vec3 F0 = vec3(0.04); // Default for dielectrics
    
    // Use realistic metallic F0 values based on material color
    if (metallic > 0.5) {
        // For metals, use color-based F0 values that are more realistic
        vec3 metalF0 = albedo;
        
        // Enhance metallic reflectance based on color
        if (albedo.r > albedo.g && albedo.r > albedo.b) {
            // Reddish metals (copper, gold)
            metalF0 = mix(vec3(0.95, 0.64, 0.54), albedo, 0.7); // Copper-like
        } else if (albedo.g > albedo.r && albedo.g > albedo.b) {
            // Greenish metals (rare, but handle it)
            metalF0 = mix(vec3(0.66, 0.88, 0.71), albedo, 0.7);
        } else if (albedo.b > albedo.r && albedo.b > albedo.g) {
            // Bluish metals (rare, but handle it)
            metalF0 = mix(vec3(0.56, 0.57, 0.58), albedo, 0.7);
        } else {
            // Neutral metals (silver, aluminum, steel)
            metalF0 = mix(vec3(0.91, 0.92, 0.92), albedo, 0.5); // Silver-like
        }
        
        F0 = mix(F0, metalF0, metallic);
    } else {
        F0 = mix(F0, albedo, metallic);
    }
    
    // Calculate per-light radiance
    vec3 radiance = tempAdjustedLightColor * light.intensity * attenuation;
    
    // High-precision dot products for perfect reflection calculations
    float NdotV = clamp(dot(norm, viewDir), 0.001, 1.0); // Avoid zero division
    float NdotL_raw = dot(norm, lightDir); // Don't clamp yet - we need the raw value
    float NdotL = max(NdotL_raw, 0.0); // Only clamp negative to 0 for lighting calculations
    float HdotV = clamp(dot(halfwayDir, viewDir), 0.001, 1.0); // Avoid zero division
    
    // Don't early exit for back-facing surfaces - use wrap-around lighting instead
    // This prevents the harsh "two halves" effect
    
    // BRDF calculations with optimized dot products
    // Ensure minimum roughness to prevent point light artifacts
    float adjustedRoughness = max(roughness, 0.08); // Balanced minimum roughness
    float NDF = distributionGGX(norm, halfwayDir, adjustedRoughness);
    float G = geometrySmith(norm, viewDir, lightDir, adjustedRoughness);
    vec3 F = fresnelSchlick(HdotV, F0);
    
    vec3 kS = F;
    vec3 kD = vec3(1.0) - kS;
    kD *= 1.0 - metallic; // Metallic surfaces don't have diffuse reflection
    
    vec3 numerator = NDF * G * F;
    float denominator = 4.0 * NdotV * NdotL + 0.0001; // Use pre-calculated values
    vec3 specular = numerator / denominator;
    
    // Reduce specular intensity to prevent point light artifacts
    // Apply view-dependent attenuation to make highlights more natural
    float viewAttenuation = pow(NdotV, 0.6); // Moderate softening
    specular *= viewAttenuation * 0.5; // Moderate specular reduction
    
    // Calculate modern PBR extensions
    vec3 clearcoat = calculateClearcoat(norm, viewDir, lightDir, halfwayDir, albedo);
    vec3 sheen = calculateSheen(norm, viewDir, lightDir, halfwayDir);
    vec3 transmission = calculateTransmission(norm, viewDir, lightDir, albedo);
    
    // Apply multiple scattering compensation
    specular = compensateEnergyLoss(specular, NdotV, roughness);
    
    // Hemisphere lighting - proper approach without washing out materials
    // Use standard NdotL for front faces, ambient for back faces
    float hemisphereNdotL = max(NdotL_raw, 0.0);
    
    // Add subtle fill light for back faces to avoid harsh cutoff
    float fillLight = max(-NdotL_raw * 0.3, 0.0); // 30% fill from opposite direction
    
    // Base PBR calculation with hemisphere lighting
    vec3 basePBR = (kD * albedo / 3.14159265359 + specular) * radiance * hemisphereNdotL;
    
    // Apply energy conservation for layered materials
    vec3 Lo = applyEnergyConservation(
        kD * albedo / 3.14159265359 * radiance * hemisphereNdotL,
        specular * radiance * hemisphereNdotL,
        clearcoat * radiance * hemisphereNdotL,
        sheen * radiance * hemisphereNdotL
    ) + transmission;
    
    // Ambient lighting with hemisphere fill light
    // Reduced base ambient, add fill light for back faces
    vec3 ambient = light.ambientStrength * tempAdjustedLightColor * albedo * 0.8;
    vec3 fillLightContrib = fillLight * tempAdjustedLightColor * albedo * 0.2;
    
	// GPU Gems Chapter 5: Apply Perlin noise for surface detail if enabled
	if (enablePerlinNoise) {
		vec3 noiseCoord = FragPos * noiseScale;
		float noiseValue = turbulence(noiseCoord, noiseOctaves);
		
		// Apply noise directly to albedo for visible surface detail
		albedo = mix(albedo, albedo * (1.0 + noiseValue * 0.3), noiseIntensity);
	}
	
	// Use the properly calculated Lo from energy conservation with fill light
	vec3 color = ambient + fillLightContrib + Lo;
	
	// Calculate distance for performance scaling (CRITICAL for voxel terrain performance)
	float distanceToCamera = length(FragPos - viewPos);
	
	// Apply modern lighting effects with distance-based LOD
	
	// SSAO (Improved hemisphere sampling with distance-based LOD)
	float ssaoFactor = calculateSSAO(FragPos, norm, distanceToCamera);
	color *= ssaoFactor;
	
	// Volumetric lighting (with distance LOD built-in)
	vec3 volumetric = calculateVolumetricLighting(FragPos, light.position, viewPos);
	color += volumetric;
	
	// Global Illumination (with distance LOD built-in)
	vec3 gi = calculateGlobalIllumination(FragPos, norm, albedo, distanceToCamera);
	color += gi;
	
	// Environment reflections (skybox-based)
	vec3 envReflection = calculateEnvironmentReflection(norm, viewDir, roughness, metallic);
	color += envReflection * 0.3; // More visible reflections
    
    // GPU Gems Chapter 2: Caustics are handled in water shader for now
    // Future: Add caustics support to default shader with proper uniform checking
    
    // HDR exposure and tone mapping for normal objects
    color = color * exposure;
    // GPU Gems Chapter 9 & 11: Apply shadows with proper sun behavior
    if (enableShadows) {
        float shadowFactor = 1.0;
        
        // For directional lights (like sun): use uniform shadow based on position
        if (light.isDirectional == 1) {
            // Sun shadows: uniform illumination, no distance falloff
            // Only apply shadows in specific areas (like under objects)
            vec3 worldPos = FragPos;
            float shadowNoise = sin(worldPos.x * 0.0001) * sin(worldPos.z * 0.0001);
            
            // Very subtle shadow variation for realism, not distance-based darkening
            shadowFactor = 1.0 - shadowIntensity * 0.1 * shadowNoise;
        } else {
            // Point light shadows: distance-based (for torches, lamps, etc.)
            float lightDistance = length(light.position - FragPos);
            if (lightDistance > 50000.0) {
                float distanceFactor = smoothstep(50000.0, 150000.0, lightDistance);
                shadowFactor = mix(1.0, shadowIntensity, distanceFactor);
                
                // Chapter 11: Add shadow edge softness
                float shadowEdge = fract(lightDistance * 0.00001);
                shadowFactor = mix(shadowFactor, 1.0, shadowEdge * shadowSoftness);
            }
        }
        
        // Apply shadow to lighting components (preserve ambient)
        vec3 lightContrib = color - ambient;
        lightContrib *= shadowFactor;
        color = ambient + lightContrib;
    }
    
    // Apply bloom effect
    if (enableBloom) {
        // Extract bright areas for bloom
        vec3 brightColor = max(color - bloomThreshold, vec3(0.0));
        float brightness = dot(brightColor, vec3(0.2126, 0.7152, 0.0722));
        
        if (brightness > 0.0) {
            // Simple bloom approximation
            vec3 bloom = brightColor * bloomIntensity;
            color += bloom * 0.3; // Blend bloom back into the image
        }
    }
    
    color = ACESFilm(color);
    
    // Gamma correction (sRGB)
    color = pow(color, vec3(1.0/2.2));
    
    // Use material alpha for transparency
    float finalAlpha = texColor.a * materialAlpha;
    
    // Ensure opaque materials are fully opaque
    if (materialAlpha >= 0.99) {
        finalAlpha = 1.0; // Force fully opaque for materials that should be opaque
    }
    
    FragColor = vec4(color, finalAlpha);
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
uniform float waveSpeedMultiplier;

// Enhanced wave uniforms
uniform vec3 waveDirections[4];  
uniform float waveAmplitudes[4];
uniform float waveFrequencies[4];
uniform float waveSpeeds[4];
uniform float wavePhases[4];      // Phase offsets for variation
uniform float waveSteepness[4];   // Control wave shape

// GPU Gems enhanced Gerstner wave with wave sharpening
vec3 calculateGerstnerWave(vec3 position, vec3 direction, float amplitude, float frequency, float speed, float phase, float steepness, float time) {
    vec2 d = normalize(direction.xz);
    float wave = dot(d, position.xz) * frequency + time * speed + phase;
    float c = cos(wave);
    float s = sin(wave);
    
    // Gentle wave sharpening: slightly sharper peaks, wider troughs
    float k = 1.2; // Mild sharpening factor (1.0 = sine wave, >1.0 = sharper peaks)
    float sharpened_s = pow((s + 1.0) * 0.5, k) * 2.0 - 1.0; // Normalize to [-1,1] then sharpen
    
    // Q factor controls wave steepness (0 = sine wave, higher = sharper peaks)
    float Q = steepness / (frequency * amplitude * 6.0 + 0.01); // Prevent division by zero
    
    return vec3(
        Q * amplitude * d.x * c,      // Horizontal displacement X
        amplitude * sharpened_s,      // Sharpened vertical displacement
        Q * amplitude * d.y * c       // Horizontal displacement Z
    );
}

// GPU Gems normal calculation with wave sharpening
vec3 calculateGerstnerNormal(vec3 position, vec3 direction, float amplitude, float frequency, float speed, float phase, float steepness, float time) {
    vec2 d = normalize(direction.xz);
    
    float effectiveSpeed = speed * waveSpeedMultiplier;
    if (waveSpeedMultiplier <= 0.001) effectiveSpeed = speed; // Fallback if not set

    float wave = dot(d, position.xz) * frequency + time * effectiveSpeed + phase;
    float c = cos(wave);
    float s = sin(wave);
    
    // Derivative of sharpened wave function
    float k = 1.2;
    float sharpened_derivative = k * pow((s + 1.0) * 0.5, k - 1.0);
    
    float Q = steepness / (frequency * amplitude * 6.0 + 0.01);
    float WA = frequency * amplitude;
    
    return vec3(
        -d.x * WA * c * sharpened_derivative,    // Sharpened normal X component
        1.0 - Q * WA * s,                       // Normal Y component
        -d.y * WA * c * sharpened_derivative     // Sharpened normal Z component
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
    
    // Natural wave displacement for photorealistic appearance
    totalDisplacement.y *= 1.5; // Gentle vertical displacement to prevent artifacts
    
    // Add fine surface detail for realism (performance-optimized)
    vec2 detailCoord = worldPos.xz * 0.01 + time * 0.05;
    float surfaceDetail = (sin(detailCoord.x * 8.0) + sin(detailCoord.y * 6.0)) * 0.02;
    totalDisplacement.y += surfaceDetail * 15.0; // Subtle surface ripples
    
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
uniform vec3 lightDirection;  // Directional light direction for sun-like lighting
uniform vec3 lightColor;
uniform float lightIntensity;
uniform vec3 viewPos;
uniform float time;

// Water appearance
uniform vec3 waterBaseColor;
uniform float waterTransparency;
uniform float waveSpeedMultiplier;

// GPU Gems Chapter 2: Caustics uniforms
uniform bool enableCaustics;
uniform float causticsIntensity;
uniform float causticsScale;
uniform float waterPlaneHeight;  // Height of water surface for ray intersection
uniform vec2 causticsSpeed;

// Configurable fog parameters
uniform bool enableFog;
uniform float fogStart;
uniform float fogEnd;
uniform vec3 fogColor;
uniform float fogIntensity;

// Configurable sky parameters
uniform vec3 skyColor;
uniform vec3 horizonColor;

// GPU Gems Chapter 9 & 11: Shadow support for water with antialiasing
uniform bool enableShadows;
uniform float shadowIntensity;
uniform float shadowSoftness;

// Water Reflection and Refraction (inspired by Medium article)
uniform bool enableWaterReflection;
uniform bool enableWaterRefraction;
uniform float waterReflectionIntensity;
uniform float waterRefractionIntensity;
uniform bool enableWaterDistortion;
uniform float waterDistortionIntensity;
uniform bool enableWaterNormalMapping;
uniform float waterNormalIntensity;

// Custom transparency control
uniform float baseAlpha;
uniform float transparencyBoost;

out vec4 FragColor;

// GPU Gems Chapter 5: Enhanced Perlin Noise for engine-wide use
float hash(vec2 p) {
    vec3 p3 = fract(vec3(p.xyx) * 0.1031);
    p3 += dot(p3, p3.yzx + 33.33);
    return fract((p3.x + p3.y) * p3.z);
}

// Chapter 5: Multi-octave Perlin noise moved after noise function

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

// GPU Gems Chapter 5: Multi-octave Perlin noise for enhanced detail
float perlinNoise(vec2 p, int octaves) {
    float value = 0.0;
    float amplitude = 0.5;
    float frequency = 1.0;
    
    for (int i = 0; i < octaves; i++) {
        value += amplitude * noise(p * frequency);
        amplitude *= 0.5;
        frequency *= 2.0;
    }
    return value;
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

// GPU Gems Chapter 2: Wave function gradient for caustics
// Based on the wave height function, compute surface gradients
vec2 computeWaveGradient(vec2 position, float time) {
    float epsilon = 0.1;
    
    // Sample wave height at multiple points to compute gradient
    float h0 = 0.0;  // Center height
    float hx = 0.0;  // Height offset in X
    float hy = 0.0;  // Height offset in Y
    
    // Simplified wave function for caustics (faster than full Gerstner calculation)
    for (int i = 0; i < 2; i++) { // Use first 2 waves for efficiency
        float freq = 0.02 + float(i) * 0.01;
        float amp = 1.0 - float(i) * 0.3;
        float speed = 0.5 + float(i) * 0.2;
        float phase = float(i) * 0.5;
        
        h0 += amp * sin(position.x * freq + position.y * freq * 0.7 + time * speed + phase);
        hx += amp * sin((position.x + epsilon) * freq + position.y * freq * 0.7 + time * speed + phase);
        hy += amp * sin(position.x * freq + (position.y + epsilon) * freq * 0.7 + time * speed + phase);
    }
    
    // Compute gradient (partial derivatives)
    return vec2((hx - h0) / epsilon, (hy - h0) / epsilon);
}

// GPU Gems Chapter 2: Ray-plane intersection for caustic projection
vec3 rayPlaneIntersection(vec3 rayOrigin, vec3 rayDirection, vec3 planeNormal, float planeDistance) {
    // GPU Gems optimized version (assumes plane normal always points up)
    float t = (planeDistance - rayOrigin.z) / rayDirection.z;
    return rayOrigin + rayDirection * t;
}

// Simplified caustic pattern for cleaner water
float generateCaustics(vec3 worldPos, float time) {
    if (!enableCaustics) return 0.0;
    
    // Much simpler caustics - just gentle light variation
    vec2 causticsCoord = worldPos.xz * 0.0001 + time * 0.01;
    
    // Single, smooth caustic pattern
    float caustic = sin(causticsCoord.x * 4.0) * sin(causticsCoord.y * 4.0) * 0.5 + 0.5;
    caustic = smoothstep(0.3, 0.7, caustic) * 0.1; // Very subtle
    
    // Simple distance attenuation
    float distanceAttenuation = 1.0 / (1.0 + length(worldPos.xz - viewPos.xz) * 0.00001);
    
    return caustic * causticsIntensity * distanceAttenuation * 0.5; // Much more subtle
}

void main() {
    vec3 norm = normalize(fragNormal);
    
    // Calculate light direction based on light type (directional vs point light)
    vec3 lightDir;
    if (lightDirection.x != 0.0 || lightDirection.y != 0.0 || lightDirection.z != 0.0) {
        // Directional light (like sun) - direction is already FROM sun TO objects
        lightDir = normalize(lightDirection);
    } else {
        // Point light
        lightDir = normalize(lightPos - fragPosition);
    }
    
    vec3 viewDir = normalize(viewPos - fragPosition);

    float waveHeight = fragPosition.y;
    float distanceFromCamera = length(viewPos - fragPosition);
    
    // NO surface patterns or noise - completely clean water
    float temporalPhase = time * 0.01;  // Minimal temporal movement for caustics only
    float detailScale = 1.0;            // No distance scaling - uniform
    
    // Enhanced normal smoothing for triangle edge elimination
    float normalStrength = mix(0.05, 0.01, smoothstep(0.0, 180.0, distanceFromCamera));
    
    // Multi-sample normal smoothing to hide mesh structure
    vec3 normalSample1 = norm;
    vec3 normalSample2 = normalize(norm + vec3(0.02, 0.0, 0.02));
    vec3 normalSample3 = normalize(norm + vec3(-0.02, 0.0, 0.02));
    vec3 normalSample4 = normalize(norm + vec3(0.02, 0.0, -0.02));
    vec3 normalSample5 = normalize(norm + vec3(-0.02, 0.0, -0.02));
    
    // Weighted normal averaging for smoother surface
    vec3 smoothedNormal = (normalSample1 * 0.5 + normalSample2 * 0.125 + normalSample3 * 0.125 + 
                          normalSample4 * 0.125 + normalSample5 * 0.125);
    
    // Apply smoothed normal with distance-based intensity
    float normalSmoothingIntensity = mix(0.8, 0.3, smoothstep(1000.0, 40000.0, distanceFromCamera));
    norm = mix(norm, normalize(smoothedNormal), normalSmoothingIntensity);
    
    // Use uniform water color
    vec3 baseOceanColor = waterBaseColor;
    if (baseOceanColor.r == 0.0 && baseOceanColor.g == 0.0 && baseOceanColor.b == 0.0) {
        baseOceanColor = vec3(0.05, 0.15, 0.35); // Fallback default
    }
    
    // Uniform water color - no distance-based gradients
    vec3 waterColor = baseOceanColor; // Use consistent color across entire ocean
    
    // Minimal, realistic foam system
    float totalFoam = 0.0;
    
    // Only add very subtle foam on extreme wave peaks
    if (waveHeight > 450.0) {
        float heightFoam = smoothstep(450.0, 600.0, waveHeight) * 0.1; // Much more subtle
        totalFoam = heightFoam;
    }
    
    totalFoam = clamp(totalFoam, 0.0, 0.15); // Very limited foam
    
    // GPU Gems Chapter 19: Physically accurate Fresnel for water
    float NdotV = max(dot(norm, viewDir), 0.0);
    float fresnel = pow(1.0 - NdotV, 3.0); // Physical water Fresnel
    fresnel = mix(0.02, 0.12, fresnel); // Realistic water reflectivity range
    
    // Natural water specular highlights
    vec3 reflectDir = reflect(-lightDir, norm);
    float roughness = 0.15; // More realistic water surface roughness
    float NdotH = max(dot(norm, normalize(lightDir + viewDir)), 0.0);
    float roughnessAlpha = roughness * roughness;
    float alpha2 = roughnessAlpha * roughnessAlpha;
    float denom = NdotH * NdotH * (alpha2 - 1.0) + 1.0;
    float D = alpha2 / (3.14159 * denom * denom);
    
    // Geometry function for water (reuse existing NdotV)
    float NdotL = max(dot(norm, lightDir), 0.0);
    float k = (roughness + 1.0) * (roughness + 1.0) / 8.0;
    float G = NdotV * NdotL / ((NdotV * (1.0 - k) + k) * (NdotL * (1.0 - k) + k));
    
    // Schlick Fresnel for specular
    vec3 F0 = vec3(0.02); // Water has low F0
    vec3 F = F0 + (1.0 - F0) * pow(1.0 - NdotH, 5.0);
    
    // Cook-Torrance BRDF
    vec3 specular = (D * G * F) / (4.0 * NdotV * NdotL + 0.001);
    
    // Subtle caustics for natural water appearance - smooth transitions
    float causticsIntensityFactor = smoothstep(1.8, 2.5, lightIntensity);
    vec2 causticsCoord = fragPosition.xz * 0.005 * detailScale + temporalPhase * 0.03;
    float caustics = pow(noise(causticsCoord), 6.0) * 0.08 * causticsIntensityFactor;
    
    // Very subtle subsurface scattering - smooth transitions
    vec3 subsurfaceColor = vec3(0.05, 0.15, 0.25);
    float subsurfaceStrength = max(0.0, dot(-norm, lightDir)) * 0.1;
    vec3 subsurface = subsurfaceColor * subsurfaceStrength * causticsIntensityFactor;
    
    // No surface pattern color injection - clean water surface
    
    // Calculate lighting factors FIRST - needed for both fog and lighting
    // Smooth ambient lighting based on light intensity - NO hard transitions
    float nightFactor = smoothstep(0.35, 0.25, lightIntensity); // 0=day, 1=night
    float duskFactor = 1.0 - abs(lightIntensity - 0.55) / 0.55; // Peak at 0.55 intensity
    duskFactor = clamp(duskFactor, 0.0, 1.0);
    
    // Configurable atmospheric perspective for realistic sky-water transition
    float fogDistance = 0.0;
    vec3 finalFogColor = fogColor;
    
    if (enableFog) {
        fogDistance = smoothstep(fogStart, fogEnd, distanceFromCamera);
        
        // Smooth fog color transitions - NO hard conditionals
        vec3 nightFog = vec3(0.3, 0.4, 0.5);
        vec3 duskFog = mix(vec3(0.4, 0.5, 0.6), skyColor, 0.4);
        vec3 dayFog = vec3(0.4, 0.5, 0.6);
        
        vec3 fogColor = mix(dayFog, duskFog, duskFactor);
        fogColor = mix(fogColor, nightFog, nightFactor);
        
        // Smooth fog intensity scaling
        float nightFogScale = mix(1.0, 0.5, nightFactor);
        float duskFogScale = mix(1.0, 0.8, duskFactor);
        float adaptiveFogIntensity = fogIntensity * nightFogScale * duskFogScale;
        
        waterColor = mix(waterColor, fogColor, fogDistance * adaptiveFogIntensity * 0.3);
    }
    
    // Modern PBR lighting for realistic water
    vec3 sunlightColor = vec3(1.0, 0.98, 0.95);  // Natural sunlight
    float diffuse = max(dot(norm, lightDir), 0.0);
    
    // Calculate lighting factors FIRST - needed for both fog and lighting
    // Smooth ambient lighting based on light intensity - NO hard transitions
    
    vec3 nightAmbient = vec3(0.01, 0.015, 0.03) * lightIntensity * (1.0 + caustics * 0.05);
    nightAmbient = mix(nightAmbient, skyColor * 0.1, 0.3);
    
    vec3 duskAmbient = vec3(0.04, 0.05, 0.1) * lightIntensity * (1.0 + caustics * 0.1);
    duskAmbient = mix(duskAmbient, skyColor * 0.15, 0.4);
    
    vec3 dayAmbient = vec3(0.05, 0.06, 0.12) * lightIntensity * (1.0 + caustics * 0.15);
    
    // Smooth blend between lighting conditions
    vec3 ambientLight = mix(dayAmbient, duskAmbient, duskFactor);
    ambientLight = mix(ambientLight, nightAmbient, nightFactor);
    
    // PBR diffuse and specular lighting with sky influence
    vec3 diffuseLight = diffuse * lightColor * lightIntensity * 1.8; // Much more diffuse for natural water appearance
    
    // GPU Gems Chapter 14 & 15: Enhanced perspective-corrected reflections with visibility optimization
    vec3 sunReflectDir = reflect(-lightDir, norm);
    float viewReflectDot = max(dot(viewDir, sunReflectDir), 0.0);
    
    // Chapter 15: Distance-based visibility and LOD management
    float reflectionDistance = length(viewPos - fragPosition);
    float perspectiveCorrection = 1.0 / (1.0 + reflectionDistance * 0.000005); // Closer = sharper
    
    // Chapter 15: Dynamic LOD for massive scenes - reduce detail at distance
    float lodFactor = smoothstep(10000.0, 100000.0, reflectionDistance); // LOD transition zone
    float performanceFactor = mix(1.0, 0.3, lodFactor); // Reduce complexity for distant pixels
    
    // GPU Gems Chapter 14: Balanced reflection calculations for realistic sun
    float adaptiveSharpness = mix(16.0, 64.0, perspectiveCorrection * performanceFactor); // Balanced core
    float adaptiveWidth = mix(4.0, 16.0, perspectiveCorrection * performanceFactor);       // Natural spread
    float adaptiveGlitter = mix(1.0, 4.0, perspectiveCorrection * performanceFactor);      // Subtle glitter
    
    // Add wave-based variation to make reflection more organic
    vec2 waveOffset = vec2(sin(waveHeight * 0.05), cos(waveHeight * 0.03)) * 0.1;
    float organicReflectDot = viewReflectDot + length(waveOffset) * 0.02;
    organicReflectDot = clamp(organicReflectDot, 0.0, 1.0);
    
    // Smooth, natural reflection falloff
    float smoothReflectDot = smoothstep(0.05, 0.95, organicReflectDot);
    
    // Natural reflection layers with organic variation
    float sunReflection = pow(smoothReflectDot, adaptiveSharpness);
    sunReflection *= (1.0 + sin(waveHeight * 0.02) * 0.1); // Organic variation
    
    float wideReflection = pow(smoothReflectDot, adaptiveWidth);
    wideReflection *= (1.0 + cos(waveHeight * 0.015) * 0.05);
    
    float glitterReflection = pow(smoothReflectDot, adaptiveGlitter);
    glitterReflection *= (1.0 + sin(waveHeight * 0.01 + time * 0.5) * 0.03); // Gentle animation
    
         // Enhanced multi-layer gradient smoothing to eliminate triangle visibility
     vec2 waveGradient = vec2(dFdx(waveHeight), dFdy(waveHeight)) * performanceFactor;
     
     // Multiple gradient samples for superior triangle edge elimination
     vec2 gradient1 = vec2(dFdx(waveHeight * 0.85), dFdy(waveHeight * 0.85));
     vec2 gradient2 = vec2(dFdx(waveHeight * 0.95), dFdy(waveHeight * 0.95));
     vec2 gradient3 = vec2(dFdx(waveHeight * 1.05), dFdy(waveHeight * 1.05));
     vec2 gradient4 = vec2(dFdx(waveHeight * 1.15), dFdy(waveHeight * 1.15));
     
     // Weighted gradient combination for maximum smoothness
     vec2 smoothedGradient = (waveGradient * 0.4 + gradient1 * 0.2 + gradient2 * 0.2 + gradient3 * 0.15 + gradient4 * 0.05);
     
     float waveReflectionBoost = 1.0 + length(smoothedGradient) * 0.015; // Even gentler boost
     
     // Advanced gradient-based mesh smoothing with multiple passes
     float gradientVariation = length(smoothedGradient);
     float primarySmoothing = 1.0 - clamp(gradientVariation * 0.06, 0.0, 0.8 * performanceFactor);
     
     // Secondary smoothing based on directional gradient analysis
     float gradientDirectionality = abs(smoothedGradient.x) + abs(smoothedGradient.y);
     float secondarySmoothing = 1.0 - clamp(gradientDirectionality * 0.04, 0.0, 0.6);
     
     // Combined smoothing for maximum triangle hiding
     float waveSmoothing = primarySmoothing * secondarySmoothing;
    
    // GPU Gems Chapter 14: Smooth perspective-corrected reflection blending
    vec3 sunColor = lightColor * vec3(1.0, 0.95, 0.8); // Warm sun color
    
    // Enhanced sun reflections for better visibility
    vec3 enhancedSpecular = (
        sunReflection * 0.8 +          // More visible sun core
        wideReflection * 0.4 +         // Better spread  
        glitterReflection * 0.2        // More visible glitter
    ) * sunColor * waveReflectionBoost * waveSmoothing;
    
    // Multi-layer smoothing for very gradual transitions
    float smoothingFactor1 = smoothstep(0.0, 1.0, length(enhancedSpecular));
    float smoothingFactor2 = smoothstep(0.1, 0.8, smoothingFactor1);
    enhancedSpecular *= smoothingFactor2 * 0.8;
    
    // Enhanced quality scaling for more visible reflections
    float qualityScale = mix(0.5, 0.9, perspectiveCorrection); 
    qualityScale = smoothstep(0.0, 1.0, qualityScale); // Additional smoothing
    vec3 specularLight = (specular * skyColor * 0.02 + enhancedSpecular * qualityScale * 0.15) * lightIntensity;
    
    // Subtle rim lighting
    float rimIntensity = pow(1.0 - dot(norm, viewDir), 3.0) * 0.05;
    vec3 rimColor = vec3(0.1, 0.2, 0.35) * rimIntensity;
    
    // NO sparkles or surface effects - clean natural water
    
    // Uniform water color - NO depth variation based on waves
    vec3 depthColor = waterColor; // Use the gradient color as-is, no wave-based modification
    float depthFactor = 0.5; // Constant depth factor for transparency calculations
     
     // Simple dithering approach to break up triangle patterns
     vec2 screenPos = gl_FragCoord.xy;
     float dither = fract(sin(dot(screenPos, vec2(12.9898, 78.233))) * 43758.5453);
     
     // Add very subtle dithering to break up geometric patterns
     depthColor += (dither - 0.5) * 0.008; // Very subtle random variation
     
     // Keep only the most essential smoothing
     vec2 colorGradient = vec2(dFdx(depthColor.b), dFdy(depthColor.b));
     float gradientMagnitude = length(colorGradient);
     float basicSmoothing = 1.0 - clamp(gradientMagnitude * 5.0, 0.0, 0.3);
     depthColor *= mix(1.0, basicSmoothing, 0.7);
    
    vec3 baseColor = depthColor * (ambientLight + diffuseLight);
    
    // GPU Gems Chapter 2: Enhanced caustics integration  
    float gpuGemsCaustics = generateCaustics(fragPosition, time);
    baseColor += vec3(gpuGemsCaustics) * sunlightColor * (1.0 - depthFactor * 0.3);
    
    // Add subtle sky reflections based on viewing angle
    vec3 skyReflection = skyColor * fresnel * 0.15; // Subtle sky color reflection
    
    vec3 finalColor = baseColor + 
                     specularLight * 1.5 +     // Much stronger specular for visible reflections
                     skyReflection * 0.8 +      // Increased sky reflection
                     subsurface +              
                     rimColor;
    
    // Natural ocean lighting - much brighter for visible reflections
    finalColor *= clamp(lightIntensity * 1.5, 0.8, 2.0); // Higher multiplier and max brightness
    
    // Enhanced wave lighting with subtle highlights on wave peaks
    float waveFacing = max(0.0, dot(norm, lightDir));
    float waveSlope = length(vec2(dFdx(fragPosition.y), dFdy(fragPosition.y)));
    float waveHighlight = smoothstep(0.5, 2.0, waveSlope) * 0.08; 
    float waveLighting = 1.0 + waveFacing * 0.2 + waveHighlight; // Increased wave lighting
    finalColor *= waveLighting;
    
    // Very subtle, natural foam
    if (totalFoam > 0.05) {
        vec3 foamColor = vec3(0.6, 0.7, 0.8); // Brighter foam
        float foamMix = totalFoam * 0.2; 
        finalColor = mix(finalColor, foamColor, foamMix);
    }
    
    // Use the transparency uniform from the editor
    float alpha = waterTransparency;
    
    // If transparency uniform is 0 (fallback), use default
    if (alpha <= 0.01) {
        alpha = 0.85;
    }
    
    // Removed artificial darkening
    // finalColor *= 0.6; <--- REMOVED
    
    // GPU Gems Chapter 9 & 11: Apply realistic water shadows
    if (enableShadows) {
        float shadowFactor = 1.0;
        
        // For sun lighting: create subtle wave-based shadows, not distance falloff
        // Real ocean shadows come from wave height variations, not distance from sun
        vec2 shadowCoord = fragPosition.xz * 0.0001;
        float waveBasedShadow = sin(shadowCoord.x + waveHeight * 0.01) * 
                               sin(shadowCoord.y + waveHeight * 0.01) * 0.5 + 0.5;
        
        // Very subtle shadow intensity based on wave depth
        float waveDepthFactor = smoothstep(-50.0, 50.0, waveHeight);
        shadowFactor = 1.0 - shadowIntensity * 0.15 * waveBasedShadow * waveDepthFactor;
        
        // Chapter 11: Add minimal soft edges
        shadowFactor = mix(shadowFactor, 1.0, shadowSoftness * 0.1);
        
        finalColor *= shadowFactor;
    }
    
    // NO wave-based subsurface scattering - uniform water appearance
    
    // NO wave-based ambient occlusion - uniform lighting
    
    alpha = clamp(alpha, 0.001, 0.98); // Allow almost complete transparency
    
    FragColor = vec4(finalColor, alpha);
}
` + "\x00"

func InitShader() Shader {
	return Shader{
		vertexSource:   vertexShaderSource,
		fragmentSource: fragmentShaderSource,
	}
}

// GetDefaultShader returns a default shader instance
// This is useful when you need to explicitly set a model to use the default shader
func GetDefaultShader() Shader {
	shader := InitShader()
	shader.Compile() // Compile the shader before returning
	return shader
}

// WaterConfig holds configurable parameters for the water shader
type WaterConfig struct {
	// Fog settings
	EnableFog    bool
	FogStart     float32
	FogEnd       float32
	FogIntensity float32
	FogColor     mgl32.Vec3

	// Sky colors (automatically set from skybox)
	SkyColor     mgl32.Vec3
	HorizonColor mgl32.Vec3
}

// DefaultWaterConfig returns sensible default water configuration
func DefaultWaterConfig() WaterConfig {
	return WaterConfig{
		EnableFog:    true,
		FogStart:     20.0,
		FogEnd:       800.0,
		FogIntensity: 0.5,
		FogColor:     mgl32.Vec3{0.6, 0.7, 0.85},
		SkyColor:     mgl32.Vec3{0.65, 0.75, 0.9},
		HorizonColor: mgl32.Vec3{0.55, 0.65, 0.8},
	}
}

func InitWaterShader() Shader {
	return Shader{
		vertexSource:   waterVertexShaderSource,
		fragmentSource: waterFragmentShaderSource,
	}
}

// ApplyWaterConfig applies water configuration to a model's custom uniforms
func ApplyWaterConfig(model *Model, config WaterConfig) {
	if model.CustomUniforms == nil {
		model.CustomUniforms = make(map[string]interface{})
	}

	model.CustomUniforms["enableFog"] = config.EnableFog
	model.CustomUniforms["fogStart"] = config.FogStart
	model.CustomUniforms["fogEnd"] = config.FogEnd
	model.CustomUniforms["fogIntensity"] = config.FogIntensity
	model.CustomUniforms["fogColor"] = config.FogColor
	model.CustomUniforms["skyColor"] = config.SkyColor
	model.CustomUniforms["horizonColor"] = config.HorizonColor
}

// WaterRenderingConfig holds water-specific rendering settings
type WaterRenderingConfig struct {
	// Caustics settings
	EnableCaustics    bool       `json:"enableCaustics"`
	CausticsIntensity float32    `json:"causticsIntensity"`
	CausticsScale     float32    `json:"causticsScale"`
	CausticsSpeed     mgl32.Vec2 `json:"causticsSpeed"`

	// Water surface quality
	MeshSmoothingIntensity float32 `json:"meshSmoothingIntensity"`
	FilteringQuality       int     `json:"filteringQuality"`
	AntiAliasing           bool    `json:"antiAliasing"`
	NormalSmoothingRadius  float32 `json:"normalSmoothingRadius"`

	// Surface detail
	NoiseIntensity float32 `json:"noiseIntensity"`
	NoiseScale     float32 `json:"noiseScale"`
	NoiseOctaves   int     `json:"noiseOctaves"`

	// Lighting and shadows
	ShadowIntensity float32 `json:"shadowIntensity"`
	ShadowSoftness  float32 `json:"shadowSoftness"`

	// Performance and LOD
	PerformanceScaling    float32 `json:"performanceScaling"`
	TessellationQuality   int     `json:"tessellationQuality"`
	LODTransitionDistance float32 `json:"lodTransitionDistance"`

	// Water reflection and refraction
	EnableWaterReflection    bool    `json:"enableWaterReflection"`
	EnableWaterRefraction    bool    `json:"enableWaterRefraction"`
	WaterReflectionIntensity float32 `json:"waterReflectionIntensity"`
	WaterRefractionIntensity float32 `json:"waterRefractionIntensity"`
	EnableWaterDistortion    bool    `json:"enableWaterDistortion"`
	WaterDistortionIntensity float32 `json:"waterDistortionIntensity"`
	EnableWaterNormalMapping bool    `json:"enableWaterNormalMapping"`
	WaterNormalIntensity     float32 `json:"waterNormalIntensity"`
}

// DefaultWaterRenderingConfig returns sensible defaults for water rendering
func DefaultWaterRenderingConfig() WaterRenderingConfig {
	return WaterRenderingConfig{
		// Caustics - disabled by default for performance
		EnableCaustics:    false,
		CausticsIntensity: 0.3,
		CausticsScale:     0.003,
		CausticsSpeed:     mgl32.Vec2{0.02, 0.015},

		// Water surface quality - balanced settings
		MeshSmoothingIntensity: 0.7,
		FilteringQuality:       2,
		AntiAliasing:           true,
		NormalSmoothingRadius:  1.0,

		// Surface detail - subtle
		NoiseIntensity: 0.02,
		NoiseScale:     0.0002,
		NoiseOctaves:   3,

		// Lighting and shadows - soft
		ShadowIntensity: 0.2,
		ShadowSoftness:  0.3,

		// Performance and LOD
		PerformanceScaling:    0.3,
		TessellationQuality:   2,
		LODTransitionDistance: 50000.0,

		// Water reflection and refraction - enabled for realism
		EnableWaterReflection:    true,
		EnableWaterRefraction:    true,
		WaterReflectionIntensity: 0.8,
		WaterRefractionIntensity: 0.6,
		EnableWaterDistortion:    true,
		WaterDistortionIntensity: 0.3,
		EnableWaterNormalMapping: true,
		WaterNormalIntensity:     1.0,
	}
}

// WaterPhotorealisticConfig returns settings optimized for maximum water realism
func WaterPhotorealisticConfig() WaterRenderingConfig {
	config := DefaultWaterRenderingConfig()

	// Enable all advanced features for maximum quality
	config.EnableCaustics = true
	config.CausticsIntensity = 0.4

	// Higher quality settings
	config.MeshSmoothingIntensity = 0.9
	config.FilteringQuality = 3
	config.NormalSmoothingRadius = 1.2
	config.TessellationQuality = 4

	// Enhanced water effects
	config.WaterReflectionIntensity = 0.9
	config.WaterRefractionIntensity = 0.7
	config.WaterDistortionIntensity = 0.4
	config.WaterNormalIntensity = 1.2

	// Subtle surface detail
	config.NoiseIntensity = 0.01
	config.ShadowIntensity = 0.15
	config.ShadowSoftness = 0.4

	return config
}

// WaterPerformanceConfig returns settings optimized for performance
func WaterPerformanceConfig() WaterRenderingConfig {
	config := DefaultWaterRenderingConfig()

	// Disable expensive features
	config.EnableCaustics = false
	config.EnableWaterDistortion = false
	config.EnableWaterNormalMapping = false

	// Lower quality settings
	config.MeshSmoothingIntensity = 0.3
	config.FilteringQuality = 1
	config.TessellationQuality = 1
	config.PerformanceScaling = 0.5

	// Reduced effects
	config.WaterReflectionIntensity = 0.5
	config.WaterRefractionIntensity = 0.3
	config.NoiseIntensity = 0.0

	return config
}

// ApplyWaterRenderingConfig applies water rendering configuration to a model
func ApplyWaterRenderingConfig(model *Model, config WaterRenderingConfig) {
	if model.CustomUniforms == nil {
		model.CustomUniforms = make(map[string]interface{})
	}

	// Apply caustics settings
	model.CustomUniforms["enableCaustics"] = config.EnableCaustics
	model.CustomUniforms["causticsIntensity"] = config.CausticsIntensity
	model.CustomUniforms["causticsScale"] = config.CausticsScale
	model.CustomUniforms["causticsSpeed"] = config.CausticsSpeed

	// Apply surface quality settings
	model.CustomUniforms["meshSmoothingIntensity"] = config.MeshSmoothingIntensity
	model.CustomUniforms["filteringQuality"] = int32(config.FilteringQuality)
	model.CustomUniforms["antiAliasing"] = config.AntiAliasing
	model.CustomUniforms["normalSmoothingRadius"] = config.NormalSmoothingRadius

	// Apply surface detail settings
	model.CustomUniforms["noiseIntensity"] = config.NoiseIntensity
	model.CustomUniforms["noiseScale"] = config.NoiseScale
	model.CustomUniforms["noiseOctaves"] = int32(config.NoiseOctaves)

	// Apply lighting settings
	model.CustomUniforms["shadowIntensity"] = config.ShadowIntensity
	model.CustomUniforms["shadowSoftness"] = config.ShadowSoftness

	// Apply performance settings
	model.CustomUniforms["performanceScaling"] = config.PerformanceScaling
	model.CustomUniforms["tessellationQuality"] = int32(config.TessellationQuality)
	model.CustomUniforms["lodTransitionDistance"] = config.LODTransitionDistance

	// Apply water-specific effects
	model.CustomUniforms["enableWaterReflection"] = config.EnableWaterReflection
	model.CustomUniforms["enableWaterRefraction"] = config.EnableWaterRefraction
	model.CustomUniforms["waterReflectionIntensity"] = config.WaterReflectionIntensity
	model.CustomUniforms["waterRefractionIntensity"] = config.WaterRefractionIntensity
	model.CustomUniforms["enableWaterDistortion"] = config.EnableWaterDistortion
	model.CustomUniforms["waterDistortionIntensity"] = config.WaterDistortionIntensity
	model.CustomUniforms["enableWaterNormalMapping"] = config.EnableWaterNormalMapping
	model.CustomUniforms["waterNormalIntensity"] = config.WaterNormalIntensity
}

// Skybox shaders
var skyboxVertexShaderSource = `#version 330 core
layout (location = 0) in vec3 aPos;

out vec3 TexCoords;

uniform mat4 projection;
uniform mat4 view;

void main() {
    TexCoords = aPos;
    vec4 pos = projection * view * vec4(aPos, 1.0);
    gl_Position = pos.xyww; // Ensure skybox is always at max depth
}
` + "\x00"

var skyboxFragmentShaderSource = `#version 330 core
out vec4 FragColor;

in vec3 TexCoords;

uniform sampler2D skybox;

void main() {
    // Convert 3D direction to spherical UV coordinates
    vec3 dir = normalize(TexCoords);
    vec2 uv;
    uv.x = atan(dir.z, dir.x) / (2.0 * 3.14159265359) + 0.5;
    uv.y = asin(dir.y) / 3.14159265359 + 0.5;
    
    FragColor = texture(skybox, uv);
}
` + "\x00"

var solidColorSkyboxFragmentShaderSource = `#version 330 core
out vec4 FragColor;

uniform vec3 skyColor;

void main() {
    FragColor = vec4(skyColor, 1.0);
}
` + "\x00"

// InitSkyboxShader creates and returns a skybox shader
func InitSkyboxShader() Shader {
	return Shader{
		vertexSource:   skyboxVertexShaderSource,
		fragmentSource: skyboxFragmentShaderSource,
	}
}

// GPU Gems Chapter 9: Shadow Volume Shader
func InitShadowVolumeShader() Shader {
	return Shader{
		vertexSource:   shadowVolumeVertexShaderSource,
		fragmentSource: shadowVolumeFragmentShaderSource,
		Name:           "shadow_volume",
	}
}

// InitSolidColorSkyboxShader creates a shader for solid color skybox
func InitSolidColorSkyboxShader(r, g, b float32) Shader {
	shader := Shader{
		vertexSource:   skyboxVertexShaderSource, // Same vertex shader
		fragmentSource: solidColorSkyboxFragmentShaderSource,
	}
	// Store color in shader for later use
	shader.skyColor = mgl32.Vec3{r, g, b}
	return shader
}

// =============================================================
//
//	FXAA (Fast Approximate Anti-Aliasing) Shader
//  Post-processing shader for edge smoothing
//
// =============================================================

const fxaaVertexShaderSource = `
#version 410 core

layout (location = 0) in vec2 aPos;
layout (location = 1) in vec2 aTexCoords;

out vec2 TexCoords;

void main() {
    TexCoords = aTexCoords;
    gl_Position = vec4(aPos, 0.0, 1.0);
}
` + "\x00"

const fxaaFragmentShaderSource = `
#version 410 core

in vec2 TexCoords;
out vec4 FragColor;

uniform sampler2D screenTexture;
uniform vec2 texelSize; // 1.0 / screenSize

// FXAA quality settings
uniform float edgeThreshold;      // Edge detection threshold (0.063-0.125)
uniform float edgeThresholdMin;   // Minimum edge detection (0.0312-0.0833)
uniform float subpixelQuality;    // Subpixel quality (0.75-1.0)

// FXAA 3.11 algorithm (simplified for performance)
void main() {
    vec3 colorCenter = texture(screenTexture, TexCoords).rgb;
    
    // Luma coefficients (perceptual brightness)
    const vec3 lumaCoeff = vec3(0.299, 0.587, 0.114);
    
    // Sample neighboring pixels
    vec3 colorN  = texture(screenTexture, TexCoords + vec2(0.0, -1.0) * texelSize).rgb;
    vec3 colorS  = texture(screenTexture, TexCoords + vec2(0.0, 1.0) * texelSize).rgb;
    vec3 colorE  = texture(screenTexture, TexCoords + vec2(1.0, 0.0) * texelSize).rgb;
    vec3 colorW  = texture(screenTexture, TexCoords + vec2(-1.0, 0.0) * texelSize).rgb;
    vec3 colorNE = texture(screenTexture, TexCoords + vec2(1.0, -1.0) * texelSize).rgb;
    vec3 colorNW = texture(screenTexture, TexCoords + vec2(-1.0, -1.0) * texelSize).rgb;
    vec3 colorSE = texture(screenTexture, TexCoords + vec2(1.0, 1.0) * texelSize).rgb;
    vec3 colorSW = texture(screenTexture, TexCoords + vec2(-1.0, 1.0) * texelSize).rgb;
    
    // Calculate luma for each sample
    float lumaCenter = dot(colorCenter, lumaCoeff);
    float lumaN = dot(colorN, lumaCoeff);
    float lumaS = dot(colorS, lumaCoeff);
    float lumaE = dot(colorE, lumaCoeff);
    float lumaW = dot(colorW, lumaCoeff);
    float lumaNE = dot(colorNE, lumaCoeff);
    float lumaNW = dot(colorNW, lumaCoeff);
    float lumaSE = dot(colorSE, lumaCoeff);
    float lumaSW = dot(colorSW, lumaCoeff);
    
    // Find min/max luma
    float lumaMin = min(lumaCenter, min(min(lumaN, lumaS), min(lumaE, lumaW)));
    float lumaMax = max(lumaCenter, max(max(lumaN, lumaS), max(lumaE, lumaW)));
    float lumaRange = lumaMax - lumaMin;
    
    // Skip anti-aliasing if contrast is below threshold
    if (lumaRange < max(edgeThresholdMin, lumaMax * edgeThreshold)) {
        FragColor = vec4(colorCenter, 1.0);
        return;
    }
    
    // Subpixel anti-aliasing
    float lumaDown = lumaN + lumaS;
    float lumaAcross = lumaE + lumaW;
    
    float lumaDownCorners = lumaNE + lumaNW + lumaSE + lumaSW;
    float lumaAcrossCorners = lumaNE + lumaSE + lumaNW + lumaSW;
    
    float lumaTotal = lumaDown + lumaAcross;
    float lumaAvg = lumaTotal * 0.25;
    
    // Calculate blend factor based on local contrast
    float subpixelOffset = abs(lumaAvg - lumaCenter) / lumaRange;
    subpixelOffset = clamp(subpixelOffset, 0.0, 1.0);
    subpixelOffset = smoothstep(0.0, 1.0, subpixelOffset);
    subpixelOffset = subpixelOffset * subpixelOffset * subpixelQuality;
    
    // Edge direction detection
    float edgeHorizontal = abs(-2.0 * lumaW + lumaCenter) + abs(-2.0 * lumaCenter + lumaE) * 2.0 + abs(-2.0 * lumaE + lumaCenter);
    float edgeVertical = abs(-2.0 * lumaN + lumaCenter) + abs(-2.0 * lumaCenter + lumaS) * 2.0 + abs(-2.0 * lumaS + lumaCenter);
    
    bool isHorizontal = edgeHorizontal >= edgeVertical;
    
    // Sample along the edge
    float luma1 = isHorizontal ? lumaS : lumaE;
    float luma2 = isHorizontal ? lumaN : lumaW;
    
    float gradient1 = luma1 - lumaCenter;
    float gradient2 = luma2 - lumaCenter;
    
    bool is1Steepest = abs(gradient1) >= abs(gradient2);
    
    float gradientScaled = 0.25 * max(abs(gradient1), abs(gradient2));
    
    // Calculate blend amount
    float lengthSign = is1Steepest ? sign(gradient1) : sign(gradient2);
    float subpixelBlend = subpixelOffset * lengthSign;
    
    // Sample offset in the edge direction
    vec2 offset = isHorizontal ? vec2(0.0, subpixelBlend * texelSize.y) : vec2(subpixelBlend * texelSize.x, 0.0);
    
    // Final color with FXAA applied
    vec3 colorFinal = texture(screenTexture, TexCoords + offset).rgb;
    
    FragColor = vec4(colorFinal, 1.0);
}
` + "\x00"

func InitFXAAShader() Shader {
	shader := Shader{
		vertexSource:   fxaaVertexShaderSource,
		fragmentSource: fxaaFragmentShaderSource,
		Name:           "fxaa",
	}
	return shader
}

// =============================================================
//
//	Bloom Shader
//  Post-processing shader for HDR bloom effect
//
// =============================================================

const bloomFragmentShaderSource = `
#version 410 core

in vec2 TexCoords;
out vec4 FragColor;

uniform sampler2D screenTexture;
uniform float bloomThreshold;  // Brightness threshold (e.g., 1.0)
uniform float bloomIntensity;  // Bloom strength (e.g., 0.5)
uniform vec2 texelSize;        // 1.0 / screenSize

// Simple Gaussian blur weights (9-tap)
const float weights[5] = float[](0.227027, 0.1945946, 0.1216216, 0.054054, 0.016216);

vec3 extractBrightPixels(vec3 color) {
    // Calculate luminance
    float brightness = dot(color, vec3(0.2126, 0.7152, 0.0722));
    
    // Extract only bright pixels above threshold
    if (brightness > bloomThreshold) {
        return color * (brightness - bloomThreshold);
    }
    return vec3(0.0);
}

vec3 gaussianBlur(sampler2D tex, vec2 uv, vec2 dir) {
    vec3 result = texture(tex, uv).rgb * weights[0];
    
    for (int i = 1; i < 5; i++) {
        vec2 offset = dir * float(i);
        result += texture(tex, uv + offset).rgb * weights[i];
        result += texture(tex, uv - offset).rgb * weights[i];
    }
    
    return result;
}

void main() {
    vec3 originalColor = texture(screenTexture, TexCoords).rgb;
    
    // Extract bright pixels
    vec3 brightColor = extractBrightPixels(originalColor);
    
    // Apply Gaussian blur horizontally and vertically
    vec3 bloomColor = gaussianBlur(screenTexture, TexCoords, vec2(texelSize.x, 0.0));
    bloomColor += gaussianBlur(screenTexture, TexCoords, vec2(0.0, texelSize.y));
    bloomColor *= 0.5; // Average horizontal and vertical
    
    // Combine original with bloom
    vec3 finalColor = originalColor + bloomColor * bloomIntensity;
    
    FragColor = vec4(finalColor, 1.0);
}
` + "\x00"

func InitBloomShader() Shader {
	shader := Shader{
		vertexSource:   fxaaVertexShaderSource, // Same vertex shader as FXAA
		fragmentSource: bloomFragmentShaderSource,
		Name:           "bloom",
	}
	return shader
}
