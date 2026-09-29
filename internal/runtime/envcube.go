package runtime

import (
	"math"

	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/renderer"
	"github.com/go-gl/gl/v3.3-core/gl"
)

const (
	skyCubeSize = 32
	skyCubeMips = 6
	probeFace   = 32
	envCubeUnit = 12
)

func (w *World) ensureSkyCube() {
	if w.skyCube != 0 {
		return
	}
	var cube uint32
	gl.GenTextures(1, &cube)
	gl.BindTexture(gl.TEXTURE_CUBE_MAP, cube)
	for mip := 0; mip < skyCubeMips; mip++ {
		sz := skyCubeSize >> mip
		if sz < 1 {
			sz = 1
		}
		radius := float32(mip) / float32(skyCubeMips-1) * 0.85
		for face := 0; face < 6; face++ {
			pix := make([]byte, sz*sz*4)
			for y := 0; y < sz; y++ {
				for x := 0; x < sz; x++ {
					dx, dy, dz := envFaceDir(face, x, y, sz)
					r, g, b := prefilterSky(dx, dy, dz, radius)
					i := (y*sz + x) * 4
					pix[i] = toByte(r)
					pix[i+1] = toByte(g)
					pix[i+2] = toByte(b)
					pix[i+3] = 255
				}
			}
			gl.TexImage2D(gl.TEXTURE_CUBE_MAP_POSITIVE_X+uint32(face), int32(mip), gl.RGBA8, int32(sz), int32(sz), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pix))
		}
	}
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_R, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_MAX_LEVEL, skyCubeMips-1)
	gl.BindTexture(gl.TEXTURE_CUBE_MAP, 0)
	w.skyCube = cube
}

func (w *World) bindEnvCube(gs *gls.GLS) {
	if gs == nil {
		return
	}
	w.ensureSkyCube()
	cube := w.skyCube
	if pr := w.nearestReadyProbe(); pr != nil {
		cube = pr.cube
	}
	if cube == 0 {
		setUni1i(gs, "HasEnvCube", 0)
		return
	}
	var prev int32
	gl.GetIntegerv(gl.ACTIVE_TEXTURE, &prev)
	gl.ActiveTexture(gl.TEXTURE0 + envCubeUnit)
	gl.BindTexture(gl.TEXTURE_CUBE_MAP, cube)
	setUni1i(gs, "uEnvCube", envCubeUnit)
	setUni1i(gs, "HasEnvCube", 1)
	if w.cam != nil {
		cp := worldPos(w.cam.GetNode())
		setUni3f(gs, "CamWorldPos", cp.X, cp.Y, cp.Z)
	}
	gl.ActiveTexture(uint32(prev))
}

func (w *World) nearestReadyProbe() *lightProbe {
	pr := w.nearestProbe()
	if pr == nil || !pr.ready || pr.cube == 0 {
		return nil
	}
	return pr
}

func (w *World) nearestProbe() *lightProbe {
	if len(w.probes) == 0 {
		return nil
	}
	x, y, z := float32(0), float32(2), float32(0)
	if w.cam != nil {
		p := worldPos(w.cam.GetNode())
		x, y, z = fromG3N(p.X, p.Y, p.Z)
	}
	var best *lightProbe
	bestD := float32(1e12)
	for _, pr := range w.probes {
		if pr == nil {
			continue
		}
		dx, dy, dz := pr.x-x, pr.y-y, pr.z-z
		d := dx*dx + dy*dy + dz*dz
		if d < bestD {
			bestD = d
			best = pr
		}
	}
	return best
}

func (w *World) ensureProbeTarget(pr *lightProbe) bool {
	if pr == nil {
		return false
	}
	if pr.cube != 0 && pr.fbo != 0 {
		return true
	}
	var cube, fbo, depth uint32
	gl.GenTextures(1, &cube)
	gl.BindTexture(gl.TEXTURE_CUBE_MAP, cube)
	for face := 0; face < 6; face++ {
		gl.TexImage2D(gl.TEXTURE_CUBE_MAP_POSITIVE_X+uint32(face), 0, gl.RGBA8, probeFace, probeFace, 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	}
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_R, gl.CLAMP_TO_EDGE)
	gl.GenerateMipmap(gl.TEXTURE_CUBE_MAP)
	gl.GenRenderbuffers(1, &depth)
	gl.BindRenderbuffer(gl.RENDERBUFFER, depth)
	gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH_COMPONENT24, probeFace, probeFace)
	gl.GenFramebuffers(1, &fbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_CUBE_MAP_POSITIVE_X, cube, 0)
	gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, depth)
	st := gl.CheckFramebufferStatus(gl.FRAMEBUFFER)
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.BindTexture(gl.TEXTURE_CUBE_MAP, 0)
	gl.BindRenderbuffer(gl.RENDERBUFFER, 0)
	if st != gl.FRAMEBUFFER_COMPLETE {
		gl.DeleteTextures(1, &cube)
		gl.DeleteFramebuffers(1, &fbo)
		gl.DeleteRenderbuffers(1, &depth)
		return false
	}
	pr.cube, pr.fbo, pr.depth = cube, fbo, depth
	return true
}

func (w *World) stepProbeCapture(rend *renderer.Renderer) {
	if w.capturingProbe || rend == nil || w.scene == nil || len(w.probes) == 0 {
		return
	}
	pr := w.nearestProbe()
	if pr == nil || !w.ensureProbeTarget(pr) {
		return
	}
	if w.probeCam == nil {
		w.probeCam = camera.New(1)
		w.probeCam.SetFov(90)
		w.probeCam.SetNear(0.08)
		w.probeCam.SetFar(400)
	}
	face := pr.face % 6
	gx, gy, gz := toG3N(pr.x, pr.y, pr.z)
	dir := cubeFaceDir[face]
	upv := cubeFaceUp[face]
	w.probeCam.SetPosition(gx, gy, gz)
	tgt := math32.Vector3{gx + dir.X, gy + dir.Y, gz + dir.Z}
	w.probeCam.LookAt(&tgt, &upv)

	var prevFBO int32
	gl.GetIntegerv(gl.FRAMEBUFFER_BINDING, &prevFBO)
	var prevVP [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &prevVP[0])
	gl.BindFramebuffer(gl.FRAMEBUFFER, pr.fbo)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_CUBE_MAP_POSITIVE_X+uint32(face), pr.cube, 0)
	gl.Viewport(0, 0, probeFace, probeFace)
	gl.ClearColor(pr.sky.R, pr.sky.G, pr.sky.B, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	w.capturingProbe = true
	_ = rend.Render(w.scene, w.probeCam)
	w.capturingProbe = false
	pr.face = face + 1
	if pr.face%6 == 0 {
		gl.BindTexture(gl.TEXTURE_CUBE_MAP, pr.cube)
		gl.GenerateMipmap(gl.TEXTURE_CUBE_MAP)
		gl.BindTexture(gl.TEXTURE_CUBE_MAP, 0)
		pr.ready = true
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, uint32(prevFBO))
	gl.Viewport(prevVP[0], prevVP[1], prevVP[2], prevVP[3])
}

func envFaceDir(face, x, y, size int) (float32, float32, float32) {
	u := 2*(float32(x)+0.5)/float32(size) - 1
	v := 2*(float32(y)+0.5)/float32(size) - 1
	var dx, dy, dz float32
	switch face {
	case 0:
		dx, dy, dz = 1, -v, -u
	case 1:
		dx, dy, dz = -1, -v, u
	case 2:
		dx, dy, dz = u, 1, v
	case 3:
		dx, dy, dz = u, -1, -v
	case 4:
		dx, dy, dz = u, -v, 1
	default:
		dx, dy, dz = -u, -v, -1
	}
	return norm3(dx, dy, dz)
}

func prefilterSky(dx, dy, dz, radius float32) (float32, float32, float32) {
	if radius < 0.02 {
		return skyColor(dx, dy, dz)
	}
	tx, ty, tz, bx, by, bz := tangentBasis(dx, dy, dz)
	var r, g, b float32
	const n = 8
	for i := 0; i < n; i++ {
		ang := float64(i) * 2.399963
		rad := float64(radius) * float64(i+1) / n
		ox := float32(math.Cos(ang) * rad)
		oy := float32(math.Sin(ang) * rad)
		sx, sy, sz := norm3(dx+tx*ox+bx*oy, dy+ty*ox+by*oy, dz+tz*ox+bz*oy)
		cr, cg, cb := skyColor(sx, sy, sz)
		r += cr
		g += cg
		b += cb
	}
	return r / n, g / n, b / n
}

func skyColor(dx, dy, dz float32) (float32, float32, float32) {
	dx, dy, dz = norm3(dx, dy, dz)
	h := dy*0.5 + 0.5
	if h < 0 {
		h = 0
	}
	if h > 1 {
		h = 1
	}
	t := h * h * (3 - 2*h)
	r := 0.22 + (0.52-0.22)*t
	g := 0.18 + (0.64-0.18)*t
	b := 0.14 + (0.86-0.14)*t
	sx, sy, sz := norm3(0.28, 0.86, 0.32)
	d := dx*sx + dy*sy + dz*sz
	if d > 0 {
		spec := float32(math.Pow(float64(d), 80))
		r += spec * 1.6
		g += spec * 1.45
		b += spec * 1.1
	}
	hz := 1 - float32(math.Abs(float64(dy)))
	r += hz * hz * 0.12
	g += hz * hz * 0.12
	b += hz * hz * 0.14
	return r, g, b
}

func tangentBasis(dx, dy, dz float32) (tx, ty, tz, bx, by, bz float32) {
	ax, ay, az := float32(0), float32(1), float32(0)
	if dy > 0.85 || dy < -0.85 {
		ax, ay, az = 1, 0, 0
	}
	tx, ty, tz = norm3(dy*az-dz*ay, dz*ax-dx*az, dx*ay-dy*ax)
	bx, by, bz = dy*tz-dz*ty, dz*tx-dx*tz, dx*ty-dy*tx
	return
}

func norm3(x, y, z float32) (float32, float32, float32) {
	l := float32(math.Sqrt(float64(x*x + y*y + z*z)))
	if l < 1e-6 {
		return 0, 1, 0
	}
	return x / l, y / l, z / l
}

func toByte(v float32) byte {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	return byte(v * 255)
}
