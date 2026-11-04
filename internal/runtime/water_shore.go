package runtime

import (
	"github.com/g3n/engine/gls"
	"github.com/go-gl/gl/v3.3-core/gl"
)

// Shallows and foam with no refraction pass. The tiers that have one read the water's
// depth from it each frame. Low has none, and without depth its sea was one flat color
// up to the sand. The depth under the water is known from the height field, so it is
// baked once into a small map: the same turquoise and foam for one texture read.

const (
	shoreMapSize   = 256
	shoreMapRangeM = 16 // meters of depth the map holds; below that is open water
)

// shoreDepths is depth below the sea at each texel, 0 to 255 over shoreMapRangeM.
func shoreDepths(height func(x, z float32) float32, sea, x0, z0, w, d float32, n int) []uint8 {
	out := make([]uint8, n*n)
	for j := 0; j < n; j++ {
		z := z0 + d*(float32(j)+0.5)/float32(n)
		for i := 0; i < n; i++ {
			x := x0 + w*(float32(i)+0.5)/float32(n)
			depth := (sea - height(x, z)) / shoreMapRangeM * 255
			if depth < 0 {
				depth = 0
			}
			if depth > 255 {
				depth = 255
			}
			out[j*n+i] = uint8(depth + 0.5)
		}
	}
	return out
}

// bindShoreMap bakes the map when it is first needed and sends it to the shader. It
// shares the refraction depth's sampler, which this tier leaves unused.
func (w *World) bindShoreMap(gs *gls.GLS, wb *waterBody) {
	var t *terrain
	for _, c := range w.terrains {
		if c != nil && !c.hf.proc && c.hf.worldW > 0 {
			t = c
			break
		}
	}
	if t == nil {
		setUni1i(gs, "WaterShoreOn", 0)
		return
	}
	if wb.shoreTex == 0 || wb.shoreFor != t || wb.shoreSea != wb.y || wb.shoreEdit != t.sculpt.editTick {
		if wb.shoreTex == 0 {
			gl.GenTextures(1, &wb.shoreTex)
		}
		depths := shoreDepths(func(x, z float32) float32 { return w.sampleHeight(&t.hf, x, z) },
			wb.y, t.hf.ox, t.hf.oz, t.hf.worldW, t.hf.worldD, shoreMapSize)
		gl.ActiveTexture(gl.TEXTURE0 + unitWaterRefractDepth)
		gl.BindTexture(gl.TEXTURE_2D, wb.shoreTex)
		gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.R8, shoreMapSize, shoreMapSize, 0, gl.RED, gl.UNSIGNED_BYTE, gl.Ptr(depths))
		gl.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
		gl.ActiveTexture(gl.TEXTURE0) // where G3N's state cache believes it is
		wb.shoreFor, wb.shoreSea, wb.shoreEdit = t, wb.y, t.sculpt.editTick
	}
	bindWaterSampler(gs, unitWaterRefractDepth, wb.shoreTex, "WaterRefractDepth")
	setUni1i(gs, "WaterShoreOn", 1)
	setUni4f(gs, "WaterShoreRect", t.hf.ox, t.hf.oz, t.hf.worldW, t.hf.worldD)
}
