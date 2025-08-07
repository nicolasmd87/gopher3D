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
	skyColor       mgl32.Vec3 // For solid color skybox
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

func (shader *Shader) SetBool(name string, value bool) {
	location := gl.GetUniformLocation(shader.program, gl.Str(name+"\x00"))
	var intValue int32 = 0
	if value {
		intValue = 1
	}
	gl.Uniform1i(location, intValue)
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

    // High-precision world position calculation
    FragPos = vec3(modelMatrix * vec4(inPosition, 1.0));
    
    // Ultra-precise normal calculation for perfect reflections
    // Extract scale factor for compensation
    vec3 scale = vec3(length(modelMatrix[0].xyz), length(modelMatrix[1].xyz), length(modelMatrix[2].xyz));
    mat3 normalMatrix = mat3(modelMatrix) / (scale.x * scale.y * scale.z);
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
    
    return num / denom;
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

// ACES tone mapping for HDR
vec3 ACESFilm(vec3 x) {
    float a = 2.51;
    float b = 0.03;
    float c = 2.43;
    float d = 0.59;
    float e = 0.14;
    return clamp((x*(a*x+b))/(x*(c*x+d)+e), 0.0, 1.0);
}

void main() {
    vec4 texColor = texture(textureSampler, fragTexCoord);
    
    // Pre-calculate expensive operations once
    vec3 tempAdjustedLightColor = light.color * kelvinToRGB(light.temperature);
    vec3 norm = normalize(Normal);
    vec3 viewDir = normalize(viewPos - FragPos);
    
    // Early exit for very dark areas (performance optimization)
    float minLightContribution = 0.001;
    if (light.intensity < minLightContribution && light.ambientStrength < minLightContribution) {
        FragColor = vec4(texColor.rgb * 0.01, texColor.a); // Very dark fallback
        return;
    }
    
    vec3 lightDir;
    float attenuation = 1.0;
    
    // Calculate light direction and attenuation based on light type
    if (light.isDirectional == 1) {
        lightDir = normalize(-light.direction);
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
    
    // Calculate F0 (surface reflection at zero incidence)
    vec3 F0 = vec3(0.04); // Default for dielectrics
    F0 = mix(F0, albedo, metallic);
    
    // Calculate per-light radiance
    vec3 radiance = tempAdjustedLightColor * light.intensity * attenuation;
    
    // High-precision dot products for perfect reflection calculations
    float NdotV = clamp(dot(norm, viewDir), 0.001, 1.0); // Avoid zero division
    float NdotL = clamp(dot(norm, lightDir), 0.0, 1.0);
    float HdotV = clamp(dot(halfwayDir, viewDir), 0.001, 1.0); // Avoid zero division
    
    // Early exit for surfaces facing away from light
    if (NdotL < 0.001) {
        vec3 ambient = light.ambientStrength * tempAdjustedLightColor * albedo;
        vec3 color = ambient * exposure;
        color = ACESFilm(color);
        color = pow(color, vec3(1.0/2.2));
        FragColor = vec4(color, texColor.a);
        return;
    }
    
    // BRDF calculations with optimized dot products
    float NDF = distributionGGX(norm, halfwayDir, roughness);
    float G = geometrySmith(norm, viewDir, lightDir, roughness);
    vec3 F = fresnelSchlick(HdotV, F0);
    
    vec3 kS = F;
    vec3 kD = vec3(1.0) - kS;
    kD *= 1.0 - metallic; // Metallic surfaces don't have diffuse reflection
    
    vec3 numerator = NDF * G * F;
    float denominator = 4.0 * NdotV * NdotL + 0.0001; // Use pre-calculated values
    vec3 specular = numerator / denominator;
    
    // Add to outgoing radiance Lo (using pre-calculated NdotL)
    vec3 Lo = (kD * albedo / 3.14159265359 + specular) * radiance * NdotL;
    
    // Ambient lighting with improved calculation
    vec3 ambient = light.ambientStrength * tempAdjustedLightColor * albedo;
    
    vec3 color = ambient + Lo;
    
    // HDR exposure and tone mapping
    color = color * exposure;
    color = ACESFilm(color);
    
    // Gamma correction (sRGB)
    color = pow(color, vec3(1.0/2.2));
    
    FragColor = vec4(color, texColor.a);
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
uniform vec3 lightDirection;  // Directional light direction for sun-like lighting
uniform vec3 lightColor;
uniform float lightIntensity;
uniform vec3 viewPos;
uniform float time;

// Configurable fog parameters
uniform bool enableFog;
uniform float fogStart;
uniform float fogEnd;
uniform vec3 fogColor;
uniform float fogIntensity;

// Configurable sky parameters
uniform vec3 skyColor;
uniform vec3 horizonColor;

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
    
    // Calculate light direction - use directional light for sun-like lighting
    vec3 lightDir = normalize(-lightDirection);  // Negate for proper lighting direction
    
    vec3 viewDir = normalize(viewPos - fragPosition);

    float waveHeight = fragPosition.y;
    float distanceFromCamera = length(viewPos - fragPosition);
    
    // Much slower temporal movement to eliminate fast-moving patterns
    float temporalPhase = time * 0.02;  // Much slower for natural movement
    
    // Distance-based detail scaling to eliminate far patterns
    float detailScale = mix(1.0, 0.1, smoothstep(50.0, 400.0, distanceFromCamera));
    
    // Static coordinates with minimal temporal movement
    vec2 coord1 = fragPosition.xz * 0.003 * detailScale + vec2(cos(temporalPhase * 0.1), sin(temporalPhase * 0.1)) * 0.05;
    vec2 coord2 = fragPosition.xz * 0.008 * detailScale + vec2(sin(temporalPhase * 0.08), cos(temporalPhase * 0.08)) * 0.03;
    vec2 coord3 = fragPosition.xz * 0.02 * detailScale + vec2(cos(temporalPhase * 0.06), sin(temporalPhase * 0.06)) * 0.02;
    
    // No surface patterns - completely smooth water surface
    float combinedSurface = 0.0;
    
    // Minimal normal perturbation for smooth reflections
    float normalStrength = mix(0.05, 0.01, smoothstep(0.0, 180.0, distanceFromCamera));
    vec3 surfaceGradient = vec3(0.0, 1.0, 0.0); // No surface gradient
    norm = normalize(norm + surfaceGradient * 0.1); // Minimal perturbation
    
    // Single uniform water color - no variations to eliminate all patterns
    vec3 waterColor = vec3(0.08, 0.25, 0.45);  // Single uniform ocean blue
    
    // No foam calculations at all - pure water only
    float totalFoam = 0.0;
    
    // Modern Fresnel effect for realistic water reflections
    float fresnel = pow(1.0 - max(dot(norm, viewDir), 0.0), 2.0);
    fresnel = mix(0.05, 0.4, fresnel); // Natural reflection range for water
    
    // Modern specular highlights using GGX distribution
    vec3 reflectDir = reflect(-lightDir, norm);
    float roughness = 0.1; // Smooth water surface
    float NdotH = max(dot(norm, normalize(lightDir + viewDir)), 0.0);
    float roughnessAlpha = roughness * roughness;
    float alpha2 = roughnessAlpha * roughnessAlpha;
    float denom = NdotH * NdotH * (alpha2 - 1.0) + 1.0;
    float D = alpha2 / (3.14159 * denom * denom);
    
    // Geometry function for water
    float NdotV = max(dot(norm, viewDir), 0.0);
    float NdotL = max(dot(norm, lightDir), 0.0);
    float k = (roughness + 1.0) * (roughness + 1.0) / 8.0;
    float G = NdotV * NdotL / ((NdotV * (1.0 - k) + k) * (NdotL * (1.0 - k) + k));
    
    // Schlick Fresnel for specular
    vec3 F0 = vec3(0.02); // Water has low F0
    vec3 F = F0 + (1.0 - F0) * pow(1.0 - NdotH, 5.0);
    
    // Cook-Torrance BRDF
    vec3 specular = (D * G * F) / (4.0 * NdotV * NdotL + 0.001);
    
    // Completely disable caustics in low light to eliminate dark patches
    float caustics = 0.0;
    if (lightIntensity > 1.3) {
        // Only enable caustics in very bright midday
        vec2 causticsCoord = fragPosition.xz * 0.05 * detailScale + temporalPhase * 0.01;
        caustics = pow(warpedNoise(causticsCoord), 4.0) * 0.1;
        caustics *= smoothstep(0.0, 0.2, waveHeight);
        // Gradual fade-in to prevent sudden appearance
        caustics *= smoothstep(1.3, 1.5, lightIntensity);
    }
    
    // Completely disable subsurface scattering in low light to eliminate dark patches
    vec3 subsurface = vec3(0.0, 0.0, 0.0);
    if (lightIntensity > 1.3) {
        // Only enable subsurface in very bright midday
        subsurface = vec3(0.0, 0.2, 0.5) * max(0.0, dot(-norm, lightDir)) * 0.3;
        subsurface *= (1.0 - smoothstep(0.0, 0.5, waveHeight));
        // Gradual fade-in to prevent sudden appearance
        subsurface *= smoothstep(1.3, 1.5, lightIntensity);
    }
    
    // No surface pattern color injection - clean water surface
    
    // Configurable atmospheric perspective for realistic sky-water transition
    float fogDistance = 0.0;
    vec3 finalFogColor = fogColor;
    
    if (enableFog) {
        fogDistance = smoothstep(fogStart, fogEnd, distanceFromCamera);
        
        // Adaptive fog color based on light intensity
        vec3 fogColor;
        if (lightIntensity < 0.3) {
            // Night - very neutral
            fogColor = vec3(0.3, 0.4, 0.5);
        } else if (lightIntensity < 0.8) {
            // Dawn/Dusk - stronger sky influence to reduce dark patches
            fogColor = mix(vec3(0.4, 0.5, 0.6), skyColor, 0.4);
        } else {
            // Day - neutral
            fogColor = vec3(0.4, 0.5, 0.6);
        }
        
        // Adaptive fog intensity
        float adaptiveFogIntensity = fogIntensity;
        if (lightIntensity < 0.3) {
            adaptiveFogIntensity *= 0.5; // Less fog at night
        } else if (lightIntensity < 0.8) {
            adaptiveFogIntensity *= 0.8; // Moderate fog at dawn/dusk
        }
        
        waterColor = mix(waterColor, fogColor, fogDistance * adaptiveFogIntensity * 0.3);
    }
    
    // Modern PBR lighting for realistic water
    vec3 sunlightColor = vec3(1.0, 0.98, 0.95);  // Natural sunlight
    float diffuse = max(dot(norm, lightDir), 0.0);
    
    // Adaptive ambient lighting based on light intensity and sky color
    vec3 ambientLight;
    if (lightIntensity < 0.3) {
        // Night - very minimal ambient with sky influence
        ambientLight = vec3(0.01, 0.015, 0.03) * lightIntensity * (1.0 + caustics * 0.05);
        ambientLight = mix(ambientLight, skyColor * 0.1, 0.3); // Sky influence at night
    } else if (lightIntensity < 0.8) {
        // Dawn/Dusk - moderate ambient with stronger sky influence
        ambientLight = vec3(0.04, 0.05, 0.1) * lightIntensity * (1.0 + caustics * 0.1);
        ambientLight = mix(ambientLight, skyColor * 0.15, 0.4); // Stronger sky influence
    } else {
        // Day - normal ambient
        ambientLight = vec3(0.05, 0.06, 0.12) * lightIntensity * (1.0 + caustics * 0.15);
    }
    
    // PBR diffuse and specular lighting with sky influence
    vec3 diffuseLight = diffuse * lightColor * lightIntensity * 0.8; // Proper water diffuse
    
    // Sky-influenced specular for realistic reflections
    vec3 skyInfluencedSpecular = mix(sunlightColor, skyColor, 0.3); // 30% sky influence
    vec3 specularLight = specular * skyInfluencedSpecular * lightIntensity * 0.3; // Reduced from 0.6 to 0.3
    
    // Subtle rim lighting
    float rimIntensity = pow(1.0 - dot(norm, viewDir), 3.0) * 0.05;
    vec3 rimColor = vec3(0.1, 0.2, 0.35) * rimIntensity;
    
    // Very subtle surface sparkles
    float sparkles = 0.0;
    if (distanceFromCamera < 100.0) {
        sparkles = pow(fbm(fragPosition.xz * 0.8 + temporalPhase * 0.05), 8.0) * 0.02;
        sparkles *= smoothstep(0.5, 1.0, waveHeight);
        sparkles *= (1.0 - smoothstep(30.0, 100.0, distanceFromCamera));
    }
    
    // Modern PBR final color assembly - proper lighting
    vec3 baseColor = waterColor * (ambientLight + diffuseLight);
    vec3 finalColor = baseColor + 
                     specularLight +           // Additive specular
                     subsurface * 0.4 +        // Subtle subsurface
                     rimColor +
                     vec3(sparkles * 0.3);     // Very subtle sparkles
    
    // Ensure water gets very dark when light intensity is low
    finalColor *= clamp(lightIntensity * 4.0, 0.02, 1.0);
    
    // Very subtle foam mixing - barely visible
    vec3 foamWithDetail = vec3(0.75, 0.8, 0.85) * (0.3 + 0.02 * totalFoam); // More subtle foam color
    finalColor = mix(finalColor, foamWithDetail, totalFoam * 0.08);  // Even more subtle foam visibility
    
    // More natural transparency
    float alpha = mix(0.88, 0.92, fresnel) + totalFoam * 0.08 + sparkles * 0.3;
    alpha = clamp(alpha, 0.88, 0.95);  // Tighter alpha range for more consistent water
    
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
