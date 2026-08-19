package runtime

import (
	"math"

	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/value"
)

type fpsCtrl struct {
	id          int
	camID       int
	height      float32
	radius      float32
	walkSpeed   float32
	sprintSpeed float32
	jumpForce   float32
	pitch       float32
	yaw         float32
	vx, vy, vz  float32
	grounded    bool
	crouching   bool
	crouchRatio float32
	headBobT    float32
	coyoteTimer float32
}

type tpsCtrl struct {
	id          int
	camID       int
	height      float32
	radius      float32
	moveSpeed   float32
	sprintSpeed float32
	jumpForce   float32
	yaw         float32
	targetYaw   float32
	vx, vy, vz  float32
	grounded    bool
	aimMode     bool
	coyoteTimer float32
}

type topDownCtrl struct {
	id           int
	speed        float32
	rotSpeed     float32
	aimYaw       float32
	vx, vy, vz   float32
	dashTimer    float32
	dashCooldown float32
}

type platformerCtrl struct {
	id          int
	speed       float32
	jumpForce   float32
	maxJumps    int
	jumpsLeft   int
	vx, vy      float32
	grounded    bool
	onWall      int // -1 left, +1 right, 0 none
	coyoteTimer float32
	jumpBuffer  float32
}

func (w *World) characterCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	blitzPos := func(id int) (float32, float32, float32) {
		e := w.ents[id]
		if e == nil || e.node == nil {
			return 0, 0, 0
		}
		p := e.node.GetNode().Position()
		return fromG3N(p.X, p.Y, p.Z)
	}

	return map[string]cmd{
		"createcharactercontroller": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			id := argI(a, 0, 0)
			px, py, pz := blitzPos(id)
			w.phys3.AddCharacterController(
				id, px, py, pz,
				float32(argN(a, 1, 1.8)),
				float32(argN(a, 2, 0.4)),
				float32(argN(a, 3, 50)),
				float32(argN(a, 4, 100)),
			)
			if e := w.ents[id]; e != nil {
				e.bodyType = 3
			}
			return value.Num(float64(id)), nil
		}),
		"setcharactershape": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			w.phys3.SetCharacterShape(argI(a, 0, 0), argS(a, 1), float32(argN(a, 2, 1.8)), float32(argN(a, 3, 0.4)))
			return z()
		}),
		"getcharactergroundstate": n(func(a []value.Value) (value.Value, error) {
			if w.phys3 == nil {
				return value.Num(3), nil
			}
			return value.Num(float64(w.phys3.CharacterGroundState(argI(a, 0, 0)))), nil
		}),
		"getcharactergroundnormal": n(func(a []value.Value) (value.Value, error) {
			if w.phys3 == nil {
				return value.Num(0), nil
			}
			nx, ny, nz := w.phys3.CharacterGroundNormal(argI(a, 0, 0))
			switch argI(a, 1, 1) {
			case 0:
				return value.Num(float64(nx)), nil
			case 2:
				return value.Num(float64(nz)), nil
			default:
				return value.Num(float64(ny)), nil
			}
		}),
		"getcharactercontact": n(func(a []value.Value) (value.Value, error) {
			if w.phys3 == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(w.phys3.CharacterContact(argI(a, 0, 0)))), nil
		}),

		// --- High-Level Modern First-Person Character Controller ---
		"createfpscontroller": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			camID := argI(a, 1, 0)
			height := float32(argN(a, 2, 1.8))
			radius := float32(argN(a, 3, 0.4))
			walkSpeed := float32(argN(a, 4, 6.0))
			sprintSpeed := float32(argN(a, 5, 10.5))
			jumpForce := float32(argN(a, 6, 9.5))

			if w.fpsControllers == nil {
				w.fpsControllers = map[int]*fpsCtrl{}
			}
			w.fpsControllers[id] = &fpsCtrl{
				id:          id,
				camID:       camID,
				height:      height,
				radius:      radius,
				walkSpeed:   walkSpeed,
				sprintSpeed: sprintSpeed,
				jumpForce:   jumpForce,
				crouchRatio: 1.0,
			}
			w.ensurePhys3()
			px, py, pz := blitzPos(id)
			w.phys3.AddCharacterController(id, px, py, pz, height, radius, 50, 80)
			if e := w.ents[id]; e != nil {
				e.bodyType = 3
			}
			return value.Num(float64(id)), nil
		}),
		"updatefps": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.fpsControllers[id]
			if c == nil {
				return z()
			}
			walkX := float32(argN(a, 1, 0))
			walkZ := float32(argN(a, 2, 0))
			isJump := argI(a, 3, 0) != 0
			isSprint := argI(a, 4, 0) != 0
			isCrouch := argI(a, 5, 0) != 0
			mouseSens := float32(argN(a, 6, 0.15))

			dt := float32(w.delta)
			if dt <= 0 {
				dt = 0.016
			}
			if dt > 0.05 {
				dt = 0.05
			}

			// Mouse look
			mxd := w.mxs * mouseSens
			myd := w.mys * mouseSens
			c.yaw += mxd
			c.pitch -= myd
			if c.pitch > 89.0 {
				c.pitch = 89.0
			}
			if c.pitch < -89.0 {
				c.pitch = -89.0
			}

			// Smooth crouch
			targetCrouch := float32(1.0)
			if isCrouch {
				targetCrouch = 0.55
			}
			c.crouchRatio += (targetCrouch - c.crouchRatio) * float32(math.Min(1.0, float64(dt*12.0)))

			// Speed
			spd := c.walkSpeed
			if isSprint && !isCrouch {
				spd = c.sprintSpeed
			}
			if isCrouch {
				spd *= 0.5
			}

			// Direction relative to camera yaw
			yr := c.yaw * math32.Pi / 180.0
			sinY := math32.Sin(yr)
			cosY := math32.Cos(yr)

			moveX := walkX*cosY + walkZ*sinY
			moveZ := -walkX*sinY + walkZ*cosY
			lenSq := moveX*moveX + moveZ*moveZ
			if lenSq > 1.0 {
				invL := 1.0 / math32.Sqrt(lenSq)
				moveX *= invL
				moveZ *= invL
			}

			// Horizontal velocity smoothing
			accel := float32(18.0)
			if !c.grounded {
				accel = 5.0
			}
			c.vx += (moveX*spd - c.vx) * float32(math.Min(1.0, float64(dt*accel)))
			c.vz += (moveZ*spd - c.vz) * float32(math.Min(1.0, float64(dt*accel)))

			// Ground detection & jump
			groundState := w.phys3.CharacterGroundState(id)
			c.grounded = (groundState == 0) // OnGround
			if c.grounded {
				c.coyoteTimer = 0.12
				c.vy = 0
			} else {
				c.coyoteTimer -= dt
				c.vy -= 22.0 * dt // Gravity
				if c.vy < -35.0 {
					c.vy = -35.0
				}
			}

			if isJump && (c.grounded || c.coyoteTimer > 0) {
				c.vy = c.jumpForce
				c.coyoteTimer = 0
				c.grounded = false
			}

			// Move physical character
			w.phys3.SetVelocity(id, c.vx, c.vy, c.vz)

			// Step head-bobbing
			if c.grounded && lenSq > 0.01 {
				c.headBobT += dt * spd * 1.8
			}
			bobOffset := math32.Sin(c.headBobT) * 0.045 * float32(math.Min(1.0, float64(lenSq)))

			// Update entity & camera position
			px, py, pz := blitzPos(id)
			if e := w.ents[id]; e != nil && e.node != nil {
				gx, gy, gz := toG3N(px, py, pz)
				e.node.GetNode().SetPosition(gx, gy, gz)
				e.yaw = c.yaw
				w.applyRot(e)
			}
			if camE := w.ents[c.camID]; camE != nil && camE.node != nil {
				eyeH := c.height * c.crouchRatio * 0.92
				gx, gy, gz := toG3N(px, py+eyeH+bobOffset, pz)
				camE.node.GetNode().SetPosition(gx, gy, gz)
				camE.pitch = c.pitch
				camE.yaw = c.yaw
				camE.roll = 0
				w.applyRot(camE)
			}

			return z()
		}),
		"getfpspitch": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			if c := w.fpsControllers[id]; c != nil {
				return value.Num(float64(c.pitch)), nil
			}
			return value.Num(0), nil
		}),
		"getfpsyaw": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			if c := w.fpsControllers[id]; c != nil {
				return value.Num(float64(c.yaw)), nil
			}
			return value.Num(0), nil
		}),
		"isfpsgrounded": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			if c := w.fpsControllers[id]; c != nil {
				if c.grounded {
					return value.Num(1), nil
				}
				return value.Num(0), nil
			}
			return value.Num(0), nil
		}),

		// --- High-Level Third-Person Action (TPS) Controller ---
		"createtpscontroller": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			camID := argI(a, 1, 0)
			height := float32(argN(a, 2, 1.8))
			radius := float32(argN(a, 3, 0.45))
			moveSpeed := float32(argN(a, 4, 6.5))
			sprintSpeed := float32(argN(a, 5, 11.0))
			jumpForce := float32(argN(a, 6, 10.0))

			if w.tpsControllers == nil {
				w.tpsControllers = map[int]*tpsCtrl{}
			}
			w.tpsControllers[id] = &tpsCtrl{
				id:          id,
				camID:       camID,
				height:      height,
				radius:      radius,
				moveSpeed:   moveSpeed,
				sprintSpeed: sprintSpeed,
				jumpForce:   jumpForce,
			}
			w.ensurePhys3()
			px, py, pz := blitzPos(id)
			w.phys3.AddCharacterController(id, px, py, pz, height, radius, 50, 80)
			if e := w.ents[id]; e != nil {
				e.bodyType = 3
			}
			return value.Num(float64(id)), nil
		}),
		"updatetps": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.tpsControllers[id]
			if c == nil {
				return z()
			}
			moveX := float32(argN(a, 1, 0))
			moveZ := float32(argN(a, 2, 0))
			isJump := argI(a, 3, 0) != 0
			isSprint := argI(a, 4, 0) != 0
			isAim := argI(a, 5, 0) != 0
			camDist := float32(argN(a, 6, 4.5))
			camPitch := float32(argN(a, 7, 16.0))
			camYaw := float32(argN(a, 8, 0.0))

			dt := float32(w.delta)
			if dt <= 0 {
				dt = 0.016
			}
			if dt > 0.05 {
				dt = 0.05
			}

			spd := c.moveSpeed
			if isSprint {
				spd = c.sprintSpeed
			}
			if isAim {
				spd *= 0.65
			}

			// Move relative to camera yaw
			cyr := camYaw * math32.Pi / 180.0
			sinCY := math32.Sin(cyr)
			cosCY := math32.Cos(cyr)

			worldMoveX := moveX*cosCY + moveZ*sinCY
			worldMoveZ := -moveX*sinCY + moveZ*cosCY
			lenSq := worldMoveX*worldMoveX + worldMoveZ*worldMoveZ
			if lenSq > 1.0 {
				invL := 1.0 / math32.Sqrt(lenSq)
				worldMoveX *= invL
				worldMoveZ *= invL
			}

			// Character rotation
			if isAim {
				c.yaw = camYaw
			} else if lenSq > 0.01 {
				targetAngle := math32.Atan2(worldMoveX, worldMoveZ) * 180.0 / math32.Pi
				diff := targetAngle - c.yaw
				for diff > 180 {
					diff -= 360
				}
				for diff < -180 {
					diff += 360
				}
				c.yaw += diff * float32(math.Min(1.0, float64(dt*14.0)))
			}

			c.vx += (worldMoveX*spd - c.vx) * float32(math.Min(1.0, float64(dt*16.0)))
			c.vz += (worldMoveZ*spd - c.vz) * float32(math.Min(1.0, float64(dt*16.0)))

			// Ground & jump
			groundState := w.phys3.CharacterGroundState(id)
			c.grounded = (groundState == 0)
			if c.grounded {
				c.coyoteTimer = 0.12
				c.vy = 0
			} else {
				c.coyoteTimer -= dt
				c.vy -= 22.0 * dt
				if c.vy < -35.0 {
					c.vy = -35.0
				}
			}

			if isJump && (c.grounded || c.coyoteTimer > 0) {
				c.vy = c.jumpForce
				c.coyoteTimer = 0
				c.grounded = false
			}

			w.phys3.SetVelocity(id, c.vx, c.vy, c.vz)

			// Update entity
			px, py, pz := blitzPos(id)
			if e := w.ents[id]; e != nil && e.node != nil {
				gx, gy, gz := toG3N(px, py, pz)
				e.node.GetNode().SetPosition(gx, gy, gz)
				e.yaw = c.yaw
				w.applyRot(e)
			}

			// SpringArm Camera update
			if camE := w.ents[c.camID]; camE != nil && camE.node != nil {
				w.updateSpringArmCamera(camE, px, py+c.height*0.85, pz, camDist, 0.25, 12.0, float64(camYaw), float64(camPitch))
			}

			return z()
		}),
		"gettpsyaw": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			if c := w.tpsControllers[id]; c != nil {
				return value.Num(float64(c.yaw)), nil
			}
			return value.Num(0), nil
		}),
		"istpsgrounded": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			if c := w.tpsControllers[id]; c != nil {
				if c.grounded {
					return value.Num(1), nil
				}
				return value.Num(0), nil
			}
			return value.Num(0), nil
		}),

		// --- High-Level Top-Down / Twin-Stick Controller ---
		"createtopdowncontroller": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			speed := float32(argN(a, 1, 8.0))
			rotSpeed := float32(argN(a, 2, 18.0))

			if w.topDownControllers == nil {
				w.topDownControllers = map[int]*topDownCtrl{}
			}
			w.topDownControllers[id] = &topDownCtrl{
				id:       id,
				speed:    speed,
				rotSpeed: rotSpeed,
			}
			w.ensurePhys3()
			px, py, pz := blitzPos(id)
			w.phys3.AddCharacterController(id, px, py, pz, 1.8, 0.45, 50, 80)
			return value.Num(float64(id)), nil
		}),
		"updatetopdown": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.topDownControllers[id]
			if c == nil {
				return z()
			}
			moveX := float32(argN(a, 1, 0))
			moveZ := float32(argN(a, 2, 0))
			aimX := float32(argN(a, 3, 0))
			aimZ := float32(argN(a, 4, 0))
			isDash := argI(a, 5, 0) != 0

			dt := float32(w.delta)
			if dt <= 0 {
				dt = 0.016
			}
			if dt > 0.05 {
				dt = 0.05
			}

			px, py, pz := blitzPos(id)

			// Aim rotation
			dx := aimX - px
			dz := aimZ - pz
			if dx*dx+dz*dz > 0.01 {
				targetAngle := math32.Atan2(dx, dz) * 180.0 / math32.Pi
				c.aimYaw = targetAngle
			}

			spd := c.speed
			if isDash && c.dashCooldown <= 0 {
				c.dashTimer = 0.18
				c.dashCooldown = 0.8
			}
			if c.dashTimer > 0 {
				c.dashTimer -= dt
				spd *= 2.8
			}
			if c.dashCooldown > 0 {
				c.dashCooldown -= dt
			}

			lenSq := moveX*moveX + moveZ*moveZ
			if lenSq > 1.0 {
				invL := 1.0 / math32.Sqrt(lenSq)
				moveX *= invL
				moveZ *= invL
			}

			c.vx += (moveX*spd - c.vx) * float32(math.Min(1.0, float64(dt*18.0)))
			c.vz += (moveZ*spd - c.vz) * float32(math.Min(1.0, float64(dt*18.0)))

			w.phys3.SetVelocity(id, c.vx, 0, c.vz)

			if e := w.ents[id]; e != nil && e.node != nil {
				gx, gy, gz := toG3N(px, py, pz)
				e.node.GetNode().SetPosition(gx, gy, gz)
				e.yaw = c.aimYaw
				w.applyRot(e)
			}
			return z()
		}),
		"gettopdownaimyaw": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			if c := w.topDownControllers[id]; c != nil {
				return value.Num(float64(c.aimYaw)), nil
			}
			return value.Num(0), nil
		}),

		// --- High-Level 2.5D / Platformer Character Controller ---
		"createplatformercontroller": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			speed := float32(argN(a, 1, 8.5))
			jumpForce := float32(argN(a, 2, 12.0))
			maxJumps := argI(a, 3, 2)

			if w.platformerControllers == nil {
				w.platformerControllers = map[int]*platformerCtrl{}
			}
			w.platformerControllers[id] = &platformerCtrl{
				id:        id,
				speed:     speed,
				jumpForce: jumpForce,
				maxJumps:  maxJumps,
				jumpsLeft: maxJumps,
			}
			w.ensurePhys3()
			px, py, pz := blitzPos(id)
			w.phys3.AddCharacterController(id, px, py, pz, 1.8, 0.4, 50, 80)
			return value.Num(float64(id)), nil
		}),
		"updateplatformer": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			c := w.platformerControllers[id]
			if c == nil {
				return z()
			}
			moveX := float32(argN(a, 1, 0))
			jumpPressed := argI(a, 2, 0) != 0
			jumpHeld := argI(a, 3, 0) != 0

			dt := float32(w.delta)
			if dt <= 0 {
				dt = 0.016
			}
			if dt > 0.05 {
				dt = 0.05
			}

			// Horizontal movement
			c.vx += (moveX*c.speed - c.vx) * float32(math.Min(1.0, float64(dt*20.0)))

			// Ground & coyote time
			groundState := w.phys3.CharacterGroundState(id)
			c.grounded = (groundState == 0)
			if c.grounded {
				c.coyoteTimer = 0.12
				c.jumpsLeft = c.maxJumps
				c.vy = 0
			} else {
				c.coyoteTimer -= dt
				// Variable jump height: cut gravity if held, stronger gravity if released or falling
				grav := float32(28.0)
				if !jumpHeld && c.vy > 0 {
					grav = 55.0
				} else if c.vy < 0 {
					grav = 36.0
				}
				c.vy -= grav * dt
				if c.vy < -30.0 {
					c.vy = -30.0
				}
			}

			if jumpPressed {
				c.jumpBuffer = 0.15
			} else if c.jumpBuffer > 0 {
				c.jumpBuffer -= dt
			}

			if c.jumpBuffer > 0 {
				if c.grounded || c.coyoteTimer > 0 {
					c.vy = c.jumpForce
					c.jumpsLeft = c.maxJumps - 1
					c.coyoteTimer = 0
					c.jumpBuffer = 0
					c.grounded = false
				} else if c.jumpsLeft > 0 {
					c.vy = c.jumpForce * 0.9
					c.jumpsLeft--
					c.jumpBuffer = 0
				}
			}

			w.phys3.SetVelocity(id, c.vx, c.vy, 0)

			px, py, pz := blitzPos(id)
			if e := w.ents[id]; e != nil && e.node != nil {
				gx, gy, gz := toG3N(px, py, pz)
				e.node.GetNode().SetPosition(gx, gy, gz)
				if moveX > 0.1 {
					e.yaw = 90
				} else if moveX < -0.1 {
					e.yaw = -90
				}
				w.applyRot(e)
			}

			return z()
		}),
		"isplatformergrounded": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			if c := w.platformerControllers[id]; c != nil {
				if c.grounded {
					return value.Num(1), nil
				}
				return value.Num(0), nil
			}
			return value.Num(0), nil
		}),
	}
}
