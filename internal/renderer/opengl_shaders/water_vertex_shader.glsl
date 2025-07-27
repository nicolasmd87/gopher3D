#version 330 core

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
