package interp

import (
	"math"

	"bitshinbasic/internal/syntax"
	"bitshinbasic/internal/value"
)

// All trig and angle helpers use degrees.
func (in *Interp) installMath(n func(string, func([]value.Value) value.Value)) {
	n("clamp", func(a []value.Value) value.Value {
		x, lo, hi := num(a, 0), num(a, 1), num(a, 2)
		if lo > hi {
			lo, hi = hi, lo
		}
		return value.Num(math.Min(hi, math.Max(lo, x)))
	})
	n("lerp", func(a []value.Value) value.Value {
		return value.Num(num(a, 0) + (num(a, 1)-num(a, 0))*num(a, 2))
	})
	n("invlerp", func(a []value.Value) value.Value {
		a0, b0, x := num(a, 0), num(a, 1), num(a, 2)
		if a0 == b0 {
			return value.Num(0)
		}
		return value.Num((x - a0) / (b0 - a0))
	})
	n("smoothstep", func(a []value.Value) value.Value {
		var t float64
		if len(a) >= 3 {
			t = clamp01(invLerp(num(a, 0), num(a, 1), num(a, 2)))
		} else {
			t = clamp01(num(a, 0))
		}
		return value.Num(t * t * (3 - 2*t))
	})
	n("easein", func(a []value.Value) value.Value {
		if len(a) >= 3 {
			t := clamp01(num(a, 2))
			return value.Num(num(a, 0) + (num(a, 1)-num(a, 0))*(t*t))
		}
		t := clamp01(num(a, 0))
		return value.Num(t * t)
	})
	n("easeout", func(a []value.Value) value.Value {
		if len(a) >= 3 {
			t := clamp01(num(a, 2))
			e := 1 - (1-t)*(1-t)
			return value.Num(num(a, 0) + (num(a, 1)-num(a, 0))*e)
		}
		t := clamp01(num(a, 0))
		return value.Num(1 - (1-t)*(1-t))
	})
	n("approach", func(a []value.Value) value.Value {
		return value.Num(approach(num(a, 0), num(a, 1), math.Abs(num(a, 2))))
	})
	n("wrapangle", func(a []value.Value) value.Value { return value.Num(wrap180(num(a, 0))) })
	n("angledelta", func(a []value.Value) value.Value { return value.Num(wrap180(num(a, 1) - num(a, 0))) })
	n("approachangle", func(a []value.Value) value.Value {
		cur, tgt, step := num(a, 0), num(a, 1), math.Abs(num(a, 2))
		d := wrap180(tgt - cur)
		if math.Abs(d) <= step {
			return value.Num(wrap180(tgt))
		}
		return value.Num(wrap180(cur + math.Copysign(step, d)))
	})
	n("pow", func(a []value.Value) value.Value { return value.Num(math.Pow(num(a, 0), num(a, 1))) })

	n("dist", func(a []value.Value) value.Value { return value.Num(dist2(a)) })
	n("distance2d", func(a []value.Value) value.Value { return value.Num(dist2(a)) })
	n("pointdistance", func(a []value.Value) value.Value {
		if len(a) >= 6 {
			return value.Num(dist3(a))
		}
		return value.Num(dist2(a))
	})
	n("distance3d", func(a []value.Value) value.Value { return value.Num(dist3(a)) })
	n("length2d", func(a []value.Value) value.Value {
		x, y := num(a, 0), num(a, 1)
		return value.Num(math.Hypot(x, y))
	})
	n("length3d", func(a []value.Value) value.Value {
		x, y, z := num(a, 0), num(a, 1), num(a, 2)
		return value.Num(math.Sqrt(x*x + y*y + z*z))
	})
	n("normx", func(a []value.Value) value.Value { return value.Num(norm2(a, 0)) })
	n("normy", func(a []value.Value) value.Value { return value.Num(norm2(a, 1)) })
	n("normx3", func(a []value.Value) value.Value { return value.Num(norm3(a, 0)) })
	n("normy3", func(a []value.Value) value.Value { return value.Num(norm3(a, 1)) })
	n("normz3", func(a []value.Value) value.Value { return value.Num(norm3(a, 2)) })

	n("dirx", func(a []value.Value) value.Value { return value.Num(math.Sin(deg(a, 0))) })
	n("dirz", func(a []value.Value) value.Value { return value.Num(math.Cos(deg(a, 0))) })
	n("diry", func(a []value.Value) value.Value { return value.Num(math.Sin(deg(a, 0))) })

	n("movepointx", func(a []value.Value) value.Value {
		return value.Num(num(a, 0) + math.Sin(deg(a, 1))*num(a, 2))
	})
	n("movepointz", func(a []value.Value) value.Value {
		return value.Num(num(a, 0) + math.Cos(deg(a, 1))*num(a, 2))
	})
	n("movepointy", func(a []value.Value) value.Value {
		return value.Num(num(a, 0) + math.Sin(deg(a, 1))*num(a, 2))
	})

	n("pointyaw", func(a []value.Value) value.Value {
		var dx, dz float64
		if len(a) >= 6 {
			dx, dz = num(a, 3)-num(a, 0), num(a, 5)-num(a, 2)
		} else {
			dx, dz = num(a, 2)-num(a, 0), num(a, 3)-num(a, 1)
		}
		return value.Num(radToDeg(math.Atan2(dx, dz)))
	})
	n("pointpitch", func(a []value.Value) value.Value {
		var dx, dy, dz float64
		if len(a) >= 6 {
			dx, dy, dz = num(a, 3)-num(a, 0), num(a, 4)-num(a, 1), num(a, 5)-num(a, 2)
		} else {
			dx, dy, dz = num(a, 0), num(a, 1), num(a, 2)
		}
		return value.Num(radToDeg(math.Atan2(dy, math.Sqrt(dx*dx+dz*dz))))
	})

	n("dot2d", func(a []value.Value) value.Value {
		return value.Num(num(a, 0)*num(a, 2) + num(a, 1)*num(a, 3))
	})
	n("dot3d", func(a []value.Value) value.Value {
		return value.Num(num(a, 0)*num(a, 3) + num(a, 1)*num(a, 4) + num(a, 2)*num(a, 5))
	})
	n("crossx", func(a []value.Value) value.Value {
		return value.Num(num(a, 1)*num(a, 5) - num(a, 2)*num(a, 4))
	})
	n("crossy", func(a []value.Value) value.Value {
		return value.Num(num(a, 2)*num(a, 3) - num(a, 0)*num(a, 5))
	})
	n("crossz", func(a []value.Value) value.Value {
		return value.Num(num(a, 0)*num(a, 4) - num(a, 1)*num(a, 3))
	})

	n("reflectx", func(a []value.Value) value.Value { rx, _ := reflect2(a); return value.Num(rx) })
	n("reflecty", func(a []value.Value) value.Value { _, ry := reflect2(a); return value.Num(ry) })
	n("bouncex", func(a []value.Value) value.Value { rx, _ := reflect2(a); return value.Num(rx) })
	n("bouncey", func(a []value.Value) value.Value { _, ry := reflect2(a); return value.Num(ry) })

	n("rotatedx", func(a []value.Value) value.Value {
		x, y, c, s := num(a, 0), num(a, 1), math.Cos(deg(a, 2)), math.Sin(deg(a, 2))
		return value.Num(x*c - y*s)
	})
	n("rotatedy", func(a []value.Value) value.Value {
		x, y, c, s := num(a, 0), num(a, 1), math.Cos(deg(a, 2)), math.Sin(deg(a, 2))
		return value.Num(x*s + y*c)
	})

	n("movewish", func(a []value.Value) value.Value { return in.moveWish(num(a, 0)) })
	n("accelerate", func(a []value.Value) value.Value {
		vx, vz := accelerate(
			num(a, 0), num(a, 1), num(a, 2), num(a, 3),
			num(a, 4), num(a, 5), num(a, 6), num(a, 7), num(a, 8),
		)
		return value.Vec([]value.Value{value.Num(vx), value.Num(vz)})
	})
	n("turntoward", func(a []value.Value) value.Value {
		return value.Num(turnToward(num(a, 0), num(a, 1), num(a, 2), num(a, 3), num(a, 4)))
	})
	n("land", func(a []value.Value) value.Value {
		pads := value.Value{}
		if len(a) > 4 {
			pads = a[4]
		}
		return in.landOn(num(a, 0), num(a, 1), num(a, 2), num(a, 3), pads)
	})
	n("material", func(a []value.Value) value.Value { return materialValue(a) })
}

func (in *Interp) keyHeld(name string) bool {
	if in.host == nil {
		return false
	}
	code, ok := syntax.KeyConstants[name]
	if !ok {
		return false
	}
	v, err := in.host.Call("keydown", []value.Value{value.Num(code)})
	if err != nil {
		return false
	}
	return v.IsTrue()
}

func (in *Interp) moveWish(yaw float64) value.Value {
	wx, wz := 0.0, 0.0
	add := func(dir float64) {
		wx += math.Sin(dir * math.Pi / 180)
		wz += math.Cos(dir * math.Pi / 180)
	}
	if in.keyHeld("key_w") {
		add(yaw)
	}
	if in.keyHeld("key_s") {
		add(yaw + 180)
	}
	if in.keyHeld("key_a") {
		add(yaw - 90)
	}
	if in.keyHeld("key_d") {
		add(yaw + 90)
	}
	if l := math.Hypot(wx, wz); l > 0.001 {
		wx /= l
		wz /= l
	}
	return value.Vec([]value.Value{value.Num(wx), value.Num(wz)})
}

func accelerate(vx, vz, wx, wz, acc, fric, maxSpd, grounded, dt float64) (float64, float64) {
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
	return vx, vz
}

func turnToward(yaw, vx, vz, rate, minSpd float64) float64 {
	if minSpd == 0 {
		minSpd = 0.45
	}
	if math.Hypot(vx, vz) <= minSpd {
		return yaw
	}
	want := math.Atan2(vx, vz) * 180 / math.Pi
	diff := wrap180(want - yaw)
	if math.Abs(diff) <= rate {
		return want
	}
	if diff > 0 {
		return yaw + rate
	}
	return yaw - rate
}

func (in *Interp) landOn(px, py, pz, vy float64, pads value.Value) value.Value {
	none := value.Vec([]value.Value{value.Num(py), value.Num(vy), value.Num(0)})
	elems, err := in.iterElems(pads)
	if err != nil {
		return none
	}
	best := -999.0
	hit := false
	for _, el := range elems {
		if el.Kind != value.KindStruct && el.Kind != value.KindMap {
			continue
		}
		x, y, z := fld(el, "x"), fld(el, "y"), fld(el, "z")
		w := fld(el, "w")
		if _, ok := el.Field("w"); !ok {
			w = fld(el, "hx")
		}
		d := fld(el, "d")
		if _, ok := el.Field("d"); !ok {
			d = fld(el, "hz")
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

func fld(v value.Value, name string) float64 {
	f, ok := v.Field(name)
	if !ok {
		return 0
	}
	return f.Number()
}

func materialValue(a []value.Value) value.Value {
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
		tex, shine = num(a, 1), num(a, 2)
		if len(a) >= 6 {
			sr, sg, sb = num(a, 3), num(a, 4), num(a, 5)
			spec = 1
		}
	} else {
		if len(a) > 0 {
			r = num(a, 0)
		}
		if len(a) > 1 {
			g = num(a, 1)
		}
		if len(a) > 2 {
			b = num(a, 2)
		}
		tex, shine = num(a, 3), num(a, 4)
		if len(a) >= 8 {
			sr, sg, sb = num(a, 5), num(a, 6), num(a, 7)
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

func clamp01(t float64) float64 {
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}

func invLerp(a, b, x float64) float64 {
	if a == b {
		return 0
	}
	return (x - a) / (b - a)
}

func approach(cur, tgt, step float64) float64 {
	d := tgt - cur
	if math.Abs(d) <= step {
		return tgt
	}
	return cur + math.Copysign(step, d)
}

func wrap180(d float64) float64 {
	for d > 180 {
		d -= 360
	}
	for d < -180 {
		d += 360
	}
	return d
}

func dist2(a []value.Value) float64 {
	return math.Hypot(num(a, 2)-num(a, 0), num(a, 3)-num(a, 1))
}

func dist3(a []value.Value) float64 {
	dx, dy, dz := num(a, 3)-num(a, 0), num(a, 4)-num(a, 1), num(a, 5)-num(a, 2)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func norm2(a []value.Value, i int) float64 {
	x, y := num(a, 0), num(a, 1)
	l := math.Hypot(x, y)
	if l == 0 {
		return 0
	}
	if i == 0 {
		return x / l
	}
	return y / l
}

func norm3(a []value.Value, i int) float64 {
	x, y, z := num(a, 0), num(a, 1), num(a, 2)
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

func reflect2(a []value.Value) (float64, float64) {
	vx, vy, nx, ny := num(a, 0), num(a, 1), num(a, 2), num(a, 3)
	nl := math.Hypot(nx, ny)
	if nl == 0 {
		return vx, vy
	}
	nx, ny = nx/nl, ny/nl
	dot := vx*nx + vy*ny
	return vx - 2*dot*nx, vy - 2*dot*ny
}
