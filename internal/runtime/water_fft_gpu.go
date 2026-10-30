package runtime

import (
	"fmt"
	"strconv"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// waterFFTN is the wave tile's resolution, on the CPU (WaterHeight) and the
// GPU (the texture the water samples) alike, so a float sits where the
// drawn wave is.
const waterFFTN = 64

// waterFFTNorm keeps the wave heights of the 32-sample tile this replaced:
// the same low frequencies keep their size, the new high ones are ripples.
const waterFFTNorm = 1.0 / (32 * 32)

// The GPU tile is a separable inverse DFT of the Phillips spectrum: rows
// into a complex RG32F target, then columns into the height texture. Each
// pass is N^3 terms (262k at 64), less than the old 32x32 direct sum.
var fftSpectrumGLSL = `
const float N = ` + strconv.Itoa(waterFFTN) + `.0;
uniform float Time;
vec2 spectrum(float ix, float iz) {
    float kx = ix - N * 0.5;
    float kz = iz - N * 0.5;
    float k = length(vec2(kx, kz));
    if (k < 0.5) { return vec2(0.0); }
    float phillips = exp(-1.0 / (k * k)) / (k * k * k * k);
    phillips *= kx * kx + kz * kz;
    float amp = sqrt(phillips) * 0.35;
    float phase = sqrt(9.81 * k / 8.0) * Time;
    return amp * vec2(cos(phase), sin(phase));
}
vec2 cmul(vec2 a, vec2 b) { return vec2(a.x * b.x - a.y * b.y, a.x * b.y + a.y * b.x); }
`

// Rows: texel (x, iz) = sum over ix of H(ix, iz) e^{i 2pi ix x / N}
var fftRowsFragment = `#version 330 core
` + fftSpectrumGLSL + `
out vec4 FragColor;
void main() {
    float x = gl_FragCoord.x - 0.5;
    float iz = gl_FragCoord.y - 0.5;
    vec2 sum = vec2(0.0);
    for (int i = 0; i < ` + strconv.Itoa(waterFFTN) + `; i++) {
        float ix = float(i);
        float ang = 6.2831853 * ix * x / N;
        sum += cmul(spectrum(ix, iz), vec2(cos(ang), sin(ang)));
    }
    FragColor = vec4(sum, 0.0, 1.0);
}
`

// Columns: height(x, z) = Re sum over iz of Rows(x, iz) e^{i 2pi iz z / N}.
// Red holds h/4 + 0.5, the encoding the water shader reads.
var fftColsFragment = `#version 330 core
const float N = ` + strconv.Itoa(waterFFTN) + `.0;
uniform sampler2D Rows;
uniform float Norm;
out vec4 FragColor;
void main() {
    int x = int(gl_FragCoord.x);
    float z = gl_FragCoord.y - 0.5;
    float sum = 0.0;
    for (int i = 0; i < ` + strconv.Itoa(waterFFTN) + `; i++) {
        vec2 r = texelFetch(Rows, ivec2(x, i), 0).rg;
        float ang = 6.2831853 * float(i) * z / N;
        sum += r.x * cos(ang) - r.y * sin(ang);
    }
    FragColor = vec4(sum * Norm / 4.0 + 0.5, 0.0, 0.0, 1.0);
}
`

const fftVertex = `#version 330 core
layout(location = 0) in vec2 aPos;
void main() {
    gl_Position = vec4(aPos, 0.0, 1.0);
}
`

func fftTarget(internal uint32, format, typ uint32, filter int32) (tex, fbo uint32) {
	n := int32(waterFFTN)
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, int32(internal), n, n, 0, format, typ, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, filter)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, filter)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	gl.GenFramebuffers(1, &fbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, tex, 0)
	ok := gl.CheckFramebufferStatus(gl.FRAMEBUFFER) == gl.FRAMEBUFFER_COMPLETE
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	if !ok {
		gl.DeleteFramebuffers(1, &fbo)
		gl.DeleteTextures(1, &tex)
		return 0, 0
	}
	return tex, fbo
}

func (w *World) drawGPUFFT(wb *waterBody, t float32) uint32 {
	if w == nil || wb == nil {
		return 0
	}
	if w.fftProg == 0 {
		if w.fftFailed {
			return 0
		}
		if err := w.initGPUFFT(); err != nil {
			w.fftFailed = true // the CPU tile takes over
			return 0
		}
	}
	if wb.fftGL == 0 || wb.fftFBO == 0 || wb.fftRows == 0 {
		rows, rowsFBO := fftTarget(gl.RG32F, gl.RG, gl.FLOAT, gl.NEAREST)
		tex, fbo := fftTarget(gl.RGBA16F, gl.RGBA, gl.HALF_FLOAT, gl.LINEAR)
		if rows == 0 || tex == 0 {
			return 0
		}
		wb.fftRows, wb.fftRowsFBO = rows, rowsFBO
		wb.fftGL, wb.fftFBO = tex, fbo
	}
	// Everything these passes bind is put back: other passes and G3N's
	// state cache expect what they left.
	var vp [4]int32
	var prog, vao, drawFBO, readFBO, unit, tex0 int32
	gl.GetIntegerv(gl.VIEWPORT, &vp[0])
	gl.GetIntegerv(gl.CURRENT_PROGRAM, &prog)
	gl.GetIntegerv(gl.VERTEX_ARRAY_BINDING, &vao)
	gl.GetIntegerv(gl.DRAW_FRAMEBUFFER_BINDING, &drawFBO)
	gl.GetIntegerv(gl.READ_FRAMEBUFFER_BINDING, &readFBO)
	gl.GetIntegerv(gl.ACTIVE_TEXTURE, &unit)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.GetIntegerv(gl.TEXTURE_BINDING_2D, &tex0)
	depth := gl.IsEnabled(gl.DEPTH_TEST)
	blend := gl.IsEnabled(gl.BLEND)
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.BLEND)
	gl.Viewport(0, 0, waterFFTN, waterFFTN)
	gl.BindVertexArray(w.fftVAO)

	gl.BindFramebuffer(gl.FRAMEBUFFER, wb.fftRowsFBO)
	gl.UseProgram(w.fftProg)
	gl.Uniform1f(gl.GetUniformLocation(w.fftProg, gl.Str("Time\x00")), t)
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	gl.BindFramebuffer(gl.FRAMEBUFFER, wb.fftFBO)
	gl.UseProgram(w.fftColsProg)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, wb.fftRows)
	gl.Uniform1i(gl.GetUniformLocation(w.fftColsProg, gl.Str("Rows\x00")), 0)
	gl.Uniform1f(gl.GetUniformLocation(w.fftColsProg, gl.Str("Norm\x00")), waterFFTNorm*wb.fftStrength())
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	gl.BindTexture(gl.TEXTURE_2D, uint32(tex0))
	gl.ActiveTexture(uint32(unit))
	gl.BindVertexArray(uint32(vao))
	gl.UseProgram(uint32(prog))
	gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, uint32(drawFBO))
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, uint32(readFBO))
	gl.Viewport(vp[0], vp[1], vp[2], vp[3])
	if depth {
		gl.Enable(gl.DEPTH_TEST)
	}
	if blend {
		gl.Enable(gl.BLEND)
	}
	wb.fftGPUTime = t
	return wb.fftGL
}

func (w *World) initGPUFFT() error {
	link := func(frag string) (uint32, error) {
		return compileLink(struct {
			kind uint32
			src  string
		}{gl.VERTEX_SHADER, fftVertex}, struct {
			kind uint32
			src  string
		}{gl.FRAGMENT_SHADER, frag})
	}
	rows, err := link(fftRowsFragment)
	if err != nil {
		return fmt.Errorf("fft rows: %w", err)
	}
	cols, err := link(fftColsFragment)
	if err != nil {
		gl.DeleteProgram(rows)
		return fmt.Errorf("fft cols: %w", err)
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
	w.fftProg = rows
	w.fftColsProg = cols
	w.fftVAO = vao
	w.fftQuad = vbo
	return nil
}
