package runtime

import (
	"fmt"
	"math"
	"sort"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
	"github.com/go-gl/gl/v3.3-core/gl"

	"moonbasic/internal/value"
)

// Rectangular area lights: soft-box, window, and screen light. Each is an
// entity, so PositionEntity, TurnEntity, PointEntity, and parenting move it.
// It emits along its -Z, the way PointEntity aims it. Diffuse integrates a 3x3 grid
// over the rectangle; specular aims at the point on the rectangle closest to
// the reflection ray and widens the lobe by the light's apparent size. No
// shadows. PBR and the default lit shader read them; terrain and water do not.

const maxAreaLights = 4

type areaLight struct {
	e         *Entity
	width     float32
	height    float32
	color     math32.Color
	intensity float32
	rng       float32 // metres; 0 is unlimited
	twoSided  bool
	panel     *graphic.Mesh
	panelMat  *pbrMat
}

// areaUniforms is one frame's packed lights, nearest the camera first.
type areaUniforms struct {
	serial int
	n      int
	pos    [maxAreaLights]math32.Vector3
	right  [maxAreaLights]math32.Vector3 // half width
	up     [maxAreaLights]math32.Vector3 // half height
	color  [maxAreaLights]math32.Color
	rng    [maxAreaLights]float32
	two    [maxAreaLights]int32
	cam    math32.Vector3
}

const areaLightGLSL = `
uniform int AreaCount;
uniform vec3 AreaPos[4];
uniform vec3 AreaRight[4];
uniform vec3 AreaUp[4];
uniform vec3 AreaColor[4];
uniform float AreaRange[4];
uniform int AreaTwoSided[4];
uniform vec3 AreaCam;

float areaWindow(int i, vec3 P) {
    if (AreaRange[i] <= 0.0) { return 1.0; }
    float r = length(AreaPos[i] - P) / AreaRange[i];
    float w = clamp(1.0 - r * r * r * r, 0.0, 1.0);
    return w * w;
}

// areaFormFactor is the share of the hemisphere above P the lit rectangle
// covers, cosine weighted: 0..1. Diffuse radiance is albedo * colour * it.
float areaFormFactor(int i, vec3 P, vec3 N) {
    vec3 ln = normalize(cross(AreaRight[i], AreaUp[i]));
    float sum = 0.0;
    for (int y = -1; y <= 1; y++) {
        for (int x = -1; x <= 1; x++) {
            vec3 q = AreaPos[i] + AreaRight[i] * (float(x) * 0.6667) + AreaUp[i] * (float(y) * 0.6667);
            vec3 L = q - P;
            float d2 = max(dot(L, L), 1e-4);
            L *= inversesqrt(d2);
            float cosS = max(dot(N, L), 0.0);
            float cosL = dot(-L, ln);
            cosL = AreaTwoSided[i] != 0 ? abs(cosL) : max(cosL, 0.0);
            sum += cosS * cosL / d2;
        }
    }
    float area = 4.0 * length(AreaRight[i]) * length(AreaUp[i]);
    return clamp(sum / 9.0 * area / 3.14159265, 0.0, 1.0);
}

// areaSpecPoint is the point on the rectangle nearest the mirror ray from P.
vec3 areaSpecPoint(int i, vec3 P, vec3 R) {
    vec3 ln = normalize(cross(AreaRight[i], AreaUp[i]));
    float denom = dot(R, ln);
    vec3 hit = AreaPos[i];
    if (abs(denom) > 1e-4) {
        float t = dot(AreaPos[i] - P, ln) / denom;
        hit = P + R * max(t, 0.0);
    }
    vec3 local = hit - AreaPos[i];
    float hw = length(AreaRight[i]);
    float hh = length(AreaUp[i]);
    vec3 ru = AreaRight[i] / max(hw, 1e-5);
    vec3 uu = AreaUp[i] / max(hh, 1e-5);
    return AreaPos[i] + ru * clamp(dot(local, ru), -hw, hw) + uu * clamp(dot(local, uu), -hh, hh);
}

// areaFacing is how much of the rectangle's light leaves toward P.
float areaFacing(int i, vec3 P) {
    vec3 ln = normalize(cross(AreaRight[i], AreaUp[i]));
    float c = dot(normalize(P - AreaPos[i]), ln);
    return AreaTwoSided[i] != 0 ? 1.0 : smoothstep(-0.05, 0.1, c);
}
`

func (w *World) areaLightOf(id int, where string) (*areaLight, error) {
	al := w.areaLights[id]
	if al == nil || w.ents[id] != al.e {
		return nil, fmt.Errorf("%s: %d is not an area light", where, id)
	}
	return al, nil
}

func (w *World) createAreaLight(width, height float32, parent int) int {
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = width
	}
	node := core.NewNode()
	e := &Entity{node: node}
	id := w.addEntity(e, parent)
	al := &areaLight{e: e, width: width, height: height, color: math32.Color{R: 1, G: 1, B: 1}, intensity: 4}
	// The panel is the light you see: unlit, facing the way it emits.
	al.panelMat = w.newPBR()
	al.panelMat.shading = shadeUnlit
	al.panelMat.recvShadow = false
	al.panelMat.GetMaterial().SetSide(material.SideDouble)
	al.panel = graphic.NewMesh(geometry.NewPlane(1, 1), al.panelMat)
	node.Add(al.panel)
	al.syncPanel()
	if w.areaLights == nil {
		w.areaLights = map[int]*areaLight{}
	}
	w.areaLights[id] = al
	return id
}

func (al *areaLight) syncPanel() {
	if al.panel == nil {
		return
	}
	al.panel.SetScale(al.width, al.height, 1)
	// Above 1 on purpose: a strong light should bloom.
	k := float32(math.Min(3, float64(0.4+al.intensity*0.15)))
	al.panelMat.albedo = math32.Color4{A: 1}
	al.panelMat.emissive = math32.Color{R: al.color.R * k, G: al.color.G * k, B: al.color.B * k}
	al.panelMat.applyFactors()
}

// packAreaLights fills the frame's uniforms once per render.
func (w *World) packAreaLights() *areaUniforms {
	u := &w.areaPack
	if u.serial == w.renderSerial && w.renderSerial != 0 {
		return u
	}
	u.serial = w.renderSerial
	u.n = 0
	if len(w.areaLights) == 0 {
		return u
	}
	if w.cam != nil {
		u.cam = worldPos(w.cam.GetNode())
	}
	type cand struct {
		al *areaLight
		p  math32.Vector3
		d  float32
	}
	var list []cand
	for id, al := range w.areaLights {
		if w.ents[id] != al.e {
			delete(w.areaLights, id)
			continue
		}
		n := al.e.node.GetNode()
		if !n.Visible() || al.intensity <= 0 {
			continue
		}
		p := worldPos(n)
		list = append(list, cand{al, p, p.DistanceTo(&u.cam)})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].d < list[j].d })
	for _, c := range list {
		if u.n == maxAreaLights {
			break
		}
		q := worldQuat(c.al.e.node.GetNode())
		var s math32.Vector3
		c.al.e.node.GetNode().WorldScale(&s)
		// -X with +Y puts the emitting normal, cross(right, up), on -Z.
		right := math32.Vector3{X: -1}
		up := math32.Vector3{Y: 1}
		right.ApplyQuaternion(&q)
		up.ApplyQuaternion(&q)
		right.MultiplyScalar(c.al.width * 0.5 * s.X)
		up.MultiplyScalar(c.al.height * 0.5 * s.Y)
		i := u.n
		u.pos[i], u.right[i], u.up[i] = c.p, right, up
		k := c.al.intensity
		u.color[i] = math32.Color{R: c.al.color.R * k, G: c.al.color.G * k, B: c.al.color.B * k}
		u.rng[i] = c.al.rng
		u.two[i] = int32(b01(c.al.twoSided))
		u.n++
	}
	return u
}

func (w *World) bindAreaLights(gs *gls.GLS) {
	u := w.packAreaLights()
	setUni1i(gs, "AreaCount", u.n)
	if u.n == 0 {
		return
	}
	setUni3f(gs, "AreaCam", u.cam.X, u.cam.Y, u.cam.Z)
	setVec3Array(gs, "AreaPos", u.pos[:])
	setVec3Array(gs, "AreaRight", u.right[:])
	setVec3Array(gs, "AreaUp", u.up[:])
	cols := make([]math32.Vector3, maxAreaLights)
	for i, c := range u.color {
		cols[i] = math32.Vector3{X: c.R, Y: c.G, Z: c.B}
	}
	setVec3Array(gs, "AreaColor", cols)
	uni := gls.Uniform{}
	uni.Init("AreaRange")
	if loc := uni.Location(gs); loc >= 0 {
		gs.Uniform1fv(loc, maxAreaLights, &u.rng[0])
	}
	two := gls.Uniform{}
	two.Init("AreaTwoSided")
	if loc := two.Location(gs); loc >= 0 {
		gl.Uniform1iv(loc, maxAreaLights, &u.two[0])
	}
}

func setVec3Array(gs *gls.GLS, name string, v []math32.Vector3) {
	u := gls.Uniform{}
	u.Init(name)
	if loc := u.Location(gs); loc >= 0 && len(v) > 0 {
		gs.Uniform3fv(loc, int32(len(v)), &v[0].X)
	}
}

func (w *World) areaLightCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		// CreateAreaLight([width, height [, parent]])
		"createarealight": need(func(a []value.Value) (value.Value, error) {
			id := w.createAreaLight(float32(argN(a, 0, 1)), float32(argN(a, 1, 0)), argI(a, 2, 0))
			return value.Num(float64(id)), nil
		}),
		// SetAreaLight light, r, g, b [, intensity]
		"setarealight": n(func(a []value.Value) (value.Value, error) {
			al, err := w.areaLightOf(argI(a, 0, 0), "SetAreaLight")
			if err != nil {
				return value.Value{}, err
			}
			al.color = math32.Color{R: colorArg(argN(a, 1, 255)), G: colorArg(argN(a, 2, 255)), B: colorArg(argN(a, 3, 255))}
			if len(a) >= 5 {
				al.intensity = float32(math.Max(0, argN(a, 4, 4)))
			}
			al.syncPanel()
			return z()
		}),
		"setarealightintensity": n(func(a []value.Value) (value.Value, error) {
			al, err := w.areaLightOf(argI(a, 0, 0), "SetAreaLightIntensity")
			if err != nil {
				return value.Value{}, err
			}
			al.intensity = float32(math.Max(0, argN(a, 1, 4)))
			al.syncPanel()
			return z()
		}),
		"setarealightsize": n(func(a []value.Value) (value.Value, error) {
			al, err := w.areaLightOf(argI(a, 0, 0), "SetAreaLightSize")
			if err != nil {
				return value.Value{}, err
			}
			al.width = float32(math.Max(0.01, argN(a, 1, 1)))
			al.height = float32(math.Max(0.01, argN(a, 2, float64(al.width))))
			al.syncPanel()
			return z()
		}),
		"setarealightrange": n(func(a []value.Value) (value.Value, error) {
			al, err := w.areaLightOf(argI(a, 0, 0), "SetAreaLightRange")
			if err != nil {
				return value.Value{}, err
			}
			al.rng = float32(math.Max(0, argN(a, 1, 0)))
			return z()
		}),
		"setarealighttwosided": n(func(a []value.Value) (value.Value, error) {
			al, err := w.areaLightOf(argI(a, 0, 0), "SetAreaLightTwoSided")
			if err != nil {
				return value.Value{}, err
			}
			al.twoSided = argI(a, 1, 1) != 0
			return z()
		}),
		// ShowAreaLightPanel light, on: the glowing rectangle itself.
		"showarealightpanel": n(func(a []value.Value) (value.Value, error) {
			al, err := w.areaLightOf(argI(a, 0, 0), "ShowAreaLightPanel")
			if err != nil {
				return value.Value{}, err
			}
			al.panel.SetVisible(argI(a, 1, 1) != 0)
			return z()
		}),
	}
}
