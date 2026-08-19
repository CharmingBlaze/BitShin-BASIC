package runtime

import (
	"math"
	"strings"

	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/math32"
)

const (
	cascadeLambda         = float32(0.70)
	defaultShadowDistance = float32(250)
	defaultCascadeZPad    = float32(60)
	shadowFadeTail        = float32(30)
	cascadeGuardFraction  = float32(0.10)
	minCascadeGuard       = float32(3)
	maxCascadeGuard       = float32(12)
)

// cascadeSplit is a practical mix of logarithmic and uniform splits.
func cascadeSplit(i, n int, near, far, lambda float32) float32 {
	if n < 1 {
		n = 1
	}
	if i < 1 {
		return near
	}
	if i >= n {
		return far
	}
	p := float32(i) / float32(n)
	logS := near * math32.Pow(far/near, p)
	uniS := near + (far-near)*p
	return logS*lambda + uniS*(1.0-lambda)
}

func (w *World) shadowDistance(cam *camera.Camera) float32 {
	dist := w.shadow.distance
	if dist < 40 {
		dist = defaultShadowDistance
	}
	if w.shadow.fadeFar > dist {
		dist = w.shadow.fadeFar
	}
	if cam != nil {
		far := cam.Far()
		if far > dist+1 {
			return dist
		}
		if far > w.shadowNear(cam)+1 {
			return far
		}
	}
	return dist
}

func (w *World) shadowNear(cam *camera.Camera) float32 {
	near := float32(0.1)
	if cam != nil {
		near = cam.Near()
	}
	if near < 0.05 {
		near = 0.05
	}
	return near
}

func (w *World) cascadeZPad() float32 {
	pad := w.shadow.zPad
	if pad < 30 {
		pad = defaultCascadeZPad
	}
	if pad > 100 {
		pad = 100
	}
	return pad
}

func (w *World) applyShadowPreset(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "low":
		w.shadow.cascades = 2
		w.shadow.size = 1024
		w.shadow.pcf = 3
		w.shadow.filter = shadowFilterPCF
		w.shadow.distance = defaultShadowDistance
	case "medium", "med":
		w.shadow.cascades = 3
		w.shadow.size = 1536
		w.shadow.pcf = 3
		w.shadow.filter = shadowFilterPCF
		w.shadow.distance = defaultShadowDistance
	case "high":
		w.shadow.cascades = 4
		w.shadow.size = 2048
		w.shadow.pcf = 5
		w.shadow.filter = shadowFilterPCF
		w.shadow.distance = defaultShadowDistance
	case "ultra":
		w.shadow.cascades = 4
		w.shadow.size = 4096
		w.shadow.pcf = 5
		w.shadow.filter = shadowFilterPCF
		w.shadow.distance = defaultShadowDistance
	default:
		return false
	}
	w.ensureShadowOn()
	return true
}

func (w *World) computeCascadeSplits(cam *camera.Camera, n int) {
	if n < 1 {
		n = 1
	}
	if n > maxShadowCascades {
		n = maxShadowCascades
	}
	near := w.shadowNear(cam)
	far := w.shadowDistance(cam)
	if far <= near+1 {
		far = near + 80
	}
	for i := 0; i < n; i++ {
		w.shadow.splits[i] = cascadeSplit(i+1, n, near, far, cascadeLambda)
	}
	for i := n; i < maxShadowCascades; i++ {
		w.shadow.splits[i] = far
	}
	fadeFar := w.shadow.fadeFar
	if fadeFar <= 0 {
		fadeFar = far
	}
	fadeNear := w.shadow.fadeNear
	if fadeNear <= 0 || fadeNear >= fadeFar {
		fadeNear = fadeFar - shadowFadeTail
	}
	if fadeNear < near {
		fadeNear = near
	}
	w.shadow.fadeNear = fadeNear
	w.shadow.fadeFar = fadeFar
}

func cameraViewBasis(cam *camera.Camera) (pos, fwd, right, up math32.Vector3) {
	cam.GetNode().UpdateMatrixWorld()
	cam.GetNode().WorldPosition(&pos)
	cam.GetNode().WorldDirection(&fwd)
	fwd.Negate()
	if fwd.Length() < 0.01 {
		fwd.Set(0, 0, 1)
	} else {
		fwd.Normalize()
	}
	worldUp := math32.Vector3{0, 1, 0}
	right.CrossVectors(&worldUp, &fwd)
	if right.Length() < 0.01 {
		worldUp = math32.Vector3{0, 0, 1}
		right.CrossVectors(&worldUp, &fwd)
	}
	right.Normalize()
	up.CrossVectors(&fwd, &right)
	up.Normalize()
	return pos, fwd, right, up
}

func frustumSliceCorners(pos, fwd, right, up math32.Vector3, fovDeg, aspect, near, far float32) [8]math32.Vector3 {
	if aspect < 0.1 {
		aspect = 1
	}
	if fovDeg < 10 {
		fovDeg = 60
	}
	tanHalf := math32.Tan(fovDeg * math32.Pi / 180 * 0.5)
	corner := func(dist, sx, sy float32) math32.Vector3 {
		h := tanHalf * dist
		wd := h * aspect
		off := math32.Vector3{}
		off.Copy(&fwd).MultiplyScalar(dist)
		r := math32.Vector3{}
		r.Copy(&right).MultiplyScalar(sx * wd)
		u := math32.Vector3{}
		u.Copy(&up).MultiplyScalar(sy * h)
		out := math32.Vector3{}
		out.Copy(&pos).Add(&off).Add(&r).Add(&u)
		return out
	}
	return [8]math32.Vector3{
		corner(near, -1, -1),
		corner(near, -1, 1),
		corner(near, 1, -1),
		corner(near, 1, 1),
		corner(far, -1, -1),
		corner(far, -1, 1),
		corner(far, 1, -1),
		corner(far, 1, 1),
	}
}

func (w *World) buildCascadeViews(cam *camera.Camera, n int) {
	s := &w.shadow
	s.tmpDir = w.shadowLightDir()
	pos, fwd, camRight, camUp := cameraViewBasis(cam)
	s.tmpCamPos = pos
	s.tmpCamDir = fwd

	lightDir := s.tmpDir
	s.tmpUp = math32.Vector3{0, 1, 0}
	if math32.Abs(lightDir.Dot(&s.tmpUp)) > 0.92 {
		s.tmpUp = math32.Vector3{0, 0, 1}
	}
	lightRight := math32.Vector3{}
	lightRight.CrossVectors(&s.tmpUp, &lightDir)
	if lightRight.Length() < 0.01 {
		s.tmpUp = math32.Vector3{0, 0, 1}
		lightRight.CrossVectors(&s.tmpUp, &lightDir)
	}
	lightRight.Normalize()
	s.tmpUp.CrossVectors(&lightDir, &lightRight)
	s.tmpUp.Normalize()

	fov := cam.Fov()
	aspect := cam.Aspect()
	nearCam := w.shadowNear(cam)
	zPad := w.cascadeZPad()
	prevSplit := nearCam
	mapSize := float32(s.size)
	if mapSize < 256 {
		mapSize = 256
	}

	for i := 0; i < n; i++ {
		splitNear := prevSplit
		splitFar := s.splits[i]
		prevSplit = splitFar
		corners := frustumSliceCorners(pos, fwd, camRight, camUp, fov, aspect, splitNear, splitFar)

		minR, maxR := float32(math.MaxFloat32), float32(-math.MaxFloat32)
		minU, maxU := float32(math.MaxFloat32), float32(-math.MaxFloat32)
		minD, maxD := float32(math.MaxFloat32), float32(-math.MaxFloat32)
		for ci := 0; ci < 8; ci++ {
			c := corners[ci]
			r := c.Dot(&lightRight)
			u := c.Dot(&s.tmpUp)
			d := c.Dot(&lightDir)
			if r < minR {
				minR = r
			}
			if r > maxR {
				maxR = r
			}
			if u < minU {
				minU = u
			}
			if u > maxU {
				maxU = u
			}
			if d < minD {
				minD = d
			}
			if d > maxD {
				maxD = d
			}
		}

		// A receiver-frustum fit alone is too tight: a platform can be just
		// outside the camera while its shadow is already visible inside it.
		// Keep a bounded light-space guard band so those casters do not pop in
		// at a cascade edge as the opening camera settles or turns.
		rawSpan := max32(maxR-minR, maxU-minU)
		guard := rawSpan * cascadeGuardFraction
		if guard < minCascadeGuard {
			guard = minCascadeGuard
		}
		if guard > maxCascadeGuard {
			guard = maxCascadeGuard
		}
		minR -= guard
		maxR += guard
		minU -= guard
		maxU += guard

		span := max32(maxR-minR, maxU-minU)
		if span < 2 {
			span = 2
		}
		texel := span / mapSize
		if texel < 0.001 {
			texel = 0.001
		}
		span += texel * 4
		texel = span / mapSize
		s.texelWorld[i] = texel

		midR := snapShadow((minR+maxR)*0.5, texel)
		midU := snapShadow((minU+maxU)*0.5, texel)
		midD := (minD + maxD) * 0.5

		s.tmpCenter.Copy(&lightRight).MultiplyScalar(midR)
		s.tmpOff.Copy(&s.tmpUp).MultiplyScalar(midU)
		s.tmpCenter.Add(&s.tmpOff)
		s.tmpOff.Copy(&lightDir).MultiplyScalar(midD)
		s.tmpCenter.Add(&s.tmpOff)

		halfZ := (maxD - minD) * 0.5
		if halfZ < 1 {
			halfZ = 1
		}
		pull := halfZ + zPad
		s.tmpOff.Copy(&lightDir).MultiplyScalar(pull)
		s.tmpEye.Copy(&s.tmpCenter).Add(&s.tmpOff)

		lc := s.orthoCam(1, 0.5, pull+halfZ+zPad, span)
		lc.SetPosition(s.tmpEye.X, s.tmpEye.Y, s.tmpEye.Z)
		lc.LookAt(&s.tmpCenter, &s.tmpUp)
		lc.GetNode().UpdateMatrixWorld()
		lc.ViewMatrix(&s.tmpView)
		lc.ProjMatrix(&s.tmpProj)
		s.lightVP[i].MultiplyMatrices(&s.tmpProj, &s.tmpView)
	}
}
