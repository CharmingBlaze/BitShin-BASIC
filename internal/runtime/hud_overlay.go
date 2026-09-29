package runtime

import (
	"image"
	"image/draw"

	"github.com/go-gl/gl/v3.3-core/gl"
)

type hudOverlayState struct {
	prog        uint32
	vao         uint32
	vbo         uint32
	uScreenSize int32
	uTexture    int32
	uUseTexture int32
	uShape      int32
	ready       bool
}

const mbhudVertex = `#version 330 core
layout(location = 0) in vec2 aPos;
layout(location = 1) in vec2 aUV;
layout(location = 2) in vec4 aColor;

out vec2 vUV;
out vec4 vColor;

uniform vec2 uScreenSize;

void main() {
	vUV = aUV;
	vColor = aColor;
	vec2 ndc = vec2((aPos.x / uScreenSize.x) * 2.0 - 1.0, 1.0 - (aPos.y / uScreenSize.y) * 2.0);
	gl_Position = vec4(ndc, 0.0, 1.0);
}
`

const mbhudFragment = `#version 330 core
in vec2 vUV;
in vec4 vColor;
out vec4 FragColor;

uniform sampler2D uTexture;
uniform int uUseTexture;
uniform int uShape; // 0 = rect/quad/line, 1 = oval/circle

void main() {
	if (uShape == 1) {
		vec2 d = vUV - vec2(0.5);
		float distSq = dot(d, d);
		if (distSq > 0.25) {
			discard;
		}
	}
	vec4 col = vColor;
	if (uUseTexture != 0) {
		col *= texture(uTexture, vUV);
	}
	FragColor = col;
}
`

func (w *World) ensureHUDOverlay() error {
	if w.hudOver.ready {
		return nil
	}
	p, err := compileLink(
		struct {
			kind uint32
			src  string
		}{gl.VERTEX_SHADER, mbhudVertex},
		struct {
			kind uint32
			src  string
		}{gl.FRAGMENT_SHADER, mbhudFragment},
	)
	if err != nil {
		return err
	}
	w.hudOver.prog = p
	w.hudOver.uScreenSize = gl.GetUniformLocation(p, gl.Str("uScreenSize\x00"))
	w.hudOver.uTexture = gl.GetUniformLocation(p, gl.Str("uTexture\x00"))
	w.hudOver.uUseTexture = gl.GetUniformLocation(p, gl.Str("uUseTexture\x00"))
	w.hudOver.uShape = gl.GetUniformLocation(p, gl.Str("uShape\x00"))

	gl.GenVertexArrays(1, &w.hudOver.vao)
	gl.GenBuffers(1, &w.hudOver.vbo)

	gl.BindVertexArray(w.hudOver.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, w.hudOver.vbo)

	stride := int32(8 * 4) // 2 floats pos, 2 floats uv, 4 floats color
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, stride, gl.PtrOffset(0))

	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(2*4))

	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointer(2, 4, gl.FLOAT, false, stride, gl.PtrOffset(4*4))

	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)

	w.hudOver.ready = true
	return nil
}

func uploadGLTexture(img image.Image) (uint32, int, int) {
	if img == nil {
		return 0, 0, 0
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)

	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(w), int32(h), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgba.Pix))
	gl.BindTexture(gl.TEXTURE_2D, 0)
	return tex, w, h
}

func (w *World) drawHUDOverlay3D(ww, hh int) {
	if len(w.draws) == 0 || ww <= 0 || hh <= 0 {
		return
	}
	if err := w.ensureHUDOverlay(); err != nil {
		return
	}

	gl.Enable(gl.BLEND)
	gl.BlendEquation(gl.FUNC_ADD)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.CULL_FACE)
	gl.Disable(gl.SCISSOR_TEST)
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.Viewport(0, 0, int32(ww), int32(hh))

	gl.UseProgram(w.hudOver.prog)
	gl.Uniform2f(w.hudOver.uScreenSize, float32(ww), float32(hh))
	gl.BindVertexArray(w.hudOver.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, w.hudOver.vbo)

	for _, op := range w.draws {
		rf := float32(op.r) / 255.0
		gf := float32(op.g) / 255.0
		bf := float32(op.b) / 255.0
		af := float32(op.a) / 255.0
		if op.a == 0 && op.kind != 0 {
			af = 1.0
		}

		switch op.kind {
		case 0: // Image
			var glTex uint32 = op.glTex
			tw, th := op.texW, op.texH
			if glTex == 0 && op.imgID > 0 {
				if im := w.images[op.imgID]; im != nil {
					if im.glTex == 0 && im.src != nil {
						im.glTex, im.w, im.h = uploadGLTexture(im.src)
					}
					glTex = im.glTex
					tw, th = im.w, im.h
				}
			}
			if glTex == 0 {
				continue
			}
			u0, v0, u1, v1 := float32(0), float32(0), float32(1), float32(1)
			if op.sw > 0 && op.sh > 0 && tw > 0 && th > 0 {
				u0 = float32(op.sx) / float32(tw)
				v0 = float32(op.sy) / float32(th)
				u1 = float32(op.sx+op.sw) / float32(tw)
				v1 = float32(op.sy+op.sh) / float32(th)
			}
			dw, dh := op.w, op.h
			if dw <= 0 && tw > 0 {
				dw = float32(tw)
			}
			if dh <= 0 && th > 0 {
				dh = float32(th)
			}
			verts := []float32{
				op.x, op.y, u0, v0, rf, gf, bf, af,
				op.x + dw, op.y, u1, v0, rf, gf, bf, af,
				op.x, op.y + dh, u0, v1, rf, gf, bf, af,

				op.x, op.y + dh, u0, v1, rf, gf, bf, af,
				op.x + dw, op.y, u1, v0, rf, gf, bf, af,
				op.x + dw, op.y + dh, u1, v1, rf, gf, bf, af,
			}
			gl.ActiveTexture(gl.TEXTURE0)
			gl.BindTexture(gl.TEXTURE_2D, glTex)
			gl.Uniform1i(w.hudOver.uTexture, 0)
			gl.Uniform1i(w.hudOver.uUseTexture, 1)
			gl.Uniform1i(w.hudOver.uShape, 0)

			gl.BufferData(gl.ARRAY_BUFFER, len(verts)*4, gl.Ptr(verts), gl.STREAM_DRAW)
			gl.DrawArrays(gl.TRIANGLES, 0, 6)

		case 1: // Rect
			gl.Uniform1i(w.hudOver.uUseTexture, 0)
			gl.Uniform1i(w.hudOver.uShape, 0)
			if op.filled {
				verts := []float32{
					op.x, op.y, 0, 0, rf, gf, bf, af,
					op.x + op.w, op.y, 1, 0, rf, gf, bf, af,
					op.x, op.y + op.h, 0, 1, rf, gf, bf, af,

					op.x, op.y + op.h, 0, 1, rf, gf, bf, af,
					op.x + op.w, op.y, 1, 0, rf, gf, bf, af,
					op.x + op.w, op.y + op.h, 1, 1, rf, gf, bf, af,
				}
				gl.BufferData(gl.ARRAY_BUFFER, len(verts)*4, gl.Ptr(verts), gl.STREAM_DRAW)
				gl.DrawArrays(gl.TRIANGLES, 0, 6)
			} else {
				verts := []float32{
					op.x, op.y, 0, 0, rf, gf, bf, af,
					op.x + op.w, op.y, 1, 0, rf, gf, bf, af,

					op.x + op.w, op.y, 1, 0, rf, gf, bf, af,
					op.x + op.w, op.y + op.h, 1, 1, rf, gf, bf, af,

					op.x + op.w, op.y + op.h, 1, 1, rf, gf, bf, af,
					op.x, op.y + op.h, 0, 1, rf, gf, bf, af,

					op.x, op.y + op.h, 0, 1, rf, gf, bf, af,
					op.x, op.y, 0, 0, rf, gf, bf, af,
				}
				gl.BufferData(gl.ARRAY_BUFFER, len(verts)*4, gl.Ptr(verts), gl.STREAM_DRAW)
				gl.DrawArrays(gl.LINES, 0, 8)
			}

		case 2: // Oval
			gl.Uniform1i(w.hudOver.uUseTexture, 0)
			gl.Uniform1i(w.hudOver.uShape, 1)
			verts := []float32{
				op.x, op.y, 0, 0, rf, gf, bf, af,
				op.x + op.w, op.y, 1, 0, rf, gf, bf, af,
				op.x, op.y + op.h, 0, 1, rf, gf, bf, af,

				op.x, op.y + op.h, 0, 1, rf, gf, bf, af,
				op.x + op.w, op.y, 1, 0, rf, gf, bf, af,
				op.x + op.w, op.y + op.h, 1, 1, rf, gf, bf, af,
			}
			gl.BufferData(gl.ARRAY_BUFFER, len(verts)*4, gl.Ptr(verts), gl.STREAM_DRAW)
			gl.DrawArrays(gl.TRIANGLES, 0, 6)

		case 3: // Line
			gl.Uniform1i(w.hudOver.uUseTexture, 0)
			gl.Uniform1i(w.hudOver.uShape, 0)
			verts := []float32{
				op.x, op.y, 0, 0, rf, gf, bf, af,
				op.x2, op.y2, 1, 1, rf, gf, bf, af,
			}
			gl.BufferData(gl.ARRAY_BUFFER, len(verts)*4, gl.Ptr(verts), gl.STREAM_DRAW)
			gl.DrawArrays(gl.LINES, 0, 2)
		}
	}

	gl.BindTexture(gl.TEXTURE_2D, 0)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)
	gl.UseProgram(0)

	w.guiRestoreSceneGL()
}
