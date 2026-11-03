package runtime

import (
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

// Linear lighting. With it on, the lit shaders turn colors into linear light,
// add the lights up there, and leave the result in the half-float scene target.
// The post pass then applies exposure and the tone curve and encodes sRGB.
// Effects that are not lit (sky, clouds, particles) only decode their output,
// so they look as they did after the round trip. Off, every shader takes its
// old path. It needs the post pass: on the low tier, with no post pass, the
// scene falls back to the old path because nothing would encode the result.

// linearGLSL is the shared switch. toLin is the identity when it is off.
const linearGLSL = `
uniform int LinearLight;
vec3 toLin(vec3 c) { return LinearLight != 0 ? pow(max(c, vec3(0.0)), vec3(2.2)) : c; }
// fogMix blends as the eye sees it. A plain mix in linear light lets a little
// bright fog bleach dark ground and water, and SetFog distances stop meaning
// what they did.
vec3 fogMix(vec3 fog, vec3 c, float f) {
    if (LinearLight == 0) { return mix(fog, c, f); }
    vec3 g = vec3(1.0 / 2.2);
    // Air scatters blue away first, so distance reddens what it transmits and
    // the view fills with sky color. Exponents follow Rayleigh's wavelength
    // falloff, softened: its true ratio turns near land orange.
    vec3 through = pow(vec3(clamp(f, 0.0, 1.0)), vec3(0.65, 1.0, 1.45));
    return pow(mix(pow(max(fog, vec3(0.0)), g), pow(max(c, vec3(0.0)), g), through), vec3(2.2));
}
`

// skyAmbientGLSL lights a surface from the sky picture it stands under, not a
// fixed two-color gradient. The sky map has no ground in it, so light from
// below is the sky's own color darkened and warmed, as if bounced off land.
const skyAmbientGLSL = `
uniform samplerCube uEnvCube;
uniform int HasEnvCube;
vec3 skyAmbient(vec3 n) {
    if (HasEnvCube == 0) { return mix(vec3(0.03, 0.025, 0.02), vec3(0.20, 0.30, 0.45), n.y * 0.5 + 0.5); }
    vec3 c = textureLod(uEnvCube, n, 5.0).rgb;
    if (HasEnvCube == 1) {
        c = toLin(c);
        // Below the horizon is sunlit ground, not sky. A wall sees half of each,
        // or every shaded side of a rock or tree comes out sky blue.
        c = mix(c, vec3(0.20, 0.17, 0.11), 1.0 - smoothstep(-0.35, 0.55, n.y));
    }
    return c;
}
`

// mbskyFragment draws a sky face in linear mode only, so an old-path sky keeps
// G3N's own shader untouched.
const mbskyFragment = `precision highp float;
in vec2 FragTexcoord;
#include <material>
out vec4 FragColor;
` + linearGLSL + `
void main() {
    vec3 c = MatDiffuseColor;
#if MAT_TEXTURES > 0
    if (MatTexVisible(0)) {
        c *= texture(MatTexture[0], FragTexcoord * MatTexRepeat(0) + MatTexOffset(0)).rgb;
    }
#endif
    FragColor = vec4(toLin(c), 1.0);
}
`

// linearActive reports whether this pass draws in linear light.
func (w *World) linearActive() bool {
	return w.linearLight && w.post.on && !w.linearHold
}

// bindLinear tells the current program which path to take.
func (w *World) bindLinear(gs *gls.GLS) {
	setUni1i(gs, "LinearLight", bool01(w.linearActive()))
}

// linearClear is a background color as the current pass needs it.
func (w *World) linearClear(c math32.Color) math32.Color {
	if !w.linearActive() {
		return c
	}
	return math32.Color{R: srgbToLinear(c.R), G: srgbToLinear(c.G), B: srgbToLinear(c.B)}
}

// srgbToLinear is toLin for one channel.
func srgbToLinear(v float32) float32 {
	if v <= 0 {
		return 0
	}
	return math32.Pow(v, 2.2)
}

// linMat tells a plain material's program about linear lighting.
type linMat struct {
	*material.Standard
	w *World
}

func (m *linMat) GetMaterial() *material.Material { return m.Standard.GetMaterial() }
func (m *linMat) Dispose()                        { m.Standard.Dispose() }

func (m *linMat) RenderSetup(gs *gls.GLS) {
	m.Standard.RenderSetup(gs)
	m.w.bindLinear(gs)
}

// syncSkyShaders puts sky faces on the linear shader while it is in use.
func (w *World) syncSkyShaders() {
	name := "standard"
	if w.linearActive() {
		name = "mbsky"
	}
	for _, s := range w.skies {
		if s == nil || s.box == nil {
			continue
		}
		for _, gm := range s.box.Materials() {
			if m, ok := gm.IMaterial().(*linMat); ok && m.Shader() != name {
				m.SetShader(name)
			}
		}
	}
}

func (w *World) linearCommands(n func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		// SetLinearLight on: light the scene in linear space and tone map it.
		// SetLook "modern" turns this on. Needs post effects (not the low tier).
		"setlinearlight": n(func(a []value.Value) (value.Value, error) {
			w.linearLight = argI(a, 0, 1) != 0
			return value.Num(float64(b01(w.linearLight))), nil
		}),
		"getlinearlight": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(b01(w.linearActive()))), nil
		}),
	}
}
