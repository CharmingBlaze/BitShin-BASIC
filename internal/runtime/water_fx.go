package runtime

import (
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/g3n/engine/gls"
	"github.com/go-gl/gl/v3.3-core/gl"

	"bitshinbasic/internal/value"
)

const wakeRes = 96

func (w *World) ensureWaterMaps(wb *waterBody) {
	if wb == nil {
		return
	}
	if wb.dudvGL == 0 {
		wb.dudvGL = uploadWaterRGBA(genWaterDuDv(256))
	}
	if wb.normGL == 0 {
		wb.normGL = uploadWaterRGBA(genWaterNormal(256))
	}
	if wb.causticGL == 0 {
		wb.causticGL = uploadWaterRGBA(genWaterCaustic(256))
	}
	w.ensureWakeTex(wb)
}

func genWaterCaustic(n int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	cells := 7.0
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			u := float64(x) / float64(n) * cells
			v := float64(y) / float64(n) * cells
			d1 := causticCell(u, v)
			d2 := causticCell(u*1.7+1.3, v*1.7+0.4)
			ridge := math.Pow(1-math.Min(d1, 1), 6) + 0.45*math.Pow(1-math.Min(d2, 1), 5)
			ridge = math.Min(ridge, 1)
			g := uint8(ridge * 255)
			img.SetRGBA(x, y, color.RGBA{g, uint8(ridge * 220), uint8(ridge * 180), 255})
		}
	}
	return img
}

func causticCell(x, y float64) float64 {
	ix, iy := math.Floor(x), math.Floor(y)
	fx, fy := x-ix, y-iy
	best := 1.0
	for j := -1; j <= 1; j++ {
		for i := -1; i <= 1; i++ {
			px := waterHash2(int(ix)+i, int(iy)+j, 512)
			py := waterHash2(int(ix)+i+19, int(iy)+j+7, 512)
			dx := float64(i) + px - fx
			dy := float64(j) + py - fy
			d := math.Sqrt(dx*dx + dy*dy)
			if d < best {
				best = d
			}
		}
	}
	return best
}

func (w *World) ensureWakeTex(wb *waterBody) {
	if wb == nil {
		return
	}
	if len(wb.wakeH) != wakeRes*wakeRes {
		wb.wakeH = make([]float32, wakeRes*wakeRes)
		wb.wakeV = make([]float32, wakeRes*wakeRes)
		wb.wakePix = make([]uint8, wakeRes*wakeRes*4)
	}
	if wb.wakeSpan <= 8 {
		wb.wakeSpan = 140
	}
	if wb.wakeGL != 0 {
		return
	}
	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, wakeRes, wakeRes, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(wb.wakePix))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	wb.wakeGL = tex
}

func (w *World) wakeOrigin(wb *waterBody) (ox, oz float32) {
	if e := w.ents[wb.ent]; e != nil && e.node != nil {
		p := worldPos(e.node.GetNode())
		ox, _, oz = fromG3N(p.X, p.Y, p.Z)
	}
	return ox, oz
}

func (w *World) wakeImpulse(wb *waterBody, x, z, amp float32) {
	if wb == nil || amp == 0 {
		return
	}
	w.ensureWakeTex(wb)
	ox, oz := w.wakeOrigin(wb)
	span := wb.wakeSpan
	if span < 8 {
		span = 140
	}
	u := (x-ox)/span + 0.5
	v := (z-oz)/span + 0.5
	if u < 0.02 || u > 0.98 || v < 0.02 || v > 0.98 {
		return
	}
	cx := u * float32(wakeRes-1)
	cy := v * float32(wakeRes-1)
	rad := float32(3.2)
	for j := -4; j <= 4; j++ {
		for i := -4; i <= 4; i++ {
			ix := int(cx) + i
			iy := int(cy) + j
			if ix < 1 || iy < 1 || ix >= wakeRes-1 || iy >= wakeRes-1 {
				continue
			}
			dx := float32(ix) - cx
			dy := float32(iy) - cy
			d2 := dx*dx + dy*dy
			if d2 > rad*rad {
				continue
			}
			k := amp * float32(math.Exp(float64(-d2/(rad*0.55))))
			wb.wakeH[iy*wakeRes+ix] += k
		}
	}
}

func (w *World) tickWake(wb *waterBody, dt float32) {
	if wb == nil {
		return
	}
	w.ensureWakeTex(wb)
	if dt <= 0 {
		dt = 1.0 / 60
	}
	if dt > 0.05 {
		dt = 0.05
	}
	h := wb.wakeH
	vel := wb.wakeV
	n := wakeRes
	c2 := float32(18.0)
	damp := float32(math.Pow(0.96, float64(dt*60)))
	for y := 1; y < n-1; y++ {
		row := y * n
		for x := 1; x < n-1; x++ {
			i := row + x
			lap := h[i-1] + h[i+1] + h[i-n] + h[i+n] - 4*h[i]
			vel[i] += (c2*lap - h[i]*2.2) * dt
			vel[i] *= damp
		}
	}
	pix := wb.wakePix
	for y := 1; y < n-1; y++ {
		for x := 1; x < n-1; x++ {
			i := y*n + x
			h[i] += vel[i] * dt
			if h[i] > 1.5 {
				h[i] = 1.5
			}
			if h[i] < -1.5 {
				h[i] = -1.5
			}
			dx := (h[i+1] - h[i-1]) * 0.5
			dz := (h[i+n] - h[i-n]) * 0.5
			r := uint8((h[i]*0.5 + 0.5) * 255)
			g := uint8((dx*0.5 + 0.5) * 255)
			b := uint8((dz*0.5 + 0.5) * 255)
			o := i * 4
			pix[o], pix[o+1], pix[o+2], pix[o+3] = r, g, b, 255
		}
	}
	if wb.wakeGL == 0 {
		return
	}
	gl.BindTexture(gl.TEXTURE_2D, wb.wakeGL)
	gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, wakeRes, wakeRes, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pix))
	gl.BindTexture(gl.TEXTURE_2D, 0)
}

func (w *World) bindWaterExtras(gs *gls.GLS) {
	if gs == nil {
		return
	}
	dummy := w.ensureDummyShadowTex()
	wb := w.currentWater()
	level := float32(-9999)
	sx, sy, sz := float32(0.4), float32(0.8), float32(0.3)
	move := float32(0)
	cau := dummy
	on := 0
	if wb != nil {
		level = wb.y
		move = wb.move
		sx, sy, sz = w.waterSunDir()
		if wb.caustics {
			w.ensureWaterMaps(wb)
			if wb.causticGL != 0 {
				cau = wb.causticGL
				on = 1
			}
		}
	}
	bindWaterSampler(gs, 15, cau, "WaterCaustic")
	setUni1i(gs, "WaterCausticsOn", on)
	setUni1f(gs, "WaterCausticLevel", level)
	setUni3f(gs, "WaterCausticSun", sx, sy, sz)
	setUni1f(gs, "WaterCausticMove", move)
}

func (w *World) setWaterAmbientSound(a []value.Value) (value.Value, error) {
	wb := w.currentWater()
	off := 0
	if len(a) >= 2 {
		if id := argI(a, 0, 0); w.waters[id] != nil {
			wb = w.waters[id]
			off = 1
		}
	}
	if wb == nil {
		return value.Num(0), nil
	}
	path := argS(a, off)
	if strings.TrimSpace(path) == "" {
		return value.Num(0), nil
	}
	vol := argN(a, off+1, -1)
	clip, err := w.loadClip([]value.Value{value.Str(path)}, true)
	if err != nil || clip.Kind != value.KindNum || clip.Num <= 0 {
		return value.Num(0), nil
	}
	id := int(clip.Num)
	wb.ambSnd = id
	wb.ambVol = vol
	if s := w.sounds[id]; s != nil {
		if vol >= 0 {
			s.vol = vol
		} else {
			w.ensureWeather()
			s.vol = w.wx.intensity
			if s.vol <= 0 {
				s.vol = 0.35
			}
		}
	}
	_, _ = w.startClip(id, true)
	return value.Num(float64(id)), nil
}

func (w *World) tickWaterAmbient(wb *waterBody) {
	if wb == nil || wb.ambSnd == 0 {
		return
	}
	s := w.sounds[wb.ambSnd]
	if s == nil || s.voice == nil {
		return
	}
	vol := wb.ambVol
	if vol < 0 {
		w.ensureWeather()
		vol = w.wx.intensity
		if vol <= 0 {
			vol = 0.25
		}
	}
	s.vol = vol
	s.voice.SetVolume(vol)
}

func (m *WaterMaterial) bindWakeSSR(gs *gls.GLS) {
	if m == nil || m.wb == nil || gs == nil {
		return
	}
	wb := m.wb
	dummy := uint32(0)
	if m.w != nil {
		dummy = m.w.ensureDummyShadowTex()
	}
	wake := wb.wakeGL
	if wake == 0 {
		wake = dummy
	}
	bindWaterSampler(gs, 12, wake, "WaterWake")
	setUni1i(gs, "WaterHasWake", bool01(wb.wakeGL != 0))
	setUni1f(gs, "WakeAmp", 0.55)
	span := wb.wakeSpan
	if span < 8 {
		span = 140
	}
	setUni1f(gs, "WakeSpan", span)
	setUni1i(gs, "WaterSSROn", bool01(wb.ssr && wb.reflectOn && wb.reflect.color != 0))
}
