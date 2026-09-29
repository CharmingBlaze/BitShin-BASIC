package runtime

import (
	"math"
	"reflect"
	"unsafe"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/light"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/texture"
)

// Falloff modes for point and spot lights.
// smooth: artist window, bright at the lamp and zero at LightRange.
// classic: LearnOpenGL 1/(constant + linear·d + quadratic·d²).
// physical: glTF KHR_lights_punctual inverse-square with a range window.
const (
	falloffSmooth   = 0
	falloffClassic  = 1
	falloffPhysical = 2
)

// shadeLightGLSL is the shared direct-light model for mbshadow, mbphysical,
// terrain, and water.
//
// The third decay component (G3N's unused padding float) selects the curve:
//
//	< -0.5  physical inverse-square window (linear stores 1/range)
//	~0 and quadratic ~0  smooth LightRange window
//	otherwise  LearnOpenGL polynomial, constant defaults to 1 when stored as 0
const shadeLightGLSL = `
#if POINT_LIGHTS>0
#define PointLightConstant(a) PointLight[3*(a)+2].z
#endif
#if SPOT_LIGHTS>0
#define SpotLightConstant(a) SpotLight[5*(a)+4].y
#endif

uniform sampler2D SpotCookie0;
uniform sampler2D SpotCookie1;
uniform int SpotCookie0On;
uniform int SpotCookie1On;

float spotCookieSample(sampler2D cookie, vec3 fromLight, vec3 spotDir, float cutoffDeg) {
    vec3 dir = normalize(spotDir);
    float along = dot(fromLight, dir);
    if (along <= 0.02) { return 0.0; }
    vec3 upv = abs(dir.y) < 0.99 ? vec3(0.0, 1.0, 0.0) : vec3(1.0, 0.0, 0.0);
    vec3 right = normalize(cross(upv, dir));
    vec3 up2 = cross(dir, right);
    float rad = max(along * tan(radians(clamp(cutoffDeg, 1.0, 89.0))), 0.001);
    vec2 uv = vec2(dot(fromLight, right), dot(fromLight, up2)) / rad;
    uv = uv * 0.5 + 0.5;
    if (uv.x < 0.0 || uv.y < 0.0 || uv.x > 1.0 || uv.y > 1.0) { return 0.0; }
    return texture(cookie, uv).r;
}

float shadeCookie(int index, vec3 fromLight, vec3 spotDir, float cutoffDeg) {
    if (index == 0 && SpotCookie0On != 0) { return spotCookieSample(SpotCookie0, fromLight, spotDir, cutoffDeg); }
    if (index == 1 && SpotCookie1On != 0) { return spotCookieSample(SpotCookie1, fromLight, spotDir, cutoffDeg); }
    return 1.0;
}

float shadeAttenuation(float dist, float lin, float quad, float constant) {
    dist = max(dist, 0.0);
    if (constant < -0.5) {
        float range = 1.0 / max(lin, 1.0e-4);
        float d2 = max(dist * dist, 1.0e-4);
        float window = clamp(1.0 - pow(clamp(sqrt(d2) / range, 0.0, 1.0), 4.0), 0.0, 1.0);
        return window / d2;
    }
    if (quad <= 1.0e-4 && constant <= 1.0e-4) {
        float range = 1.0 / max(lin, 1.0e-4);
        float x = clamp(dist / range, 0.0, 1.0);
        float edge = 1.0 - x;
        edge *= edge;
        return edge / (1.0 + 3.0 * x * x);
    }
    float c = constant;
    if (c < 1.0e-4) { c = 1.0; }
    return 1.0 / (c + lin * dist + quad * dist * dist);
}

float shadeSpot(vec3 toLight, vec3 spotDir, float cutoffDeg, float angular, float constant) {
    float outerDeg = clamp(cutoffDeg, 1.0, 89.0);
    float innerDeg = outerDeg * 0.62;
    if (angular > 0.5 && angular < outerDeg) {
        innerDeg = angular;
    }
    float cosOuter = cos(radians(outerDeg));
    float cosInner = cos(radians(max(innerDeg, 0.5)));
    float cosTheta = dot(normalize(-toLight), normalize(spotDir));
    if (constant < -0.5) {
        float scale = 1.0 / max(0.001, cosInner - cosOuter);
        float cone = clamp(cosTheta * scale + (-cosOuter * scale), 0.0, 1.0);
        return cone * cone;
    }
    return smoothstep(cosOuter, max(cosInner, cosOuter + 0.001), cosTheta);
}

float shadeSpec(float ndh, float shininess) {
    float n = max(shininess, 1.0);
    float norm = min((n + 2.0) * 0.125, 2.0);
    return norm * pow(max(ndh, 0.0), n);
}

vec3 shadeKnee(vec3 c) {
    c = max(c, vec3(0.0));
    vec3 over = max(c - vec3(1.0), vec3(0.0));
    return min(c, vec3(1.0)) + over / (vec3(1.0) + over);
}
`

// applyLightRange stores a photometric radius. Quadratic decay 0 selects the
// windowed curve in shadeAttenuation; linear decay is 1/range so shadow
// cameras (entityLightRange) use the same distance.
func applyLightRange(node interface{ SetLinearDecay(float32) }, rng float32) {
	if rng < 0.25 {
		rng = 0.25
	}
	node.SetLinearDecay(1 / rng)
	if q, ok := node.(interface{ SetQuadraticDecay(float32) }); ok {
		q.SetQuadraticDecay(0)
	}
	setLightConstant(node, 0)
}

func setLightConstant(node interface{}, c float32) {
	switch n := node.(type) {
	case *light.Point:
		setUnexportedFloat(n, "udata", "dummy", c)
	case *light.Spot:
		setUnexportedFloat(n, "udata", "dummy1", c)
	}
}

func lightConstant(node interface{}) float32 {
	switch n := node.(type) {
	case *light.Point:
		return unexportedFloat(n, "udata", "dummy")
	case *light.Spot:
		return unexportedFloat(n, "udata", "dummy1")
	}
	return 0
}

func setUnexportedFloat(ptr any, structName, field string, v float32) {
	f := unexportedField(ptr, structName, field)
	if !f.IsValid() {
		return
	}
	reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem().SetFloat(float64(v))
}

func unexportedFloat(ptr any, structName, field string) float32 {
	f := unexportedField(ptr, structName, field)
	if !f.IsValid() {
		return 0
	}
	return float32(reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem().Float())
}

func unexportedField(ptr any, structName, field string) reflect.Value {
	rv := reflect.ValueOf(ptr)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return reflect.Value{}
	}
	udata := rv.Elem().FieldByName(structName)
	if !udata.IsValid() {
		return reflect.Value{}
	}
	return udata.FieldByName(field)
}

func (w *World) setLightRange(e *Entity, rng float32) {
	if e == nil {
		return
	}
	if rng < 0.25 {
		rng = 0.25
	}
	e.lgtRange = rng
	node, ok := e.node.(interface{ SetLinearDecay(float32) })
	if !ok {
		return
	}
	if e.lgtFalloff == falloffClassic {
		return
	}
	applyLightRange(node, rng)
	if e.lgtFalloff == falloffPhysical {
		setLightConstant(e.node, -1)
	}
}

func (w *World) setLightAttenuation(e *Entity, constant, linear, quadratic, rng float32) {
	if e == nil {
		return
	}
	if constant < 0 {
		constant = 0
	}
	e.lgtFalloff = falloffClassic
	e.lgtC, e.lgtLin, e.lgtQuad = constant, linear, quadratic
	if p, ok := e.node.(interface{ SetLinearDecay(float32) }); ok {
		p.SetLinearDecay(linear)
	}
	if q, ok := e.node.(interface{ SetQuadraticDecay(float32) }); ok {
		q.SetQuadraticDecay(quadratic)
	}
	setLightConstant(e.node, constant)
	if rng <= 0 {
		rng = classicLightRange(constant, linear, quadratic)
	}
	if rng < 0.25 {
		rng = 0.25
	}
	e.lgtRange = rng
}

func classicLightRange(c, lin, quad float32) float32 {
	if c < 1e-4 {
		c = 1
	}
	lo, hi := float32(0.25), float32(500)
	for i := 0; i < 24; i++ {
		mid := (lo + hi) * 0.5
		att := 1 / (c + lin*mid + quad*mid*mid)
		if att > 0.04 {
			lo = mid
		} else {
			hi = mid
		}
	}
	return hi
}

func (w *World) setLightFalloff(e *Entity, mode string) {
	if e == nil {
		return
	}
	rng := e.lgtRange
	if rng < 0.25 {
		rng = entityLightRange(e)
	}
	switch mode {
	case "physical", "inverse", "gltf", "punctual", "2":
		e.lgtFalloff = falloffPhysical
		if p, ok := e.node.(interface{ SetLinearDecay(float32) }); ok {
			applyLightRange(p, rng)
		}
		setLightConstant(e.node, -1)
		e.lgtRange = rng
	case "classic", "phong", "learnopengl", "1":
		c, lin, quad := e.lgtC, e.lgtLin, e.lgtQuad
		if c == 0 && lin == 0 && quad == 0 {
			c, lin, quad = 1, 0.09, 0.032
		}
		w.setLightAttenuation(e, c, lin, quad, rng)
	default:
		e.lgtFalloff = falloffSmooth
		if p, ok := e.node.(interface{ SetLinearDecay(float32) }); ok {
			applyLightRange(p, rng)
		}
		e.lgtRange = rng
	}
}

func (w *World) litMatOf(e *Entity) *litMat {
	if e == nil || e.mesh == nil {
		return nil
	}
	lm, _ := e.mesh.GetMaterial(0).(*litMat)
	return lm
}

func (w *World) setPhongMap(e *Entity, tex *texture.Texture2D, spec bool) {
	lm := w.litMatOf(e)
	if lm == nil || tex == nil {
		return
	}
	if spec {
		lm.specMap = tex
	} else {
		lm.emitMap = tex
	}
	if e.mat != nil {
		e.mat.SetShader("bsshadow")
	}
}

func bindNamedMap(gs *gls.GLS, tex *texture.Texture2D, unit int, sampler string) {
	if gs == nil || tex == nil {
		return
	}
	a, b := tex.GetUniformNames()
	tex.SetUniformNames(sampler, sampler+"Info")
	tex.RenderSetup(gs, unit, 0)
	if a != "" || b != "" {
		tex.SetUniformNames(a, b)
	}
}

func (w *World) eachSceneNode(fn func(core.INode)) {
	if w == nil || w.scene == nil || fn == nil {
		return
	}
	var walk func(core.INode)
	walk = func(n core.INode) {
		if n == nil || n.GetNode() == nil {
			return
		}
		fn(n)
		for _, ch := range n.GetNode().Children() {
			walk(ch)
		}
	}
	walk(w.scene)
}

func lightRadiance(n core.INode) (math32.Color, float32) {
	type ci interface {
		Color() math32.Color
		Intensity() float32
	}
	if l, ok := n.(ci); ok {
		return l.Color(), l.Intensity()
	}
	return math32.Color{1, 1, 1}, 1
}

func (w *World) uploadPhongLightTerms(gs *gls.GLS) {
	if gs == nil {
		return
	}
	var dirS, dirA, ptS, ptA, spS, spA [8]math32.Color
	di, pi, si := 0, 0, 0
	w.eachSceneNode(func(n core.INode) {
		var slot *[8]math32.Color
		var amb *[8]math32.Color
		var idx *int
		switch n.(type) {
		case *light.Directional:
			slot, amb, idx = &dirS, &dirA, &di
		case *light.Point:
			slot, amb, idx = &ptS, &ptA, &pi
		case *light.Spot:
			slot, amb, idx = &spS, &spA, &si
		default:
			return
		}
		if *idx >= 8 {
			return
		}
		col, inten := lightRadiance(n)
		spec := col
		spec.MultiplyScalar(inten)
		var extra math32.Color
		for _, e := range w.ents {
			if e == nil || e.node != n {
				continue
			}
			if e.lgtSpecOn {
				spec = e.lgtSpec
				spec.MultiplyScalar(inten)
			}
			if e.lgtAmbOn {
				extra = e.lgtAmb
				extra.MultiplyScalar(inten)
			}
			break
		}
		slot[*idx] = spec
		amb[*idx] = extra
		*idx++
	})
	setUni3Array(gs, "DirSpec", &dirS)
	setUni3Array(gs, "DirAmb", &dirA)
	setUni3Array(gs, "PointSpec", &ptS)
	setUni3Array(gs, "PointAmb", &ptA)
	setUni3Array(gs, "SpotSpec", &spS)
	setUni3Array(gs, "SpotAmb", &spA)
}

func setUni3Array(gs *gls.GLS, name string, cols *[8]math32.Color) {
	u := gls.Uniform{}
	u.Init(name)
	if loc := u.Location(gs); loc >= 0 {
		gs.Uniform3fv(loc, 8, &cols[0].R)
	}
}

func (w *World) setEnvHemi(sky, ground math32.Color) {
	if w.shaderUnis == nil {
		w.shaderUnis = map[string]shaderUni{}
	}
	w.shaderUnis["ProbeEnabled"] = shaderUni{n: 1, v: [4]float32{1}}
	w.shaderUnis["ProbeSky"] = shaderUni{n: 3, v: [4]float32{sky.R, sky.G, sky.B}}
	w.shaderUnis["ProbeGround"] = shaderUni{n: 3, v: [4]float32{ground.R, ground.G, ground.B}}
}

func (w *World) eachLocalLight(fn func(*Entity)) {
	for _, e := range w.ents {
		if e == nil || e.lgtKind < 1 || e.lgtKind > 3 {
			continue
		}
		fn(e)
	}
}

func lightIntensityOf(e *Entity) (float32, bool) {
	if e == nil || e.node == nil {
		return 0, false
	}
	in, ok := e.node.(interface{ Intensity() float32 })
	if !ok {
		return 0, false
	}
	return in.Intensity(), true
}

func setLightIntensityOf(e *Entity, v float32) {
	if e == nil || e.lgtI == nil {
		return
	}
	e.lgtI.SetIntensity(v)
}

// applyLightEnvironment builds a daylight rig or a practical interior.
// Outdoor is a directional sun (lux-style, no distance falloff) plus a sky/ground
// hemisphere. Indoor dims that sun to a window and switches point and spot
// lights to glTF inverse-square so fixtures fall off the way real rooms do.
func (w *World) applyLightEnvironment(mode string) int {
	switch mode {
	case "outdoor", "exterior", "day", "sun", "1":
		w.lightEnv = 1
	case "indoor", "interior", "room", "2":
		w.lightEnv = 2
	default:
		w.lightEnv = 0
		return 0
	}
	if w.ambient == nil && w.scene != nil {
		w.ambient = light.NewAmbient(&math32.Color{0.28, 0.30, 0.36}, 1)
		w.scene.Add(w.ambient)
	}
	if w.lightEnv == 1 {
		hasSun := false
		w.eachLocalLight(func(e *Entity) {
			if e.lgtKind != 1 {
				if e.lgtEnvSaved {
					setLightIntensityOf(e, e.lgtEnvI)
					switch e.lgtEnvFall {
					case falloffClassic:
						w.setLightAttenuation(e, e.lgtC, e.lgtLin, e.lgtQuad, e.lgtRange)
					case falloffPhysical:
						w.setLightFalloff(e, "physical")
					default:
						w.setLightFalloff(e, "smooth")
					}
					e.lgtEnvSaved = false
				}
				return
			}
			hasSun = true
			if e.lgtEnvSaved {
				setLightIntensityOf(e, e.lgtEnvI)
				e.lgtEnvSaved = false
			}
			if cur, ok := lightIntensityOf(e); ok && cur < 4 {
				setLightIntensityOf(e, 1.45)
			}
			if e.lgt != nil {
				e.lgt.SetColor(&math32.Color{1, 0.95, 0.86})
			}
		})
		if !hasSun && w.scene != nil {
			id := w.makeLight(1, 0)
			if e := w.ents[id]; e != nil {
				setLightIntensityOf(e, 1.45)
				if e.lgt != nil {
					e.lgt.SetColor(&math32.Color{1, 0.95, 0.86})
				}
			}
		}
		if w.ambient != nil {
			w.ambient.SetColor(&math32.Color{0.16, 0.18, 0.22})
			w.ambient.SetIntensity(1)
		}
		w.setEnvHemi(math32.Color{0.42, 0.52, 0.68}, math32.Color{0.20, 0.16, 0.11})
		w.clear = math32.Color{0.42, 0.58, 0.78}
		w.fogMode = 1
		w.fogNear, w.fogFar = 48, 220
		w.fogRGB = math32.Color{0.55, 0.68, 0.82}
		w.shadow.distance = 240
		w.shadow.fadeNear, w.shadow.fadeFar = 200, 240
		w.post.exposure = 1.05
		w.post.tonemap = 1
		w.applyLitShaders()
		return 1
	}
	w.eachLocalLight(func(e *Entity) {
		cur, ok := lightIntensityOf(e)
		if e.lgtKind == 1 {
			if ok && !e.lgtEnvSaved {
				e.lgtEnvI = cur
				e.lgtEnvSaved = true
			}
			setLightIntensityOf(e, 0.28)
			if e.lgt != nil {
				e.lgt.SetColor(&math32.Color{1, 0.90, 0.74})
			}
			return
		}
		if !e.lgtEnvSaved {
			e.lgtEnvI = cur
			e.lgtEnvFall = e.lgtFalloff
			e.lgtEnvSaved = true
			if ok && cur < 20 {
				if e.lgtKind == 3 {
					setLightIntensityOf(e, 160)
				} else {
					setLightIntensityOf(e, 110)
				}
			}
		}
		w.setLightFalloff(e, "physical")
	})
	if w.ambient != nil {
		w.ambient.SetColor(&math32.Color{0.035, 0.032, 0.028})
		w.ambient.SetIntensity(1)
	}
	w.setEnvHemi(math32.Color{0.07, 0.065, 0.055}, math32.Color{0.03, 0.026, 0.022})
	w.clear = math32.Color{0.04, 0.04, 0.045}
	w.fogMode = 0
	w.shadow.distance = 40
	w.shadow.fadeNear, w.shadow.fadeFar = 24, 40
	w.post.exposure = 1.45
	w.post.tonemap = 1
	w.applyLitShaders()
	return 2
}

func (w *World) bindSpotCookies(gs *gls.GLS) {
	if gs == nil {
		return
	}
	var cookies [2]*texture.Texture2D
	n := 0
	w.eachSceneNode(func(node core.INode) {
		if n >= 2 {
			return
		}
		if _, ok := node.(*light.Spot); !ok {
			return
		}
		for _, e := range w.ents {
			if e != nil && e.node == node && e.lgtCookie != nil {
				cookies[n] = e.lgtCookie
				break
			}
		}
		n++
	})
	setUni1i(gs, "SpotCookie0On", bool01(cookies[0] != nil))
	setUni1i(gs, "SpotCookie1On", bool01(cookies[1] != nil))
	if cookies[0] != nil {
		bindNamedMap(gs, cookies[0], 12, "SpotCookie0")
	}
	if cookies[1] != nil {
		bindNamedMap(gs, cookies[1], 13, "SpotCookie1")
	}
}

func (w *World) setPhongNormal(e *Entity, tex *texture.Texture2D) {
	lm := w.litMatOf(e)
	if lm == nil || tex == nil {
		return
	}
	lm.normMap = tex
	if e.mat != nil {
		e.mat.SetShader("bsshadow")
	}
}

func kelvinToRGB(k float64) math32.Color {
	if k < 1000 {
		k = 1000
	}
	if k > 40000 {
		k = 40000
	}
	temp := k / 100
	var r, g, b float64
	if temp <= 66 {
		r = 255
		g = 99.4708025861*math.Log(temp) - 161.1195681661
		if temp <= 19 {
			b = 0
		} else {
			b = 138.5177312231*math.Log(temp-10) - 305.0447927307
		}
	} else {
		r = 329.698727446 * math.Pow(temp-60, -0.1332047592)
		g = 288.1221695283 * math.Pow(temp-60, -0.0755148492)
		b = 255
	}
	clamp := func(v float64) float32 {
		if v < 0 {
			v = 0
		}
		if v > 255 {
			v = 255
		}
		return float32(v / 255)
	}
	return math32.Color{clamp(r), clamp(g), clamp(b)}
}

func (w *World) setLightTemperature(e *Entity, kelvin float64) {
	if e == nil || e.lgt == nil {
		return
	}
	c := kelvinToRGB(kelvin)
	e.lgtKelvin = float32(kelvin)
	e.lgt.SetColor(&c)
}

func (w *World) setLightEnabled(e *Entity, on bool) {
	if e == nil {
		return
	}
	if !on {
		if !e.lgtDisabled {
			if cur, ok := lightIntensityOf(e); ok {
				e.lgtOffI = cur
			}
			e.lgtDisabled = true
		}
		setLightIntensityOf(e, 0)
	} else if e.lgtDisabled {
		setLightIntensityOf(e, e.lgtOffI)
		e.lgtDisabled = false
	}
	if e.node != nil && e.node.GetNode() != nil {
		e.node.GetNode().SetVisible(on)
	}
}

func (w *World) ensureSun() *Entity {
	var sun *Entity
	w.eachLocalLight(func(e *Entity) {
		if e.lgtKind == 1 && sun == nil {
			sun = e
		}
	})
	if sun == nil && w.scene != nil {
		id := w.makeLight(1, 0)
		sun = w.ents[id]
	}
	return sun
}

// setTimeOfDay moves the sun through a day. 6 is sunrise, 12 noon, 18 sunset, 0 midnight.
func (w *World) setTimeOfDay(hours float64) {
	for hours < 0 {
		hours += 24
	}
	for hours >= 24 {
		hours -= 24
	}
	w.timeOfDay = hours
	w.lightEnv = 1
	elev := math.Sin((hours - 6) / 12 * math.Pi)
	pitch := elev * 72
	yaw := math.Mod(hours*15, 360)
	sun := w.ensureSun()
	if sun != nil {
		w.setLightDir(sun, pitch, yaw, 0)
		kelvin := 5600.0
		inten := 0.08
		if elev > 0.2 {
			kelvin = 5600
			inten = 0.35 + elev*1.15
		} else if elev > 0 {
			kelvin = 2500 + (elev/0.2)*3100
			inten = 0.2 + elev*1.2
		} else {
			kelvin = 11000
			inten = 0.05
		}
		w.setLightTemperature(sun, kelvin)
		setLightIntensityOf(sun, float32(inten))
	}
	sky := math32.Color{0.05, 0.07, 0.12}
	ground := math32.Color{0.02, 0.02, 0.025}
	amb := math32.Color{0.02, 0.025, 0.04}
	if elev > 0 {
		t := float32(elev)
		if t > 1 {
			t = 1
		}
		sky = math32.Color{0.15 + 0.30*t, 0.18 + 0.36*t, 0.28 + 0.40*t}
		ground = math32.Color{0.08 + 0.12*t, 0.06 + 0.10*t, 0.04 + 0.06*t}
		amb = math32.Color{0.05 + 0.12*t, 0.06 + 0.13*t, 0.08 + 0.14*t}
		w.clear = math32.Color{0.25 + 0.25*t, 0.35 + 0.28*t, 0.55 + 0.25*t}
	} else {
		w.clear = math32.Color{0.02, 0.025, 0.05}
	}
	w.setEnvHemi(sky, ground)
	if w.ambient == nil && w.scene != nil {
		w.ambient = light.NewAmbient(&amb, 1)
		w.scene.Add(w.ambient)
	}
	if w.ambient != nil {
		w.ambient.SetColor(&amb)
		w.ambient.SetIntensity(1)
	}
}

func (w *World) createThreePoint() int {
	key := w.makeLight(1, 0)
	if e := w.ents[key]; e != nil {
		w.setLightDir(e, 48, 35, 0)
		setLightIntensityOf(e, 1.35)
		w.setLightTemperature(e, 5200)
		if e.node != nil {
			e.node.GetNode().SetName("key")
		}
		e.name = "key"
	}
	fill := w.makeLight(1, 0)
	if e := w.ents[fill]; e != nil {
		w.setLightDir(e, 20, -40, 0)
		setLightIntensityOf(e, 0.38)
		w.setLightTemperature(e, 7500)
		e.castShadow = false
		e.name = "fill"
	}
	rim := w.makeLight(1, 0)
	if e := w.ents[rim]; e != nil {
		w.setLightDir(e, 28, 160, 0)
		setLightIntensityOf(e, 0.72)
		w.setLightTemperature(e, 6500)
		e.castShadow = false
		e.name = "rim"
	}
	return key
}

func tuneLocalLight(node interface{}) {
	switch l := node.(type) {
	case *light.Point:
		l.SetIntensity(2.6)
		applyLightRange(l, 14)
	case *light.Spot:
		l.SetIntensity(3.4)
		applyLightRange(l, 16)
		l.SetAngularDecay(18)
		l.SetCutoffAngle(40)
	}
}
