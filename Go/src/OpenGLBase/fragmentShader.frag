#version 430

in vec3 normal;
in vec3 pos;
in vec3 lightSpacePos;

uniform sampler2D lightDepthTex;

uniform vec3 color;
uniform vec3 light;
uniform bool isLight;

uniform bool shadowEnabled;

out vec4 colorOut;

float linearizeDepth (float depth) {
    float nearPlane = 1.0, farPlane = 200.0;
    return (2.0*nearPlane) / (farPlane + nearPlane - depth * (farPlane - nearPlane));
}

float calcShadow() {
    float camDepth = lightSpacePos.z;

    camDepth = linearizeDepth(camDepth);

    vec2 moments = texture(lightDepthTex, lightSpacePos.xy).rg;

    // Surface is fully lit. as the current fragment is before the light occluder
    if (camDepth <= moments.x)
        return 1.0 ;

    // The fragment is either in shadow or penumbra. We now use chebyshev's upperBound to check
    // How likely this pixel is to be lit (p_max)
    float variance = moments.y - (moments.x*moments.x);
    variance = max(variance,0.001);

    float d = camDepth - moments.x;
    float p_max = variance / (variance + d*d);

    return p_max;
}

void main() {
    colorOut = vec4(color, 1);

    vec3 l = normalize(light - pos);

    vec3 specularColor = color*3;
    float dotProduct = max(dot(normal,l), 0.0);
    vec3 specular = specularColor * pow(dotProduct, 8.0);
    specular = clamp(specular, 0.0, 1.0);

    vec3 diffuseColor = color*2;
    vec3 diffuse  = diffuseColor * max(dot(normal, l), 0);
    diffuse = clamp(diffuse, 0.0, 1.0);

    vec3 diffuseColorNeg = color*3;
    vec3 diffuseNeg  = diffuseColorNeg * max(dot(-normal, l), 0);
    diffuseNeg = clamp(diffuseNeg, 0.0, 1.0);
    diffuseNeg = vec3(1)-diffuseNeg;

    vec3 ambient = color / 1.5;

    if (isLight) {
        colorOut = vec4(color, 1);
    } else {
        colorOut = vec4(diffuseNeg/4 + diffuse/4 + ambient/4 + specular/4 + color/3, 1.0);
    }

    if (shadowEnabled) {
        colorOut *= calcShadow();
    }
}


