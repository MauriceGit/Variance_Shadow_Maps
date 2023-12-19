#version 430

out vec2 colorOut;

float linearizeDepth (float depth) {
    float nearPlane = 1.0, farPlane = 200.0;
    return (2.0*nearPlane) / (farPlane + nearPlane - depth * (farPlane - nearPlane));
}

void main() {

    float m1 = gl_FragCoord.z;

    m1 = linearizeDepth(m1);

    float m2 = m1*m1;

    float dx = dFdx(m1);
    float dy = dFdy(m1);
    m2 += 0.25*(dx*dx+dy*dy);

    colorOut = vec2(m1, m2);
}
