#version 430

layout (location = 0) in vec3 vertPos;

// Normal attributes
uniform mat4 viewProjectionMat;
uniform mat4 modelMat;

void main() {
    gl_Position = viewProjectionMat * modelMat * (vec4(vertPos,1));
}

