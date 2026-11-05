package runtime

import (
	"fmt"
	"image"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// The sky cube, prefiltered on the GPU. On the CPU a rebuild stalled the frame by tens of
// milliseconds every time the sky changed, which is every frame of a day passing, and it
// could only afford an 8-sample blur. This samples the real reflection lobe (GGX), and
// stores linear light in half float, so shaders need no decode and dark mips do not band.
//
// The sun is not added here: its highlight comes from the direct light, and adding it to
// the sky as well would count it twice.

const mbEnvVertex = `#version 330 core
out vec2 vUV;
void main() {
	vec2 p = vec2(gl_VertexID == 1 ? 3.0 : -1.0, gl_VertexID == 2 ? 3.0 : -1.0);
	vUV = p * 0.5 + 0.5;
	gl_Position = vec4(p, 0.0, 1.0);
}
`

// After Karis, "Real Shading in Unreal Engine 4". Each sample reads the mip of the source
// whose texels match the part of the lobe it stands for, so few samples stay smooth.
const mbEnvPrefilter = `#version 330 core
in vec2 vUV;
out vec4 frag;
uniform samplerCube uSrc;
uniform int uFace;
uniform float uRough;
uniform float uSrcSize;
uniform int uDecode;

const int SAMPLES = 64;

// The same layout as the images uploaded to uSrc, and as OpenGL's own cube faces.
vec3 faceDir(vec2 uv) {
	vec2 q = uv * 2.0 - 1.0;
	if (uFace == 0) { return normalize(vec3(1.0, -q.y, -q.x)); }
	if (uFace == 1) { return normalize(vec3(-1.0, -q.y, q.x)); }
	if (uFace == 2) { return normalize(vec3(q.x, 1.0, q.y)); }
	if (uFace == 3) { return normalize(vec3(q.x, -1.0, -q.y)); }
	if (uFace == 4) { return normalize(vec3(q.x, -q.y, 1.0)); }
	return normalize(vec3(-q.x, -q.y, -1.0));
}

float radicalInverse(uint b) {
	b = (b << 16u) | (b >> 16u);
	b = ((b & 0x55555555u) << 1u) | ((b & 0xAAAAAAAAu) >> 1u);
	b = ((b & 0x33333333u) << 2u) | ((b & 0xCCCCCCCCu) >> 2u);
	b = ((b & 0x0F0F0F0Fu) << 4u) | ((b & 0xF0F0F0F0u) >> 4u);
	b = ((b & 0x00FF00FFu) << 8u) | ((b & 0xFF00FF00u) >> 8u);
	return float(b) * 2.3283064365386963e-10;
}

vec3 readSrc(vec3 d, float lod) {
	vec3 c = textureLod(uSrc, d, lod).rgb;
	return uDecode != 0 ? pow(c, vec3(2.2)) : c;
}

void main() {
	vec3 N = faceDir(vUV);
	if (uRough <= 0.0) {
		frag = vec4(readSrc(N, 0.0), 1.0);
		return;
	}
	vec3 up = abs(N.y) < 0.99 ? vec3(0.0, 1.0, 0.0) : vec3(1.0, 0.0, 0.0);
	vec3 T = normalize(cross(up, N));
	vec3 B = cross(N, T);
	float a = uRough * uRough;
	float a2 = a * a;
	float texelArea = 12.566371 / (6.0 * uSrcSize * uSrcSize);
	vec3 sum = vec3(0.0);
	float weight = 0.0;
	for (int i = 0; i < SAMPLES; i++) {
		float u = float(i) / float(SAMPLES);
		float v = radicalInverse(uint(i));
		float cosH = sqrt((1.0 - v) / (1.0 + (a2 - 1.0) * v));
		float sinH = sqrt(max(1.0 - cosH * cosH, 0.0));
		float phi = 6.2831853 * u;
		vec3 H = normalize(T * (cos(phi) * sinH) + B * (sin(phi) * sinH) + N * cosH);
		vec3 L = reflect(-N, H);
		float nl = dot(N, L);
		if (nl <= 0.0) { continue; }
		float d = (cosH * cosH) * (a2 - 1.0) + 1.0;
		float pdf = max(a2 / (3.1415927 * d * d) * 0.25, 1e-5);
		float lod = 0.5 * log2((1.0 / (float(SAMPLES) * pdf)) / texelArea);
		sum += readSrc(L, max(lod, 0.0)) * nl;
		weight += nl;
	}
	frag = vec4(sum / max(weight, 1e-4), 1.0);
}
`

type envGPU struct {
	prog, vao, fbo, src uint32
	broken              bool
	seamless            bool
	srcFor              *image.RGBA
}

// skyFaceImages is the sky to reflect: its images, or a plain gradient made once.
func (w *World) skyFaceImages() [6]*image.RGBA {
	if w.skyImgs[0] != nil && w.skyImgs[5] != nil {
		return w.skyImgs
	}
	if w.envFallback[0] == nil {
		const size = 32
		for face := range w.envFallback {
			img := image.NewRGBA(image.Rect(0, 0, size, size))
			for y := 0; y < size; y++ {
				for x := 0; x < size; x++ {
					r, g, b := skyColor(envFaceDir(face, x, y, size))
					o := img.PixOffset(x, y)
					img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = toByte(r), toByte(g), toByte(b), 255
				}
			}
			w.envFallback[face] = img
		}
	}
	return w.envFallback
}

// prefilterSkyGPU fills the sky cube. It reports false if this machine cannot, and the
// caller uses the CPU path, as with an unusable half-float target.
func (w *World) prefilterSkyGPU(cube uint32, size int, decode bool) bool {
	g := &w.envGPU
	if g.broken {
		return false
	}
	if g.prog == 0 {
		p, err := compileLink(
			struct {
				kind uint32
				src  string
			}{gl.VERTEX_SHADER, mbEnvVertex},
			struct {
				kind uint32
				src  string
			}{gl.FRAGMENT_SHADER, mbEnvPrefilter},
		)
		if err != nil {
			fmt.Println("sky cube:", err)
			g.broken = true
			return false
		}
		g.prog = p
		gl.GenVertexArrays(1, &g.vao)
		gl.GenFramebuffers(1, &g.fbo)
		gl.GenTextures(1, &g.src)
	}

	// The source: the sky images as they are, with their own mips.
	imgs := w.skyFaceImages()
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_CUBE_MAP, g.src)
	if g.srcFor != imgs[0] {
		for face, img := range imgs {
			b := img.Bounds()
			gl.TexImage2D(gl.TEXTURE_CUBE_MAP_POSITIVE_X+uint32(face), 0, gl.RGBA8, int32(b.Dx()), int32(b.Dy()), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(img.Pix))
		}
		gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
		gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_R, gl.CLAMP_TO_EDGE)
		gl.GenerateMipmap(gl.TEXTURE_CUBE_MAP)
		g.srcFor = imgs[0]
	}

	var prevFBO int32
	var prevVP [4]int32
	var prevProg, prevVAO int32
	gl.GetIntegerv(gl.FRAMEBUFFER_BINDING, &prevFBO)
	gl.GetIntegerv(gl.VIEWPORT, &prevVP[0])
	gl.GetIntegerv(gl.CURRENT_PROGRAM, &prevProg)
	gl.GetIntegerv(gl.VERTEX_ARRAY_BINDING, &prevVAO)
	depth, blend, cull, scissor := gl.IsEnabled(gl.DEPTH_TEST), gl.IsEnabled(gl.BLEND), gl.IsEnabled(gl.CULL_FACE), gl.IsEnabled(gl.SCISSOR_TEST)
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.BLEND)
	gl.Disable(gl.CULL_FACE)
	gl.Disable(gl.SCISSOR_TEST)

	gl.BindFramebuffer(gl.FRAMEBUFFER, g.fbo)
	gl.DrawBuffer(gl.COLOR_ATTACHMENT0)
	gl.UseProgram(g.prog)
	gl.BindVertexArray(g.vao)
	loc := func(name string) int32 { return gl.GetUniformLocation(g.prog, gl.Str(name+"\x00")) }
	gl.Uniform1i(loc("uSrc"), 0)
	gl.Uniform1f(loc("uSrcSize"), float32(imgs[0].Bounds().Dx()))
	gl.Uniform1i(loc("uDecode"), int32(b01(decode)))
	ok := true
	for mip := 0; mip < skyCubeMips && ok; mip++ {
		sz := int32(size >> mip)
		if sz < 1 {
			sz = 1
		}
		gl.Viewport(0, 0, sz, sz)
		gl.Uniform1f(loc("uRough"), float32(mip)/float32(skyCubeMips-1))
		for face := 0; face < 6; face++ {
			gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_CUBE_MAP_POSITIVE_X+uint32(face), cube, int32(mip))
			if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
				ok = false
				break
			}
			gl.Uniform1i(loc("uFace"), int32(face))
			gl.DrawArrays(gl.TRIANGLES, 0, 3)
		}
	}
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_CUBE_MAP_POSITIVE_X, 0, 0)

	gl.BindFramebuffer(gl.FRAMEBUFFER, uint32(prevFBO))
	gl.Viewport(prevVP[0], prevVP[1], prevVP[2], prevVP[3])
	gl.UseProgram(uint32(prevProg))
	gl.BindVertexArray(uint32(prevVAO))
	restoreCap := func(c uint32, on bool) {
		if on {
			gl.Enable(c)
		} else {
			gl.Disable(c)
		}
	}
	restoreCap(gl.DEPTH_TEST, depth)
	restoreCap(gl.BLEND, blend)
	restoreCap(gl.CULL_FACE, cull)
	restoreCap(gl.SCISSOR_TEST, scissor)
	gl.BindTexture(gl.TEXTURE_CUBE_MAP, 0)
	drainGL("sky cube")
	if !ok {
		fmt.Println("sky cube: half-float target incomplete; building it on the CPU")
		g.broken = true
	}
	return ok
}
