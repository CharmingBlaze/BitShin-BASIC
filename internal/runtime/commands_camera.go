package runtime

import (
	"math"

	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/value"
)

func (w *World) updateSpringArmCamera(camE *Entity, tx, ty, tz, dist, radius, damp float32, yaw, pitch float64) {
	if camE == nil || camE.node == nil {
		return
	}
	pr := pitch * math.Pi / 180.0
	yr := yaw * math.Pi / 180.0

	horiz := float64(dist) * math.Cos(pr)
	wishX := float64(tx) - math.Sin(yr)*horiz
	wishY := float64(ty) + float64(dist)*math.Sin(pr)
	wishZ := float64(tz) - math.Cos(yr)*horiz

	// Raycast collision check from target to desired camera location
	actualDist := float64(dist)
	if w.phys3 != nil {
		dirX := wishX - float64(tx)
		dirY := wishY - float64(ty)
		dirZ := wishZ - float64(tz)
		fullLen := math.Sqrt(dirX*dirX + dirY*dirY + dirZ*dirZ)
		if fullLen > 0.1 {
			normDirX := float32(dirX / fullLen)
			normDirY := float32(dirY / fullLen)
			normDirZ := float32(dirZ / fullLen)
			if hitID, hx, hy, hz, hit := w.phys3.Raycast(tx, ty, tz, normDirX*float32(fullLen), normDirY*float32(fullLen), normDirZ*float32(fullLen)); hit && hitID > 0 {
				hitDist := math.Sqrt(float64((hx-tx)*(hx-tx) + (hy-ty)*(hy-ty) + (hz-tz)*(hz-tz)))
				if hitDist < actualDist {
					actualDist = math.Max(0.5, hitDist-float64(radius))
					horiz = actualDist * math.Cos(pr)
					wishX = float64(tx) - math.Sin(yr)*horiz
					wishY = float64(ty) + actualDist*math.Sin(pr)
					wishZ = float64(tz) - math.Cos(yr)*horiz
				}
			}
		}
	}

	dt := float64(w.delta)
	if dt <= 0 {
		dt = 0.016
	}
	if dt > 0.05 {
		dt = 0.05
	}

	cn := camE.node.GetNode()
	cp := cn.Position()
	cx, cy, cz := fromG3N(cp.X, cp.Y, cp.Z)

	k := 1.0
	if camE.camFollowed && damp > 0 {
		k = 1.0 - math.Exp(-float64(damp)*dt)
	}
	camE.camFollowed = true

	nx := float64(cx) + (wishX-float64(cx))*k
	ny := float64(cy) + (wishY-float64(cy))*k
	nz := float64(cz) + (wishZ-float64(cz))*k

	gx, gy, gz := toG3N(float32(nx), float32(ny), float32(nz))
	cn.SetPosition(gx, gy, gz)

	lookX, lookY, lookZ := toG3N(tx, ty, tz)
	up := math32.Vector3{0, 1, 0}
	cn.LookAt(&math32.Vector3{lookX, lookY, lookZ}, &up)
	camE.yaw = float32(yaw)
	camE.pitch = float32(pitch)
}

func (w *World) cameraCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	_ = need
	return map[string]cmd{
		"cameraspringarm": n(func(a []value.Value) (value.Value, error) {
			camE, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			tgt, err := w.nodeOf(argI(a, 1, 0))
			if err != nil {
				return value.Value{}, err
			}
			offsetY := float32(argN(a, 2, 1.5))
			dist := float32(argN(a, 3, 5.0))
			radius := float32(argN(a, 4, 0.25))
			damp := float32(argN(a, 5, 12.0))
			yaw := argN(a, 6, float64(camE.yaw))
			pitch := argN(a, 7, 18.0)

			tp := worldPos(tgt)
			tx, ty, tz := fromG3N(tp.X, tp.Y, tp.Z)
			w.updateSpringArmCamera(camE, tx, ty+offsetY, tz, dist, radius, damp, yaw, pitch)
			return z()
		}),

		"cameraorbit": n(func(a []value.Value) (value.Value, error) {
			camE, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			tgt, err := w.nodeOf(argI(a, 1, 0))
			if err != nil {
				return value.Value{}, err
			}
			dist := argN(a, 2, 8.0)
			height := argN(a, 3, 2.0)
			yaw := argN(a, 4, float64(camE.yaw))
			pitch := argN(a, 5, 20.0)

			tp := worldPos(tgt)
			tx, ty, tz := fromG3N(tp.X, tp.Y, tp.Z)

			pr := pitch * math.Pi / 180.0
			yr := yaw * math.Pi / 180.0
			horiz := dist * math.Cos(pr)
			cx := float64(tx) - math.Sin(yr)*horiz
			cy := float64(ty) + height + dist*math.Sin(pr)
			cz := float64(tz) - math.Cos(yr)*horiz

			gx, gy, gz := toG3N(float32(cx), float32(cy), float32(cz))
			cn := camE.node.GetNode()
			cn.SetPosition(gx, gy, gz)

			lookX, lookY, lookZ := toG3N(tx, ty+float32(height*0.5), tz)
			up := math32.Vector3{0, 1, 0}
			cn.LookAt(&math32.Vector3{lookX, lookY, lookZ}, &up)
			camE.yaw = float32(yaw)
			camE.pitch = float32(pitch)
			return z()
		}),

		"camerashake": n(func(a []value.Value) (value.Value, error) {
			w.shakeTrauma = float32(argN(a, 0, 1.0))
			w.shakeDuration = float32(argN(a, 1, 0.5))
			w.shakeElapsed = 0
			w.shakeFreq = float32(argN(a, 2, 24.0))
			if w.shakeTrauma > 1.0 {
				w.shakeTrauma = 1.0
			}
			return z()
		}),

		"addcamerashake": n(func(a []value.Value) (value.Value, error) {
			trauma := float32(argN(a, 0, 0.5))
			w.shakeTrauma += trauma
			if w.shakeTrauma > 1.0 {
				w.shakeTrauma = 1.0
			}
			w.shakeDuration = float32(argN(a, 1, 0.4))
			w.shakeElapsed = 0
			w.shakeFreq = float32(argN(a, 2, 24.0))
			return z()
		}),

		"updatecamerashake": n(func(a []value.Value) (value.Value, error) {
			camE, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if w.shakeTrauma <= 0.001 {
				return z()
			}
			dt := float32(w.delta)
			if dt <= 0 {
				dt = 0.016
			}
			w.shakeElapsed += dt
			decay := float32(1.0)
			if w.shakeDuration > 0 {
				decay = 1.0 - (w.shakeElapsed / w.shakeDuration)
				if decay < 0 {
					decay = 0
					w.shakeTrauma = 0
				}
			}
			shakeMag := w.shakeTrauma * w.shakeTrauma * decay
			t := w.shakeElapsed * w.shakeFreq
			offX := math32.Sin(t*1.7) * math32.Cos(t*2.3) * shakeMag * 0.45
			offY := math32.Cos(t*1.9) * math32.Sin(t*3.1) * shakeMag * 0.35
			pitchOffset := math32.Sin(t*2.8) * shakeMag * 4.5
			yawOffset := math32.Cos(t*2.1) * shakeMag * 4.5

			cn := camE.node.GetNode()
			p := cn.Position()
			cn.SetPosition(p.X+offX, p.Y+offY, p.Z)
			camE.pitch += pitchOffset
			camE.yaw += yawOffset
			w.applyRot(camE)
			return z()
		}),

		"camerasmoothlook": n(func(a []value.Value) (value.Value, error) {
			camE, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			tx := float32(argN(a, 1, 0))
			ty := float32(argN(a, 2, 0))
			tz := float32(argN(a, 3, 0))
			speed := float32(argN(a, 4, 8.0))

			dt := w.delta
			if dt <= 0 {
				dt = 0.016
			}
			k := float32(1.0 - math.Exp(-float64(speed)*float64(dt)))

			cn := camE.node.GetNode()
			cp := cn.Position()
			cx, cy, cz := fromG3N(cp.X, cp.Y, cp.Z)

			dx := tx - cx
			dy := ty - cy
			dz := tz - cz
			targetYaw := math32.Atan2(dx, dz) * 180.0 / math32.Pi
			horiz := math32.Sqrt(dx*dx + dz*dz)
			targetPitch := -math32.Atan2(dy, horiz) * 180.0 / math32.Pi

			diffY := targetYaw - camE.yaw
			for diffY > 180 {
				diffY -= 360
			}
			for diffY < -180 {
				diffY += 360
			}
			camE.yaw += diffY * k
			camE.pitch += (targetPitch - camE.pitch) * k
			w.applyRot(camE)
			return z()
		}),

		"camerarts": n(func(a []value.Value) (value.Value, error) {
			camE, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			targetX := float32(argN(a, 1, 0))
			targetZ := float32(argN(a, 2, 0))
			zoom := float32(argN(a, 3, 20.0))
			pitch := float32(argN(a, 4, 55.0))
			yaw := float32(argN(a, 5, 0.0))

			pr := pitch * math32.Pi / 180.0
			yr := yaw * math32.Pi / 180.0
			horiz := zoom * math32.Cos(pr)
			cx := targetX - math32.Sin(yr)*horiz
			cy := zoom * math32.Sin(pr)
			cz := targetZ - math32.Cos(yr)*horiz

			gx, gy, gz := toG3N(cx, cy, cz)
			cn := camE.node.GetNode()
			cn.SetPosition(gx, gy, gz)

			lookX, lookY, lookZ := toG3N(targetX, 0, targetZ)
			up := math32.Vector3{0, 1, 0}
			cn.LookAt(&math32.Vector3{lookX, lookY, lookZ}, &up)
			camE.yaw = yaw
			camE.pitch = pitch
			return z()
		}),
	}
}
