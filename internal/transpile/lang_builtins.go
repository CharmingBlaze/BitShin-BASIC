package transpile

import (
	"fmt"
	"strings"
)

// langBuiltins are interpreter functions that are not runtime commands.
// The value is the generated helper name. A user function of the same name wins.
var langBuiltins = map[string]string{
	"float": "float", "sgn": "sgn", "pow": "pow",
	"seedrnd": "seedrnd", "rndseed": "rndseed",
	"clamp": "clamp", "lerp": "lerp", "invlerp": "invlerp",
	"smoothstep": "smoothstep", "easein": "easein", "easeout": "easeout",
	"approach": "approach", "wrapangle": "wrapangle",
	"angledelta": "angledelta", "approachangle": "approachangle",
	"dist": "distance2d", "distance2d": "distance2d",
	"distance3d": "distance3d", "pointdistance": "pointdistance",
	"length2d": "length2d", "length3d": "length3d",
	"normx": "normx", "normy": "normy",
	"normx3": "normx3", "normy3": "normy3", "normz3": "normz3",
	"dirx": "dirx", "diry": "diry", "dirz": "dirz",
	"movepointx": "movepointx", "movepointy": "movepointy", "movepointz": "movepointz",
	"pointyaw": "pointyaw", "pointpitch": "pointpitch",
	"dot2d": "dot2d", "dot3d": "dot3d",
	"crossx": "crossx", "crossy": "crossy", "crossz": "crossz",
	"reflectx": "reflectx", "reflecty": "reflecty",
	"bouncex": "reflectx", "bouncey": "reflecty",
	"rotatedx": "rotatedx", "rotatedy": "rotatedy",
	"accelerate": "accelerate", "turntoward": "turntoward",
	"material": "material", "callback": "callback",
	"createlist": "createlist", "listadd": "listadd", "listget": "listget",
	"listset": "listset", "listcount": "listcount", "listremove": "listremove",
	"land": "land",
}

var langWorldBuiltins = map[string]string{
	"movewish": "movewish",
}

func emitLangBuiltin(name, worldVar string, args []string) (string, bool) {
	if fn, ok := langWorldBuiltins[name]; ok {
		if len(args) == 0 {
			return fmt.Sprintf("_bb_%s(%s)", fn, worldVar), true
		}
		return fmt.Sprintf("_bb_%s(%s, %s)", fn, worldVar, strings.Join(args, ", ")), true
	}
	fn, ok := langBuiltins[name]
	if !ok {
		return "", false
	}
	if len(args) == 0 {
		return fmt.Sprintf("_bb_%s()", fn), true
	}
	return fmt.Sprintf("_bb_%s(%s)", fn, strings.Join(args, ", ")), true
}

func emitLangHelpers(w *writer) {
	w.buf.WriteString(langHelperSrc)
}

const langHelperSrc = `
var _bb_rng_seed int64
var _bb_lists = map[int][]value.Value{}
var _bb_list_n int

func _bb_n(a []value.Value, i int) float64 {
	if i >= len(a) {
		return 0
	}
	return a[i].Number()
}

func _bb_clamp01(t float64) float64 {
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}

func _bb_wrap180(d float64) float64 {
	for d > 180 {
		d -= 360
	}
	for d < -180 {
		d += 360
	}
	return d
}

func _bb_float(a ...value.Value) value.Value { return value.Num(_bb_n(a, 0)) }

func _bb_sgn(a ...value.Value) value.Value {
	x := _bb_n(a, 0)
	if x < 0 {
		return value.Num(-1)
	}
	if x > 0 {
		return value.Num(1)
	}
	return value.Num(0)
}

func _bb_pow(a ...value.Value) value.Value {
	return value.Num(math.Pow(_bb_n(a, 0), _bb_n(a, 1)))
}

func _bb_seedrnd(a ...value.Value) value.Value {
	s := uint64(int64(_bb_n(a, 0)))
	_bb_rng_seed = int64(s)
	_bb_rng = rand.New(rand.NewPCG(s, s^0xA0761D6478BD642F))
	return value.Num(0)
}

func _bb_rndseed(a ...value.Value) value.Value { return value.Num(float64(_bb_rng_seed)) }

func _bb_clamp(a ...value.Value) value.Value {
	x, lo, hi := _bb_n(a, 0), _bb_n(a, 1), _bb_n(a, 2)
	if lo > hi {
		lo, hi = hi, lo
	}
	return value.Num(math.Min(hi, math.Max(lo, x)))
}

func _bb_lerp(a ...value.Value) value.Value {
	return value.Num(_bb_n(a, 0) + (_bb_n(a, 1)-_bb_n(a, 0))*_bb_n(a, 2))
}

func _bb_invlerp(a ...value.Value) value.Value {
	a0, b0, x := _bb_n(a, 0), _bb_n(a, 1), _bb_n(a, 2)
	if a0 == b0 {
		return value.Num(0)
	}
	return value.Num((x - a0) / (b0 - a0))
}

func _bb_smoothstep(a ...value.Value) value.Value {
	var t float64
	if len(a) >= 3 {
		t = _bb_clamp01(_bb_invlerp(a[0], a[1], a[2]).Number())
	} else {
		t = _bb_clamp01(_bb_n(a, 0))
	}
	return value.Num(t * t * (3 - 2*t))
}

func _bb_easein(a ...value.Value) value.Value {
	if len(a) >= 3 {
		t := _bb_clamp01(_bb_n(a, 2))
		return value.Num(_bb_n(a, 0) + (_bb_n(a, 1)-_bb_n(a, 0))*(t*t))
	}
	t := _bb_clamp01(_bb_n(a, 0))
	return value.Num(t * t)
}

func _bb_easeout(a ...value.Value) value.Value {
	if len(a) >= 3 {
		t := _bb_clamp01(_bb_n(a, 2))
		e := 1 - (1-t)*(1-t)
		return value.Num(_bb_n(a, 0) + (_bb_n(a, 1)-_bb_n(a, 0))*e)
	}
	t := _bb_clamp01(_bb_n(a, 0))
	return value.Num(1 - (1-t)*(1-t))
}

func _bb_approach(a ...value.Value) value.Value {
	cur, tgt, step := _bb_n(a, 0), _bb_n(a, 1), math.Abs(_bb_n(a, 2))
	d := tgt - cur
	if math.Abs(d) <= step {
		return value.Num(tgt)
	}
	return value.Num(cur + math.Copysign(step, d))
}

func _bb_wrapangle(a ...value.Value) value.Value { return value.Num(_bb_wrap180(_bb_n(a, 0))) }

func _bb_angledelta(a ...value.Value) value.Value {
	return value.Num(_bb_wrap180(_bb_n(a, 1) - _bb_n(a, 0)))
}

func _bb_approachangle(a ...value.Value) value.Value {
	cur, tgt, step := _bb_n(a, 0), _bb_n(a, 1), math.Abs(_bb_n(a, 2))
	d := _bb_wrap180(tgt - cur)
	if math.Abs(d) <= step {
		return value.Num(_bb_wrap180(tgt))
	}
	return value.Num(_bb_wrap180(cur + math.Copysign(step, d)))
}

func _bb_distance2d(a ...value.Value) value.Value {
	return value.Num(math.Hypot(_bb_n(a, 2)-_bb_n(a, 0), _bb_n(a, 3)-_bb_n(a, 1)))
}

func _bb_distance3d(a ...value.Value) value.Value {
	dx, dy, dz := _bb_n(a, 3)-_bb_n(a, 0), _bb_n(a, 4)-_bb_n(a, 1), _bb_n(a, 5)-_bb_n(a, 2)
	return value.Num(math.Sqrt(dx*dx + dy*dy + dz*dz))
}

func _bb_pointdistance(a ...value.Value) value.Value {
	if len(a) >= 6 {
		return _bb_distance3d(a...)
	}
	return _bb_distance2d(a...)
}

func _bb_length2d(a ...value.Value) value.Value {
	return value.Num(math.Hypot(_bb_n(a, 0), _bb_n(a, 1)))
}

func _bb_length3d(a ...value.Value) value.Value {
	x, y, z := _bb_n(a, 0), _bb_n(a, 1), _bb_n(a, 2)
	return value.Num(math.Sqrt(x*x + y*y + z*z))
}

func _bb_norm2(a []value.Value, i int) float64 {
	x, y := _bb_n(a, 0), _bb_n(a, 1)
	l := math.Hypot(x, y)
	if l == 0 {
		return 0
	}
	if i == 0 {
		return x / l
	}
	return y / l
}

func _bb_normx(a ...value.Value) value.Value  { return value.Num(_bb_norm2(a, 0)) }
func _bb_normy(a ...value.Value) value.Value  { return value.Num(_bb_norm2(a, 1)) }

func _bb_norm3(a []value.Value, i int) float64 {
	x, y, z := _bb_n(a, 0), _bb_n(a, 1), _bb_n(a, 2)
	l := math.Sqrt(x*x + y*y + z*z)
	if l == 0 {
		return 0
	}
	switch i {
	case 0:
		return x / l
	case 1:
		return y / l
	default:
		return z / l
	}
}

func _bb_normx3(a ...value.Value) value.Value { return value.Num(_bb_norm3(a, 0)) }
func _bb_normy3(a ...value.Value) value.Value { return value.Num(_bb_norm3(a, 1)) }
func _bb_normz3(a ...value.Value) value.Value { return value.Num(_bb_norm3(a, 2)) }

func _bb_dirx(a ...value.Value) value.Value { return value.Num(math.Sin(_bb_deg(_bb_n(a, 0)))) }
func _bb_diry(a ...value.Value) value.Value { return value.Num(math.Sin(_bb_deg(_bb_n(a, 0)))) }
func _bb_dirz(a ...value.Value) value.Value { return value.Num(math.Cos(_bb_deg(_bb_n(a, 0)))) }

func _bb_movepointx(a ...value.Value) value.Value {
	return value.Num(_bb_n(a, 0) + math.Sin(_bb_deg(_bb_n(a, 1)))*_bb_n(a, 2))
}
func _bb_movepointy(a ...value.Value) value.Value {
	return value.Num(_bb_n(a, 0) + math.Sin(_bb_deg(_bb_n(a, 1)))*_bb_n(a, 2))
}
func _bb_movepointz(a ...value.Value) value.Value {
	return value.Num(_bb_n(a, 0) + math.Cos(_bb_deg(_bb_n(a, 1)))*_bb_n(a, 2))
}

func _bb_pointyaw(a ...value.Value) value.Value {
	var dx, dz float64
	if len(a) >= 6 {
		dx, dz = _bb_n(a, 3)-_bb_n(a, 0), _bb_n(a, 5)-_bb_n(a, 2)
	} else {
		dx, dz = _bb_n(a, 2)-_bb_n(a, 0), _bb_n(a, 3)-_bb_n(a, 1)
	}
	return value.Num(_bb_rad(math.Atan2(dx, dz)))
}

func _bb_pointpitch(a ...value.Value) value.Value {
	var dx, dy, dz float64
	if len(a) >= 6 {
		dx, dy, dz = _bb_n(a, 3)-_bb_n(a, 0), _bb_n(a, 4)-_bb_n(a, 1), _bb_n(a, 5)-_bb_n(a, 2)
	} else {
		dx, dy, dz = _bb_n(a, 0), _bb_n(a, 1), _bb_n(a, 2)
	}
	return value.Num(_bb_rad(math.Atan2(dy, math.Sqrt(dx*dx+dz*dz))))
}

func _bb_dot2d(a ...value.Value) value.Value {
	return value.Num(_bb_n(a, 0)*_bb_n(a, 2) + _bb_n(a, 1)*_bb_n(a, 3))
}
func _bb_dot3d(a ...value.Value) value.Value {
	return value.Num(_bb_n(a, 0)*_bb_n(a, 3) + _bb_n(a, 1)*_bb_n(a, 4) + _bb_n(a, 2)*_bb_n(a, 5))
}
func _bb_crossx(a ...value.Value) value.Value {
	return value.Num(_bb_n(a, 1)*_bb_n(a, 5) - _bb_n(a, 2)*_bb_n(a, 4))
}
func _bb_crossy(a ...value.Value) value.Value {
	return value.Num(_bb_n(a, 2)*_bb_n(a, 3) - _bb_n(a, 0)*_bb_n(a, 5))
}
func _bb_crossz(a ...value.Value) value.Value {
	return value.Num(_bb_n(a, 0)*_bb_n(a, 4) - _bb_n(a, 1)*_bb_n(a, 3))
}

func _bb_reflect2(a []value.Value) (float64, float64) {
	vx, vy, nx, ny := _bb_n(a, 0), _bb_n(a, 1), _bb_n(a, 2), _bb_n(a, 3)
	nl := math.Hypot(nx, ny)
	if nl == 0 {
		return vx, vy
	}
	nx, ny = nx/nl, ny/nl
	dot := vx*nx + vy*ny
	return vx - 2*dot*nx, vy - 2*dot*ny
}

func _bb_reflectx(a ...value.Value) value.Value {
	rx, _ := _bb_reflect2(a)
	return value.Num(rx)
}
func _bb_reflecty(a ...value.Value) value.Value {
	_, ry := _bb_reflect2(a)
	return value.Num(ry)
}

func _bb_rotatedx(a ...value.Value) value.Value {
	x, y, c, s := _bb_n(a, 0), _bb_n(a, 1), math.Cos(_bb_deg(_bb_n(a, 2))), math.Sin(_bb_deg(_bb_n(a, 2)))
	return value.Num(x*c - y*s)
}
func _bb_rotatedy(a ...value.Value) value.Value {
	x, y, c, s := _bb_n(a, 0), _bb_n(a, 1), math.Cos(_bb_deg(_bb_n(a, 2))), math.Sin(_bb_deg(_bb_n(a, 2)))
	return value.Num(x*s + y*c)
}

func _bb_movewish(w *runtime.World, a ...value.Value) value.Value {
	yaw := _bb_n(a, 0)
	wx, wz := 0.0, 0.0
	add := func(dir float64) {
		wx += math.Sin(dir * math.Pi / 180)
		wz += math.Cos(dir * math.Pi / 180)
	}
	held := func(name string) bool {
		if w == nil {
			return false
		}
		code, ok := syntax.KeyConstants[name]
		if !ok {
			return false
		}
		v, err := w.Call("keydown", []value.Value{value.Num(code)})
		return err == nil && v.IsTrue()
	}
	if held("key_w") {
		add(yaw)
	}
	if held("key_s") {
		add(yaw + 180)
	}
	if held("key_a") {
		add(yaw - 90)
	}
	if held("key_d") {
		add(yaw + 90)
	}
	if l := math.Hypot(wx, wz); l > 0.001 {
		wx /= l
		wz /= l
	}
	return value.Vec([]value.Value{value.Num(wx), value.Num(wz)})
}

func _bb_accelerate(a ...value.Value) value.Value {
	vx, vz := _bb_n(a, 0), _bb_n(a, 1)
	wx, wz := _bb_n(a, 2), _bb_n(a, 3)
	acc, fric, maxSpd := _bb_n(a, 4), _bb_n(a, 5), _bb_n(a, 6)
	grounded, dt := _bb_n(a, 7), _bb_n(a, 8)
	if spd := math.Hypot(vx, vz); spd > 0.001 {
		nspd := spd - fric*dt
		if nspd < 0 {
			nspd = 0
		}
		vx *= nspd / spd
		vz *= nspd / spd
	}
	if math.Hypot(wx, wz) > 0.001 {
		vx += wx * acc * dt
		vz += wz * acc * dt
		hsp := math.Hypot(vx, vz)
		cap := maxSpd
		if grounded == 0 && hsp > maxSpd {
			cap = hsp
		}
		if grounded != 0 && hsp > maxSpd {
			vx *= maxSpd / hsp
			vz *= maxSpd / hsp
		} else if grounded == 0 && hsp > cap {
			vx *= cap / hsp
			vz *= cap / hsp
		}
	}
	return value.Vec([]value.Value{value.Num(vx), value.Num(vz)})
}

func _bb_turntoward(a ...value.Value) value.Value {
	yaw, vx, vz, rate, minSpd := _bb_n(a, 0), _bb_n(a, 1), _bb_n(a, 2), _bb_n(a, 3), _bb_n(a, 4)
	if minSpd == 0 {
		minSpd = 0.45
	}
	if math.Hypot(vx, vz) <= minSpd {
		return value.Num(yaw)
	}
	want := math.Atan2(vx, vz) * 180 / math.Pi
	diff := _bb_wrap180(want - yaw)
	if math.Abs(diff) <= rate {
		return value.Num(want)
	}
	if diff > 0 {
		return value.Num(yaw + rate)
	}
	return value.Num(yaw - rate)
}

func _bb_land_elems(v value.Value) []value.Value {
	switch {
	case v.Kind == value.KindNum && v.TypeName == "list":
		return append([]value.Value(nil), _bb_lists[v.Int()]...)
	case v.Kind == value.KindVec:
		return append([]value.Value(nil), v.Elems...)
	case v.Kind == value.KindArray:
		if v.TypeName == "" {
			return append([]value.Value(nil), v.Elems...)
		}
		out := make([]value.Value, 0, len(v.Elems))
		for _, el := range v.Elems {
			if el.Kind == value.KindNum && el.Num == 0 && el.TypeName == "" {
				continue
			}
			out = append(out, el)
		}
		return out
	default:
		return nil
	}
}

func _bb_fld(v value.Value, name string) float64 {
	f, ok := v.Field(name)
	if !ok {
		return 0
	}
	return f.Number()
}

func _bb_land(a ...value.Value) value.Value {
	px, py, pz, vy := _bb_n(a, 0), _bb_n(a, 1), _bb_n(a, 2), _bb_n(a, 3)
	none := value.Vec([]value.Value{value.Num(py), value.Num(vy), value.Num(0)})
	var pads value.Value
	if len(a) > 4 {
		pads = a[4]
	}
	best := -999.0
	hit := false
	for _, el := range _bb_land_elems(pads) {
		if el.Kind != value.KindStruct && el.Kind != value.KindMap {
			continue
		}
		x, y, z := _bb_fld(el, "x"), _bb_fld(el, "y"), _bb_fld(el, "z")
		w := _bb_fld(el, "w")
		if _, ok := el.Field("w"); !ok {
			w = _bb_fld(el, "hx")
		}
		d := _bb_fld(el, "d")
		if _, ok := el.Field("d"); !ok {
			d = _bb_fld(el, "hz")
		}
		if math.Abs(px-x) < w && math.Abs(pz-z) < d && py <= y+0.45 && py >= y-0.85 && vy <= 0.35 {
			if !hit || y > best {
				best = y
				hit = true
			}
		}
	}
	if !hit {
		return none
	}
	return value.Vec([]value.Value{value.Num(best), value.Num(0), value.Num(1)})
}

func _bb_material(a ...value.Value) value.Value {
	r, g, b := 255.0, 255.0, 255.0
	tex, shine := 0.0, 0.0
	sr, sg, sb := 0.0, 0.0, 0.0
	spec := 0.0
	if len(a) > 0 && (a[0].Kind == value.KindVec || a[0].Number() > 255) {
		if x, y, z, ok := a[0].XYZ(); ok {
			r, g, b = x, y, z
		} else {
			r, g, b = syntax.PackedRGB(a[0].Int())
		}
		tex, shine = _bb_n(a, 1), _bb_n(a, 2)
		if len(a) >= 6 {
			sr, sg, sb = _bb_n(a, 3), _bb_n(a, 4), _bb_n(a, 5)
			spec = 1
		}
	} else {
		if len(a) > 0 {
			r = _bb_n(a, 0)
		}
		if len(a) > 1 {
			g = _bb_n(a, 1)
		}
		if len(a) > 2 {
			b = _bb_n(a, 2)
		}
		tex, shine = _bb_n(a, 3), _bb_n(a, 4)
		if len(a) >= 8 {
			sr, sg, sb = _bb_n(a, 5), _bb_n(a, 6), _bb_n(a, 7)
			spec = 1
		}
	}
	m := value.StructOf("material", []string{"r", "g", "b", "tex", "shine", "sr", "sg", "sb", "spec"})
	m.SetField("r", value.Num(r))
	m.SetField("g", value.Num(g))
	m.SetField("b", value.Num(b))
	m.SetField("tex", value.Num(tex))
	m.SetField("shine", value.Num(shine))
	m.SetField("sr", value.Num(sr))
	m.SetField("sg", value.Num(sg))
	m.SetField("sb", value.Num(sb))
	m.SetField("spec", value.Num(spec))
	return m
}

func _bb_callback(a ...value.Value) value.Value {
	s := ""
	if len(a) > 0 {
		s = a[0].String()
	}
	return value.Func(s)
}

func _bb_createlist(a ...value.Value) value.Value {
	id := _bb_list_n
	_bb_list_n++
	_bb_lists[id] = []value.Value{}
	v := value.Num(float64(id))
	v.TypeName = "list"
	return v
}

func _bb_listadd(a ...value.Value) value.Value {
	id := int(_bb_n(a, 0))
	var item value.Value
	if len(a) > 1 {
		item = a[1]
	}
	_bb_lists[id] = append(_bb_lists[id], item)
	return value.Num(float64(len(_bb_lists[id])))
}

func _bb_listget(a ...value.Value) value.Value {
	lst := _bb_lists[int(_bb_n(a, 0))]
	i := int(_bb_n(a, 1))
	if i < 0 || i >= len(lst) {
		return value.Num(0)
	}
	return lst[i]
}

func _bb_listset(a ...value.Value) value.Value {
	id := int(_bb_n(a, 0))
	lst := _bb_lists[id]
	i := int(_bb_n(a, 1))
	if i < 0 || i >= len(lst) {
		return value.Num(0)
	}
	var item value.Value
	if len(a) > 2 {
		item = a[2]
	}
	lst[i] = item
	_bb_lists[id] = lst
	return value.Num(1)
}

func _bb_listcount(a ...value.Value) value.Value {
	return value.Num(float64(len(_bb_lists[int(_bb_n(a, 0))])))
}

func _bb_listremove(a ...value.Value) value.Value {
	id := int(_bb_n(a, 0))
	lst := _bb_lists[id]
	i := int(_bb_n(a, 1))
	if i < 0 || i >= len(lst) {
		return value.Num(0)
	}
	_bb_lists[id] = append(lst[:i], lst[i+1:]...)
	return value.Num(1)
}
`
