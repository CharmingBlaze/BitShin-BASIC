package runtime

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// FSR 1: when the scene draws smaller than the window, the post pass finishes
// it into a small 8-bit target, EASU upsamples that along edges, and RCAS
// sharpens it onto the window. HUD text draws after, at full size.
//
// The GLSL is DBPRO2026's port of AMD FidelityFX FSR 1 ffx_fsr1.h.
//
// Copyright (c) 2021 Advanced Micro Devices, Inc. All rights reserved.
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

const mbEASUFragment = `#version 330 core
out vec4 finalColor;
uniform sampler2D uInput;   // tone-mapped, sRGB-encoded image at the render size
uniform vec2 uInSize;       // its size in pixels
uniform vec2 uOutSize;      // the size being drawn

vec3 tap(ivec2 p) {
    return texelFetch(uInput, clamp(p, ivec2(0), ivec2(uInSize) - 1), 0).rgb;
}

float luma(vec3 c) {
    return c.b * 0.5 + (c.r * 0.5 + c.g);
}

// Direction and length of the edge around one of the four centre texels, from
// the cross a (above), b (left), c (centre), d (right), e (below), weighted by w.
void accumulate(inout vec2 dir, inout float len, float w, float lA, float lB, float lC, float lD, float lE) {
    float dc = lD - lC;
    float cb = lC - lB;
    float lenX = 1.0 / max(max(abs(dc), abs(cb)), 1e-30);
    float dirX = lD - lB;
    dir.x += dirX * w;
    lenX = clamp(abs(dirX) * lenX, 0.0, 1.0);
    len += lenX * lenX * w;

    float ec = lE - lC;
    float ca = lC - lA;
    float lenY = 1.0 / max(max(abs(ec), abs(ca)), 1e-30);
    float dirY = lE - lA;
    dir.y += dirY * w;
    lenY = clamp(abs(dirY) * lenY, 0.0, 1.0);
    len += lenY * lenY * w;
}

// One tap of the approximate Lanczos(2) kernel, stretched along the edge.
void kernel(inout vec3 aC, inout float aW, vec2 off, vec2 dir, vec2 len2, float lob, float clp, vec3 c) {
    vec2 v = vec2(off.x * dir.x + off.y * dir.y, off.x * (-dir.y) + off.y * dir.x) * len2;
    float d2 = min(dot(v, v), clp);
    float wB = 2.0 / 5.0 * d2 - 1.0;
    float wA = lob * d2 - 1.0;
    wB *= wB;
    wA *= wA;
    wB = 25.0 / 16.0 * wB - (25.0 / 16.0 - 1.0);
    float w = wB * wA;
    aC += c * w;
    aW += w;
}

void main() {
    vec2 pp = gl_FragCoord.xy * (uInSize / uOutSize) - 0.5;
    vec2 fp = floor(pp);
    pp -= fp;
    ivec2 f0 = ivec2(fp);

    // The 12-tap footprint:   b c
    //                       e f g h
    //                       i j k l
    //                         n o
    vec3 b = tap(f0 + ivec2(0, -1)), c = tap(f0 + ivec2(1, -1));
    vec3 e = tap(f0 + ivec2(-1, 0)), f = tap(f0), g = tap(f0 + ivec2(1, 0)), h = tap(f0 + ivec2(2, 0));
    vec3 i = tap(f0 + ivec2(-1, 1)), j = tap(f0 + ivec2(0, 1)), k = tap(f0 + ivec2(1, 1)), l = tap(f0 + ivec2(2, 1));
    vec3 n = tap(f0 + ivec2(0, 2)), o = tap(f0 + ivec2(1, 2));

    float bL = luma(b), cL = luma(c), eL = luma(e), fL = luma(f), gL = luma(g), hL = luma(h);
    float iL = luma(i), jL = luma(j), kL = luma(k), lL = luma(l), nL = luma(n), oL = luma(o);

    vec2 dir = vec2(0.0);
    float len = 0.0;
    accumulate(dir, len, (1.0 - pp.x) * (1.0 - pp.y), bL, eL, fL, gL, jL);
    accumulate(dir, len, pp.x * (1.0 - pp.y), cL, fL, gL, hL, kL);
    accumulate(dir, len, (1.0 - pp.x) * pp.y, fL, iL, jL, kL, nL);
    accumulate(dir, len, pp.x * pp.y, gL, jL, kL, lL, oL);

    // Normalise the direction (no edge: point along x).
    float dirR = dot(dir, dir);
    bool zro = dirR < 1.0 / 32768.0;
    dirR = zro ? 1.0 : inversesqrt(dirR);
    dir.x = zro ? 1.0 : dir.x;
    dir *= dirR;

    // Stronger edges stretch the kernel along the edge and shrink it across.
    len = len * 0.5;
    len *= len;
    float stretch = dot(dir, dir) / max(abs(dir.x), abs(dir.y));
    vec2 len2 = vec2(1.0 + (stretch - 1.0) * len, 1.0 - 0.5 * len);
    float lob = 0.5 + ((1.0 / 4.0 - 0.04) - 0.5) * len;
    float clp = 1.0 / lob;

    vec3 aC = vec3(0.0);
    float aW = 0.0;
    kernel(aC, aW, vec2(0.0, -1.0) - pp, dir, len2, lob, clp, b);
    kernel(aC, aW, vec2(1.0, -1.0) - pp, dir, len2, lob, clp, c);
    kernel(aC, aW, vec2(-1.0, 1.0) - pp, dir, len2, lob, clp, i);
    kernel(aC, aW, vec2(0.0, 1.0) - pp, dir, len2, lob, clp, j);
    kernel(aC, aW, vec2(0.0, 0.0) - pp, dir, len2, lob, clp, f);
    kernel(aC, aW, vec2(-1.0, 0.0) - pp, dir, len2, lob, clp, e);
    kernel(aC, aW, vec2(1.0, 1.0) - pp, dir, len2, lob, clp, k);
    kernel(aC, aW, vec2(2.0, 1.0) - pp, dir, len2, lob, clp, l);
    kernel(aC, aW, vec2(2.0, 0.0) - pp, dir, len2, lob, clp, h);
    kernel(aC, aW, vec2(1.0, 0.0) - pp, dir, len2, lob, clp, g);
    kernel(aC, aW, vec2(1.0, 2.0) - pp, dir, len2, lob, clp, o);
    kernel(aC, aW, vec2(0.0, 2.0) - pp, dir, len2, lob, clp, n);

    // No ringing: stay inside the range of the four nearest texels.
    vec3 lo = min(min(f, g), min(j, k));
    vec3 hi = max(max(f, g), max(j, k));
    finalColor = vec4(min(hi, max(lo, aC / aW)), 1.0);
}
`

const mbRCASFragment = `#version 330 core
out vec4 finalColor;
uniform sampler2D uInput;
uniform float uSharpness;   // exp2(-stops): 1 = strongest

// The most negative lobe weight (keeps the 5-tap filter from going unstable).
const float LIMIT = 0.25 - 1.0 / 16.0;

vec3 tap(ivec2 p) {
    return texelFetch(uInput, clamp(p, ivec2(0), textureSize(uInput, 0) - 1), 0).rgb;
}

void main() {
    ivec2 sp = ivec2(gl_FragCoord.xy);
    //   b
    // d e f
    //   h
    vec3 b = tap(sp + ivec2(0, -1));
    vec3 d = tap(sp + ivec2(-1, 0));
    vec3 e = tap(sp);
    vec3 f = tap(sp + ivec2(1, 0));
    vec3 h = tap(sp + ivec2(0, 1));

    vec3 mn4 = min(min(b, d), min(f, h));
    vec3 mx4 = max(max(b, d), max(f, h));
    // How far the ring can push the centre before it clips at 0 or 1.
    vec3 hitMin = min(mn4, e) / max(4.0 * mx4, vec3(1e-5));
    vec3 hitMax = (1.0 - max(mx4, e)) / min(4.0 * mn4 - 4.0, vec3(-1e-5));
    vec3 lobeRGB = max(-hitMin, hitMax);
    float lobe = max(-LIMIT, min(max(lobeRGB.r, max(lobeRGB.g, lobeRGB.b)), 0.0)) * uSharpness;

    float rcpL = 1.0 / (4.0 * lobe + 1.0);
    finalColor = vec4((lobe * (b + d + f + h) + e) * rcpL, 1.0);
}
`

// fsrTarget is one 8-bit color target for the upscale chain.
type fsrTarget struct {
	fbo, tex uint32
	w, h     int
}

func (t *fsrTarget) ensure(ww, hh int) bool {
	if t.fbo != 0 && t.w == ww && t.h == hh {
		return true
	}
	t.dispose()
	gl.GenTextures(1, &t.tex)
	gl.BindTexture(gl.TEXTURE_2D, t.tex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(ww), int32(hh), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR) // FXAA reads between texels
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.GenFramebuffers(1, &t.fbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, t.fbo)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, t.tex, 0)
	ok := gl.CheckFramebufferStatus(gl.FRAMEBUFFER) == gl.FRAMEBUFFER_COMPLETE
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	if !ok {
		t.dispose()
		return false
	}
	t.w, t.h = ww, hh
	return true
}

func (t *fsrTarget) dispose() {
	if t.tex != 0 {
		gl.DeleteTextures(1, &t.tex)
		t.tex = 0
	}
	if t.fbo != 0 {
		gl.DeleteFramebuffers(1, &t.fbo)
		t.fbo = 0
	}
	t.w, t.h = 0, 0
}

type fsrState struct {
	easu, rcas uint32
	failed     bool
	small, big fsrTarget
	mid        fsrTarget // the FXAA result, on its way to the upscale
	fxaa       uint32
	fxaaFailed bool
}

func fsrProgram(frag string) (uint32, error) {
	return compileLink(
		struct {
			kind uint32
			src  string
		}{gl.VERTEX_SHADER, mbpostVertex},
		struct {
			kind uint32
			src  string
		}{gl.FRAGMENT_SHADER, frag},
	)
}

func (w *World) ensureFSR() bool {
	f := &w.fsr
	if f.failed {
		return false
	}
	if f.easu != 0 && f.rcas != 0 {
		return true
	}
	var err error
	if f.easu, err = fsrProgram(mbEASUFragment); err == nil {
		f.rcas, err = fsrProgram(mbRCASFragment)
	}
	if err != nil {
		fmt.Println("FSR: plain upscale:", err)
		f.failed = true
		return false
	}
	return true
}

// beginFSR points the post pass at the small target when the scene is smaller
// than the window. False leaves it drawing straight to the window, stretched.
func (w *World) beginFSR(pw, ph, dw, dh int) bool {
	if w.retroScr.lowW > 0 || pw >= dw || ph >= dh || !w.ensureFSR() {
		return false
	}
	if !w.fsr.small.ensure(pw, ph) || !w.fsr.big.ensure(dw, dh) {
		return false
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, w.fsr.small.fbo)
	gl.Viewport(0, 0, int32(pw), int32(ph))
	return true
}

// endFSR takes the small picture up to the window. src holds it.
func (w *World) endFSR(src uint32, pw, ph, dw, dh int) {
	f := &w.fsr
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindVertexArray(w.post.vao)

	gl.BindFramebuffer(gl.FRAMEBUFFER, f.big.fbo)
	gl.Viewport(0, 0, int32(dw), int32(dh))
	gl.UseProgram(f.easu)
	gl.BindTexture(gl.TEXTURE_2D, src)
	gl.Uniform1i(gl.GetUniformLocation(f.easu, gl.Str("uInput\x00")), 0)
	gl.Uniform2f(gl.GetUniformLocation(f.easu, gl.Str("uInSize\x00")), float32(pw), float32(ph))
	gl.Uniform2f(gl.GetUniformLocation(f.easu, gl.Str("uOutSize\x00")), float32(dw), float32(dh))
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.UseProgram(f.rcas)
	gl.BindTexture(gl.TEXTURE_2D, f.big.tex)
	gl.Uniform1i(gl.GetUniformLocation(f.rcas, gl.Str("uInput\x00")), 0)
	gl.Uniform1f(gl.GetUniformLocation(f.rcas, gl.Str("uSharpness\x00")), 0.5)
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	gl.BindVertexArray(0)
	gl.BindTexture(gl.TEXTURE_2D, 0)
}
