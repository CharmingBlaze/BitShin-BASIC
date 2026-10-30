package runtime

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// gpuFFT draws a 32×32 inverse DFT of the Phillips spectrum into a texture.
// WaterHeight still uses the CPU tile. The vertex shader samples this texture.
const fftFragment = `#version 330 core
uniform float Time;
out vec4 FragColor;
void main() {
    const float N = 32.0;
    const float G = 9.81;
    const float WIND = 8.0;
    float px = gl_FragCoord.x - 0.5;
    float pz = gl_FragCoord.y - 0.5;
    float sum = 0.0;
    for (int z = 0; z < 32; z++) {
        for (int x = 0; x < 32; x++) {
            float kx = float(x) - 16.0;
            float kz = float(z) - 16.0;
            float k = length(vec2(kx, kz));
            if (k < 0.5) { continue; }
            float phillips = exp(-1.0 / (k * k)) / (k * k * k * k);
            phillips *= kx * kx + kz * kz;
            float amp = sqrt(phillips) * 0.35;
            float omega = sqrt(G * k / WIND);
            float phase = omega * Time;
            float ang = 6.2831853 * (float(x) * px + float(z) * pz) / N;
            sum += (amp * cos(phase)) * cos(ang) - (amp * sin(phase)) * sin(ang);
        }
    }
    float h = sum / (N * N);
    FragColor = vec4(clamp(h / 4.0 + 0.5, 0.0, 1.0), 0.0, 0.0, 1.0);
}
`

const fftVertex = `#version 330 core
layout(location = 0) in vec2 aPos;
void main() {
    gl_Position = vec4(aPos, 0.0, 1.0);
}
`

func (w *World) drawGPUFFT(wb *waterBody, t float32) uint32 {
	if w == nil || wb == nil {
		return 0
	}
	if w.fftProg == 0 {
		if err := w.initGPUFFT(); err != nil {
			return 0
		}
	}
	if wb.fftGL == 0 || wb.fftFBO == 0 {
		var tex, fbo uint32
		gl.GenTextures(1, &tex)
		gl.BindTexture(gl.TEXTURE_2D, tex)
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, 32, 32, 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
		gl.GenFramebuffers(1, &fbo)
		gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, tex, 0)
		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
		gl.BindTexture(gl.TEXTURE_2D, 0)
		wb.fftGL = tex
		wb.fftFBO = fbo
	}
	var vp [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &vp[0])
	depth := gl.IsEnabled(gl.DEPTH_TEST)
	gl.Disable(gl.DEPTH_TEST)
	gl.BindFramebuffer(gl.FRAMEBUFFER, wb.fftFBO)
	gl.Viewport(0, 0, 32, 32)
	gl.UseProgram(w.fftProg)
	loc := gl.GetUniformLocation(w.fftProg, gl.Str("Time\x00"))
	gl.Uniform1f(loc, t)
	gl.BindVertexArray(w.fftVAO)
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
	gl.BindVertexArray(0)
	gl.UseProgram(0)
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.Viewport(vp[0], vp[1], vp[2], vp[3])
	if depth {
		gl.Enable(gl.DEPTH_TEST)
	}
	wb.fftGPUTime = t
	return wb.fftGL
}

func (w *World) initGPUFFT() error {
	vs, err := compileGLShader(gl.VERTEX_SHADER, fftVertex)
	if err != nil {
		return err
	}
	fs, err := compileGLShader(gl.FRAGMENT_SHADER, fftFragment)
	if err != nil {
		gl.DeleteShader(vs)
		return err
	}
	prog := gl.CreateProgram()
	gl.AttachShader(prog, vs)
	gl.AttachShader(prog, fs)
	gl.LinkProgram(prog)
	gl.DeleteShader(vs)
	gl.DeleteShader(fs)
	var ok int32
	gl.GetProgramiv(prog, gl.LINK_STATUS, &ok)
	if ok == gl.FALSE {
		log := glInfoLog(prog, true)
		gl.DeleteProgram(prog)
		return fmt.Errorf("fft link: %s", log)
	}
	var vao, vbo uint32
	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)
	quad := []float32{-1, -1, 1, -1, -1, 1, 1, 1}
	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(quad)*4, gl.Ptr(quad), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 8, nil)
	gl.BindVertexArray(0)
	w.fftProg = prog
	w.fftVAO = vao
	w.fftQuad = vbo
	return nil
}
