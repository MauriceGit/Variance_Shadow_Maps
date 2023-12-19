#version 430

layout (location = 0) in vec3 vertPos;
layout (location = 1) in vec3 vertNormal;

// Shadow mapping
uniform mat4 lightProjectionMat;

// Normal attributes
uniform mat4 viewProjectionMat;
uniform mat4 modelMat;

out vec3 normal;
out vec3 pos;
out vec3 lightSpacePos;

void main() {

    normal = normalize(vertNormal);
    pos = (modelMat * vec4(vertPos,1)).xyz;

    vec4 tmpLightSpacePos = lightProjectionMat * vec4(pos,1);
    lightSpacePos = tmpLightSpacePos.xyz/tmpLightSpacePos.w;
    lightSpacePos = lightSpacePos*0.5 + 0.5;

    gl_Position = viewProjectionMat * modelMat * (vec4(vertPos,1));

}

