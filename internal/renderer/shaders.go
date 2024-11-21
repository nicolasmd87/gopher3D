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
}

func (shader *Shader) Use() {
	gl.UseProgram(shader.program)
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

var waterVertexShaderSource = `version 330 core

layout(location = 0) in vec3 inPosition; // Vertex position
layout(location = 1) in vec2 inTexCoord; // Texture Coordinate
layout(location = 2) in vec3 inNormal;   // Normal vector

uniform mat4 model;
uniform mat4 viewProjection;
uniform float time;

// Gerstner Wave parameters
uniform int waveCount;
uniform vec3 waveDirections[5];
uniform float waveAmplitudes[5];
uniform float waveFrequencies[5];
uniform float waveSpeeds[5];

out vec2 fragTexCoord;
out vec3 fragNormal;
out vec3 fragPosition;

void main() {
    vec3 position = inPosition;
    vec3 waveDisplacement = vec3(0.0);

    for (int i = 0; i < waveCount; i++) {
        float theta = dot(vec2(position.x, position.z), waveDirections[i].xz) * waveFrequencies[i] - waveSpeeds[i] * time;
        waveDisplacement.x += waveDirections[i].x * waveAmplitudes[i] * cos(theta);
        waveDisplacement.z += waveDirections[i].z * waveAmplitudes[i] * cos(theta);
        waveDisplacement.y += waveAmplitudes[i] * sin(theta);
    }

    position += waveDisplacement;

    fragPosition = vec3(model * vec4(position, 1.0));
    fragNormal = normalize(mat3(model) * inNormal);
    fragTexCoord = inTexCoord;

    gl_Position = viewProjection * vec4(fragPosition, 1.0);
}
` + "\x00"

var waterFragmentShaderSource = `#version 330 core

in vec2 fragTexCoord;
in vec3 fragNormal;
in vec3 fragPosition;

uniform sampler2D foamTexture;
uniform vec3 lightPos;
uniform vec3 lightColor;
uniform float lightIntensity;
uniform vec3 viewPos;

// Foam parameters
uniform float foamThreshold;

out vec4 FragColor;

void main() {
    vec3 norm = normalize(fragNormal);
    vec3 lightDir = normalize(lightPos - fragPosition);

    // Lighting calculations
    float diff = max(dot(norm, lightDir), 0.0);
    vec3 diffuse = diff * lightColor * lightIntensity;

    vec3 viewDir = normalize(viewPos - fragPosition);
    vec3 reflectDir = reflect(-lightDir, norm);
    float spec = pow(max(dot(viewDir, reflectDir), 0.0), 32.0); // Shininess
    vec3 specular = spec * lightColor * lightIntensity;

    // Foam effect based on height
    float foam = clamp((fragPosition.y / foamThreshold), 0.0, 1.0);
    vec3 foamColor = texture(foamTexture, fragTexCoord).rgb * foam;

    vec3 waterColor = vec3(0.0, 0.3, 0.8);
    vec3 finalColor = waterColor + diffuse + specular + foamColor;

    FragColor = vec4(finalColor, 1.0);
}` + "\x00"

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
