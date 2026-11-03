package runtime

import "github.com/go-gl/gl/v3.3-core/gl"

// Anisotropic filtering keeps ground sharp toward the horizon, where plain
// mipmapping blurs it. It is an extension, near universal but not core until
// GL 4.6, so it is checked at startup. Each tier caps it, since it adds texture
// reads at grazing angles.
const (
	glTextureMaxAnisotropy = 0x84FE
	glMaxAnisotropy        = 0x84FF
)

var anisoByTier = [...]float32{qualityLow: 2, qualityMedium: 4, qualityHigh: 8, qualityUltra: 16}

// applyAnisotropy sets the level on the texture just bound to the active unit.
func (w *World) applyAnisotropy() {
	limit := w.glmod.caps.maxAniso
	if limit < 1 {
		return
	}
	level := anisoByTier[qualityHigh]
	if w.quality.set {
		level = anisoByTier[w.quality.tier]
	}
	if level > limit {
		level = limit
	}
	gl.TexParameterf(gl.TEXTURE_2D, glTextureMaxAnisotropy, level)
}
