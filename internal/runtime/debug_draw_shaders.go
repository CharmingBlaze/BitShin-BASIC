package runtime

// The debug overlay is plain coloured lines: no textures, no lighting.
// View and Proj are the active camera's matrices in G3N world space.

const mbdebugVertex = `#version 330 core
layout(location = 0) in vec3 aPos;
layout(location = 1) in vec4 aColor;

uniform mat4 View;
uniform mat4 Proj;
// Pulls each vertex this fraction of its distance toward the eye. The
// screen position is unchanged, but a wire lying on a mesh surface wins the
// depth test instead of flickering through it.
uniform float Pull;

out vec4 vColor;

void main() {
	vColor = aColor;
	vec4 v = View * vec4(aPos, 1.0);
	v.xyz *= 1.0 - Pull;
	gl_Position = Proj * v;
}
`

const mbdebugFragment = `#version 330 core
in vec4 vColor;
out vec4 FragColor;

uniform float Dim;

void main() {
	FragColor = vec4(vColor.rgb, vColor.a * Dim);
}
`
