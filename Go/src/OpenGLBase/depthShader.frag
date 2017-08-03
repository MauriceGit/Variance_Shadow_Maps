#version 430

out vec2 colorOut;

void main() {

    float m1 = gl_FragCoord.z;
    float m2 = m1*m1;

    float dx = dFdx(m1);
    float dy = dFdy(m1);
    m2 += 0.25*(dx*dx+dy*dy);

    colorOut = vec2(m1, m2);
}
