package runtime

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

const mbRainVertex = `#version 330 core
layout(location = 0) in vec2 aPos;
layout(location = 1) in vec2 aUV;
out vec2 vUV;
void main() {
	vUV = aUV;
	gl_Position = vec4(aPos, 0.0, 1.0);
}
`

const mbRainFragment = `#version 330 core
in vec2 vUV;
out vec4 frag;
uniform float uRain;
uniform float uTime;
uniform float uWet;

float hash(vec2 p) {
	return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453);
}

void main() {
	float n = 0.0;
	for (int i = 0; i < 3; i++) {
		float fi = float(i);
		vec2 cell = vec2(16.0 + fi * 6.0, 7.0 + fi * 2.5);
		vec2 p = vUV * cell;
		p.y += uTime * (2.6 + fi * 0.7);
		vec2 id = floor(p);
		vec2 f = fract(p);
		float h = hash(id + fi * 13.1);
		float drop = smoothstep(0.07, 0.0, abs(f.x - mix(0.2, 0.8, h)))
			* smoothstep(0.62, 0.0, f.y)
			* step(0.55, h);
		n += drop * (0.38 - fi * 0.08);
	}
	float a = n * uRain * clamp(uWet * 1.15, 0.18, 1.0);
	frag = vec4(0.78, 0.84, 0.94, a);
}
`

func (w *World) ensureRainOverlay() error {
	if w.rainProg == 0 {
		p, err := compileLink(
			struct {
				kind uint32
				src  string
			}{gl.VERTEX_SHADER, mbRainVertex},
			struct {
				kind uint32
				src  string
			}{gl.FRAGMENT_SHADER, mbRainFragment},
		)
		if err != nil {
			return err
		}
		w.rainProg = p
	}
	if w.rainBlitOK {
		return nil
	}
	quad := []float32{
		-1, -1, 0, 0,
		1, -1, 1, 0,
		-1, 1, 0, 1,
		1, 1, 1, 1,
	}
	gl.GenVertexArrays(1, &w.rainVAO)
	gl.GenBuffers(1, &w.rainVBO)
	gl.BindVertexArray(w.rainVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, w.rainVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(quad)*4, gl.Ptr(quad), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 4*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, 4*4, gl.PtrOffset(2*4))
	gl.BindVertexArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	w.rainBlitOK = true
	return nil
}

func (w *World) drawCameraRain(ww, hh int) {
	amt := w.cameraRainAmount()
	if amt <= 0.01 || w.app == nil || w.app.Gls() == nil {
		return
	}
	if err := w.ensureRainOverlay(); err != nil {
		fmt.Println("CameraRain:", err)
		return
	}
	gs := w.app.Gls()
	restore := saveModernGL(gs)
	defer restore()
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	if ww > 0 && hh > 0 {
		gl.Viewport(0, 0, int32(ww), int32(hh))
	}
	gl.Disable(gl.DEPTH_TEST)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Disable(gl.CULL_FACE)
	gl.UseProgram(w.rainProg)
	gl.Uniform1f(gl.GetUniformLocation(w.rainProg, gl.Str("uRain\x00")), amt)
	gl.Uniform1f(gl.GetUniformLocation(w.rainProg, gl.Str("uTime\x00")), float32(w.wx.time))
	gl.Uniform1f(gl.GetUniformLocation(w.rainProg, gl.Str("uWet\x00")), w.wetness)
	gl.BindVertexArray(w.rainVAO)
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
	gl.BindVertexArray(0)
	gl.UseProgram(0)
	gl.Disable(gl.BLEND)
	gl.Enable(gl.DEPTH_TEST)
	drainGL("CameraRain")
}
