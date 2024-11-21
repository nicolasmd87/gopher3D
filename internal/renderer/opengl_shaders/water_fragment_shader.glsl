#version 330 core

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
}
