package runtime

import (
	"math"

	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/value"
)

type activeTween struct {
	kind     int // 0 pos, 1 rot, 2 col
	entID    int
	start    [3]float32
	target   [3]float32
	duration float32
	elapsed  float32
	ease     string
}

func evaluateEase(t float32, mode string) float32 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	switch mode {
	case "in":
		return t * t
	case "out":
		return t * (2 - t)
	case "inout":
		if t < 0.5 {
			return 2 * t * t
		}
		return -1 + (4-2*t)*t
	case "smooth":
		return t * t * (3 - 2*t)
	case "smoother":
		return t * t * t * (t*(t*6-15) + 10)
	default:
		return t
	}
}

func (w *World) updateTweens(dt float32) {
	if len(w.actTweens) == 0 {
		return
	}
	active := w.actTweens[:0]
	for _, tw := range w.actTweens {
		e := w.ents[tw.entID]
		if e == nil || e.node == nil {
			continue
		}
		tw.elapsed += dt
		ratio := tw.elapsed / tw.duration
		if ratio > 1.0 {
			ratio = 1.0
		}
		f := evaluateEase(ratio, tw.ease)
		cx := tw.start[0] + (tw.target[0]-tw.start[0])*f
		cy := tw.start[1] + (tw.target[1]-tw.start[1])*f
		cz := tw.start[2] + (tw.target[2]-tw.start[2])*f

		switch tw.kind {
		case 0: // Position
			gx, gy, gz := toG3N(cx, cy, cz)
			e.node.GetNode().SetPosition(gx, gy, gz)
		case 1: // Rotation
			e.pitch = cx
			e.yaw = cy
			e.roll = cz
			w.applyRot(e)
		case 2: // Color
			w.setEntityRGB(e, cx, cy, cz)
		}

		if ratio < 1.0 {
			active = append(active, tw)
		}
	}
	w.actTweens = active
}

func (w *World) gameplayCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	_ = need
	return map[string]cmd{
		// --- Math & Interpolation Helpers ---
		"lerp": n(func(a []value.Value) (value.Value, error) {
			v0 := argN(a, 0, 0)
			v1 := argN(a, 1, 0)
			t := argN(a, 2, 0)
			return value.Num(v0 + (v1-v0)*t), nil
		}),

		"lerpangle": n(func(a []value.Value) (value.Value, error) {
			a0 := argN(a, 0, 0)
			a1 := argN(a, 1, 0)
			t := argN(a, 2, 0)
			diff := math.Mod(a1-a0+180, 360) - 180
			if diff < -180 {
				diff += 360
			}
			return value.Num(a0 + diff*t), nil
		}),

		"smoothdamp": n(func(a []value.Value) (value.Value, error) {
			current := argN(a, 0, 0)
			target := argN(a, 1, 0)
			smoothTime := argN(a, 2, 0.25)
			maxSpeed := argN(a, 3, 1000.0)

			dt := float64(w.delta)
			if dt <= 0 {
				dt = 0.016
			}
			if smoothTime < 0.0001 {
				smoothTime = 0.0001
			}
			omega := 2.0 / smoothTime
			x := omega * dt
			exp := 1.0 / (1.0 + x + 0.48*x*x + 0.235*x*x*x)
			change := current - target
			maxChange := maxSpeed * smoothTime
			if change > maxChange {
				change = maxChange
			} else if change < -maxChange {
				change = -maxChange
			}
			temp := (0 + omega*change) * dt
			output := target + (change+temp)*exp
			if (target-current > 0) == (output > target) {
				output = target
			}
			return value.Num(output), nil
		}),

		"movetowards": n(func(a []value.Value) (value.Value, error) {
			current := argN(a, 0, 0)
			target := argN(a, 1, 0)
			maxDelta := argN(a, 2, 0.1)
			if math.Abs(target-current) <= maxDelta {
				return value.Num(target), nil
			}
			return value.Num(current + math.Copysign(maxDelta, target-current)), nil
		}),

		"clamp": n(func(a []value.Value) (value.Value, error) {
			val := argN(a, 0, 0)
			minV := argN(a, 1, 0)
			maxV := argN(a, 2, 1)
			if val < minV {
				return value.Num(minV), nil
			}
			if val > maxV {
				return value.Num(maxV), nil
			}
			return value.Num(val), nil
		}),

		"wrapangle": n(func(a []value.Value) (value.Value, error) {
			ang := argN(a, 0, 0)
			for ang > 180 {
				ang -= 360
			}
			for ang < -180 {
				ang += 360
			}
			return value.Num(ang), nil
		}),

		"anglediff": n(func(a []value.Value) (value.Value, error) {
			a0 := argN(a, 0, 0)
			a1 := argN(a, 1, 0)
			diff := math.Mod(a1-a0+180, 360) - 180
			if diff < -180 {
				diff += 360
			}
			return value.Num(diff), nil
		}),

		"distance3d": n(func(a []value.Value) (value.Value, error) {
			x1 := argN(a, 0, 0)
			y1 := argN(a, 1, 0)
			z1 := argN(a, 2, 0)
			x2 := argN(a, 3, 0)
			y2 := argN(a, 4, 0)
			z2 := argN(a, 5, 0)
			dx := x2 - x1
			dy := y2 - y1
			dz := z2 - z1
			return value.Num(math.Sqrt(dx*dx + dy*dy + dz*dz)), nil
		}),

		"distance2d": n(func(a []value.Value) (value.Value, error) {
			x1 := argN(a, 0, 0)
			y1 := argN(a, 1, 0)
			x2 := argN(a, 2, 0)
			y2 := argN(a, 3, 0)
			dx := x2 - x1
			dy := y2 - y1
			return value.Num(math.Sqrt(dx*dx + dy*dy)), nil
		}),

		// --- Input & Gameplay Helpers ---
		"getaxis": n(func(a []value.Value) (value.Value, error) {
			negKey := argI(a, 0, 0)
			posKey := argI(a, 1, 0)
			val := float64(0)
			if w.keyDown(posKey) {
				val += 1.0
			}
			if w.keyDown(negKey) {
				val -= 1.0
			}
			return value.Num(val), nil
		}),

		"isgrounded": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			rayLen := float32(argN(a, 1, 1.2))
			e := w.ents[id]
			if e == nil || e.node == nil || w.phys3 == nil {
				return value.Num(0), nil
			}
			p := e.node.GetNode().Position()
			px, py, pz := fromG3N(p.X, p.Y, p.Z)
			if hitID, _, _, _, hit := w.phys3.Raycast(px, py, pz, 0, -rayLen, 0); hit && hitID != id {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),

		"getmouseworldx": n(func(a []value.Value) (value.Value, error) {
			camID := argI(a, 0, 0)
			planeY := float32(argN(a, 1, 0))
			camE, err := w.ent(camID)
			if err != nil || camE.node == nil {
				return value.Num(0), nil
			}
			camNode := camE.node.GetNode()
			cp := camNode.Position()
			cx, cy, _ := fromG3N(cp.X, cp.Y, cp.Z)

			pitchR := float64(camE.pitch) * math.Pi / 180.0
			yawR := float64(camE.yaw) * math.Pi / 180.0
			dirX := math.Sin(yawR) * math.Cos(pitchR)
			dirY := -math.Sin(pitchR)

			if math.Abs(dirY) > 0.001 {
				t := float64(planeY-cy) / dirY
				return value.Num(float64(cx) + dirX*t), nil
			}
			return value.Num(float64(cx)), nil
		}),

		"getmouseworldz": n(func(a []value.Value) (value.Value, error) {
			camID := argI(a, 0, 0)
			planeY := float32(argN(a, 1, 0))
			camE, err := w.ent(camID)
			if err != nil || camE.node == nil {
				return value.Num(0), nil
			}
			camNode := camE.node.GetNode()
			cp := camNode.Position()
			_, cy, cz := fromG3N(cp.X, cp.Y, cp.Z)

			pitchR := float64(camE.pitch) * math.Pi / 180.0
			yawR := float64(camE.yaw) * math.Pi / 180.0
			dirY := -math.Sin(pitchR)
			dirZ := math.Cos(yawR) * math.Cos(pitchR)

			if math.Abs(dirY) > 0.001 {
				t := float64(planeY-cy) / dirY
				return value.Num(float64(cz) + dirZ*t), nil
			}
			return value.Num(float64(cz)), nil
		}),

		// --- 3D Spatial Audio ---
		"playsound3d": n(func(a []value.Value) (value.Value, error) {
			sndID := argI(a, 0, 0)
			x := float32(argN(a, 1, 0))
			y := float32(argN(a, 2, 0))
			z := float32(argN(a, 3, 0))
			maxDist := float32(argN(a, 4, 30.0))
			baseVol := float32(argN(a, 5, 1.0))

			if maxDist < 1.0 {
				maxDist = 1.0
			}

			// Calculate listener distance from active camera or listener entity
			lx, ly, lz := float32(0), float32(0), float32(0)
			if w.cam != nil {
				var p math32.Vector3
				w.cam.GetNode().WorldPosition(&p)
				lx, ly, lz = fromG3N(p.X, p.Y, p.Z)
			}

			dx := x - lx
			dy := y - ly
			dz := z - lz
			dist := math32.Sqrt(dx*dx + dy*dy + dz*dz)
			if dist > maxDist {
				return value.Num(0), nil
			}

			attenuation := 1.0 - (dist / maxDist)
			vol := baseVol * attenuation * attenuation
			if vol <= 0.01 {
				return value.Num(0), nil
			}

			// Play sound slot with volume attenuation
			slot, ok := w.sounds[sndID]
			if !ok || slot == nil || slot.clip == nil {
				return value.Num(0), nil
			}
			w.startClipWithVol(sndID, false, float64(vol))
			return value.Num(1), nil
		}),

		// --- Time Scale (Slow Motion / Fast Forward / Bullet Time) ---
		"settimescale": n(func(a []value.Value) (value.Value, error) {
			scale := float32(argN(a, 0, 1.0))
			if scale < 0 {
				scale = 0
			}
			if scale > 10.0 {
				scale = 10.0
			}
			w.timeScale = scale
			return z()
		}),

		"gettimescale": n(func(a []value.Value) (value.Value, error) {
			ts := w.timeScale
			if ts <= 0 {
				ts = 1.0
			}
			return value.Num(float64(ts)), nil
		}),

		"timescale": n(func(a []value.Value) (value.Value, error) {
			if len(a) > 0 {
				scale := float32(argN(a, 0, 1.0))
				if scale < 0 {
					scale = 0
				}
				w.timeScale = scale
				return z()
			}
			ts := w.timeScale
			if ts <= 0 {
				ts = 1.0
			}
			return value.Num(float64(ts)), nil
		}),

		// --- Tweens ---
		"tweenposition": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			e := w.ents[id]
			if e == nil || e.node == nil {
				return z()
			}
			p := e.node.GetNode().Position()
			sx, sy, sz := fromG3N(p.X, p.Y, p.Z)
			tx := float32(argN(a, 1, float64(sx)))
			ty := float32(argN(a, 2, float64(sy)))
			tz := float32(argN(a, 3, float64(sz)))
			dur := float32(argN(a, 4, 1.0))
			ease := argS(a, 5)
			if dur <= 0.001 {
				dur = 0.001
			}
			w.actTweens = append(w.actTweens, &activeTween{
				kind:     0,
				entID:    id,
				start:    [3]float32{sx, sy, sz},
				target:   [3]float32{tx, ty, tz},
				duration: dur,
				ease:     ease,
			})
			return z()
		}),

		"followpath": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			if w.ents[id] == nil {
				return z()
			}
			speed, loop, pts := parsePathArgs(a)
			if len(pts) < 2 {
				return z()
			}
			if w.pathFollows == nil {
				w.pathFollows = map[int]*pathFollow{}
			}
			w.pathFollows[id] = &pathFollow{pts: pts, speed: speed, loop: loop}
			return z()
		}),

		"tweenrotation": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			e := w.ents[id]
			if e == nil || e.node == nil {
				return z()
			}
			sx, sy, sz := e.pitch, e.yaw, e.roll
			tx := float32(argN(a, 1, float64(sx)))
			ty := float32(argN(a, 2, float64(sy)))
			tz := float32(argN(a, 3, float64(sz)))
			dur := float32(argN(a, 4, 1.0))
			ease := argS(a, 5)
			if dur <= 0.001 {
				dur = 0.001
			}
			w.actTweens = append(w.actTweens, &activeTween{
				kind:     1,
				entID:    id,
				start:    [3]float32{sx, sy, sz},
				target:   [3]float32{tx, ty, tz},
				duration: dur,
				ease:     ease,
			})
			return z()
		}),

		"tweencolor": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			e := w.ents[id]
			if e == nil {
				return z()
			}
			sx, sy, sz := float32(255), float32(255), float32(255)
			tx := float32(argN(a, 1, 255))
			ty := float32(argN(a, 2, 255))
			tz := float32(argN(a, 3, 255))
			dur := float32(argN(a, 4, 1.0))
			ease := argS(a, 5)
			if dur <= 0.001 {
				dur = 0.001
			}
			w.actTweens = append(w.actTweens, &activeTween{
				kind:     2,
				entID:    id,
				start:    [3]float32{sx, sy, sz},
				target:   [3]float32{tx, ty, tz},
				duration: dur,
				ease:     ease,
			})
			return z()
		}),
	}
}
