package runtime

import (
	"fmt"

	"moonbasic/internal/phys2d"
	"moonbasic/internal/value"
)

// phys2DCommands holds the newer Chipmunk features: kinematic bodies,
// capsules, collision layers, sensors, one-way platforms, shape casts and the
// platformer character. The core 2D body commands live in commands_phys.go.
// Body arguments are the sprite (or entity) handle the body was created on,
// as with CreateBox2D.
func (w *World) phys2DCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	body := func(name string, a []value.Value, i int) (int, error) {
		w.ensurePhys2()
		id := argI(a, i, 0)
		if !w.phys2.Has(id) {
			return 0, fmt.Errorf("%s: %d is not a 2D body", name, id)
		}
		return id, nil
	}
	sensor := func(name string, a []value.Value) (int, error) {
		id, err := body(name, a, 0)
		if err != nil {
			return 0, err
		}
		if !w.phys2.IsSensor(id) {
			return 0, fmt.Errorf("%s: body %d is not a sensor (use SetSensor2D)", name, id)
		}
		return id, nil
	}
	char := func(name string, a []value.Value) (*phys2d.Character, error) {
		w.ensurePhys2()
		c := w.phys2.Character(argI(a, 0, 0))
		if c == nil {
			return nil, fmt.Errorf("%s: %d is not a 2D character (use CreateCharacter2D)", name, argI(a, 0, 0))
		}
		return c, nil
	}
	flag := func(a []value.Value, i int) bool { return argI(a, i, 1) != 0 }
	bool01 := func(b bool) value.Value {
		if b {
			return value.Num(1)
		}
		return value.Num(0)
	}
	// listQuery answers 1/0 for "is body in the list", or the first id when body is omitted.
	listQuery := func(name string, list func(int) []int) cmd {
		return n(func(a []value.Value) (value.Value, error) {
			id, err := sensor(name, a)
			if err != nil {
				return value.Value{}, err
			}
			ids := list(id)
			if len(a) < 2 {
				if len(ids) == 0 {
					return value.Num(0), nil
				}
				return value.Num(float64(ids[0])), nil
			}
			want := argI(a, 1, 0)
			for _, b := range ids {
				if b == want {
					return value.Num(1), nil
				}
			}
			return value.Num(0), nil
		})
	}
	// cast stores a hit for the Cast*2D getters and PickedEntity / PickedX/Y.
	cast := func(h phys2d.Hit) (value.Value, error) {
		if !h.OK {
			w.clearPick()
			return value.Num(0), nil
		}
		w.pickID, w.pickX, w.pickY, w.pickZ = h.ID, float32(h.X), float32(h.Y), 0
		w.pickNX, w.pickNY, w.pickNZ, w.pickFrac = float32(h.NX), float32(h.NY), 0, float32(h.Fraction)
		return value.Num(float64(h.ID)), nil
	}
	lastHit := func(get func(phys2d.Hit) float64) cmd {
		return n(func(a []value.Value) (value.Value, error) {
			if w.phys2 == nil {
				return value.Num(0), nil
			}
			return value.Num(get(w.phys2.LastHit())), nil
		})
	}
	setChar := func(name string, set func(c *phys2d.Character, v float64)) cmd {
		return n(func(a []value.Value) (value.Value, error) {
			c, err := char(name, a)
			if err != nil {
				return value.Value{}, err
			}
			set(c, argN(a, 1, 0))
			return z()
		})
	}

	setKinematic := n(func(a []value.Value) (value.Value, error) {
		id, err := body("SetKinematic2D", a, 0)
		if err != nil {
			return value.Value{}, err
		}
		w.phys2.SetKinematic(id, flag(a, 1))
		return z()
	})
	setSensor := n(func(a []value.Value) (value.Value, error) {
		id, err := body("SetSensor2D", a, 0)
		if err != nil {
			return value.Value{}, err
		}
		w.phys2.SetSensor(id, flag(a, 1))
		return z()
	})
	sensorCount := n(func(a []value.Value) (value.Value, error) {
		id, err := sensor("SensorCount2D", a)
		if err != nil {
			return value.Value{}, err
		}
		return value.Num(float64(len(w.phys2.SensorBodies(id)))), nil
	})
	sensorOverlaps := n(func(a []value.Value) (value.Value, error) {
		id, err := sensor("SensorOverlaps2D", a)
		if err != nil {
			return value.Value{}, err
		}
		return bool01(w.phys2.SensorHas(id, argI(a, 1, 0))), nil
	})
	shapeCast := n(func(a []value.Value) (value.Value, error) {
		w.ensurePhys2()
		x1, y1, x2, y2 := argN(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0)
		ww, hh := argN(a, 4, 1), argN(a, 5, 0)
		if hh <= 0 {
			return cast(w.phys2.CircleCast(x1, y1, x2, y2, ww/2, argI(a, 6, 0)))
		}
		return cast(w.phys2.BoxCast(x1, y1, x2, y2, ww, hh, argI(a, 6, 0)))
	})

	return map[string]cmd{
		"setkinematic2d": setKinematic,
		"iskinematic2d": n(func(a []value.Value) (value.Value, error) {
			id, err := body("IsKinematic2D", a, 0)
			if err != nil {
				return value.Value{}, err
			}
			return bool01(w.phys2.IsKinematic(id)), nil
		}),
		"createcapsule2d": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			id := argI(a, 0, 0)
			motion := argI(a, 4, 1)
			x, y := w.pos2D(id)
			w.phys2.AddCapsule(id, x, y, argN(a, 1, 16), argN(a, 2, 32), argN(a, 3, 1), motion != 0)
			if motion == 2 {
				w.phys2.SetKinematic(id, true)
			}
			return value.Num(float64(id)), nil
		}),
		"setcollisionlayer2d": n(func(a []value.Value) (value.Value, error) {
			id, err := body("SetCollisionLayer2D", a, 0)
			if err != nil {
				return value.Value{}, err
			}
			if l := argI(a, 1, 1); !w.phys2.SetCollisionLayer(id, l) {
				return value.Value{}, fmt.Errorf("SetCollisionLayer2D: layer %d out of range 1..%d", l, phys2d.MaxLayer)
			}
			return z()
		}),
		"getcollisionlayer2d": n(func(a []value.Value) (value.Value, error) {
			id, err := body("GetCollisionLayer2D", a, 0)
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(w.phys2.CollisionLayer(id))), nil
		}),
		"setlayercollides2d": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			la, lb := argI(a, 0, 1), argI(a, 1, 1)
			if !w.phys2.SetLayerCollides(la, lb, flag(a, 2)) {
				return value.Value{}, fmt.Errorf("SetLayerCollides2D: layers %d, %d out of range 1..%d", la, lb, phys2d.MaxLayer)
			}
			return z()
		}),
		"setcollisionmask2d": n(func(a []value.Value) (value.Value, error) {
			id, err := body("SetCollisionMask2D", a, 0)
			if err != nil {
				return value.Value{}, err
			}
			m := argN(a, 1, -1)
			mask := ^uint(0)
			if m >= 0 {
				mask = uint(uint32(int64(m)))
			}
			w.phys2.SetCollisionMask(id, mask)
			return z()
		}),
		"setsensor2d":       setSensor,
		"settrigger2d":      setSensor,
		"setbodysensor2d":   setSensor,
		"sensorcount2d":     sensorCount,
		"triggeroverlaps2d": sensorCount,
		"sensorbody2d": n(func(a []value.Value) (value.Value, error) {
			id, err := sensor("SensorBody2D", a)
			if err != nil {
				return value.Value{}, err
			}
			ids := w.phys2.SensorBodies(id)
			i := argI(a, 1, 1)
			if i < 1 || i > len(ids) {
				return value.Num(0), nil
			}
			return value.Num(float64(ids[i-1])), nil
		}),
		"sensoroverlaps2d": sensorOverlaps,
		"checktrigger2d":   sensorOverlaps,
		"sensorentered2d":  listQuery("SensorEntered2D", func(id int) []int { return w.phys2.SensorEntered(id) }),
		"sensorexited2d":   listQuery("SensorExited2D", func(id int) []int { return w.phys2.SensorExited(id) }),
		"setoneway2d": n(func(a []value.Value) (value.Value, error) {
			id, err := body("SetOneWay2D", a, 0)
			if err != nil {
				return value.Value{}, err
			}
			w.phys2.SetOneWay(id, flag(a, 1))
			return z()
		}),
		"shapecast2d": shapeCast,
		"boxcast2d":   shapeCast,
		"circlecast2d": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			return cast(w.phys2.CircleCast(argN(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0), argN(a, 4, 1), argI(a, 5, 0)))
		}),
		"casthit2d": lastHit(func(h phys2d.Hit) float64 {
			if h.OK {
				return 1
			}
			return 0
		}),
		"castbody2d":     lastHit(func(h phys2d.Hit) float64 { return float64(h.ID) }),
		"castx2d":        lastHit(func(h phys2d.Hit) float64 { return h.X }),
		"casty2d":        lastHit(func(h phys2d.Hit) float64 { return h.Y }),
		"castnormalx2d":  lastHit(func(h phys2d.Hit) float64 { return h.NX }),
		"castnormaly2d":  lastHit(func(h phys2d.Hit) float64 { return h.NY }),
		"castfraction2d": lastHit(func(h phys2d.Hit) float64 { return h.Fraction }),
		"bodyx2d": n(func(a []value.Value) (value.Value, error) {
			id, err := body("BodyX2D", a, 0)
			if err != nil {
				return value.Value{}, err
			}
			x, _, _ := w.phys2.GetPosition(id)
			return value.Num(x), nil
		}),
		"bodyy2d": n(func(a []value.Value) (value.Value, error) {
			id, err := body("BodyY2D", a, 0)
			if err != nil {
				return value.Value{}, err
			}
			_, y, _ := w.phys2.GetPosition(id)
			return value.Num(y), nil
		}),
		"freebody2d": n(func(a []value.Value) (value.Value, error) {
			if w.phys2 != nil {
				w.phys2.Remove(argI(a, 0, 0))
			}
			return z()
		}),
		"createcharacter2d": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys2()
			var id int
			var x, y, cw, ch float64
			if len(a) >= 4 {
				// A free-standing body: take a fresh handle from the sprite /
				// entity counter so the sync in UpdateWorld never moves another sprite.
				id = w.nextID
				w.nextID++
				x, y, cw, ch = argN(a, 0, 0), argN(a, 1, 0), argN(a, 2, 24), argN(a, 3, 48)
			} else {
				id = argI(a, 0, 0)
				if id == 0 {
					return value.Value{}, fmt.Errorf("CreateCharacter2D: want (x, y, w, h) or (sprite, w, h)")
				}
				x, y = w.pos2D(id)
				cw, ch = argN(a, 1, 24), argN(a, 2, 48)
			}
			if cw <= 0 || ch <= 0 {
				return value.Value{}, fmt.Errorf("CreateCharacter2D: size must be positive, got %v x %v", cw, ch)
			}
			w.phys2.AddCharacter(id, x, y, cw, ch)
			return value.Num(float64(id)), nil
		}),
		"movecharacter2d": n(func(a []value.Value) (value.Value, error) {
			c, err := char("MoveCharacter2D", a)
			if err != nil {
				return value.Value{}, err
			}
			c.Move(argN(a, 1, 0))
			return z()
		}),
		"jumpcharacter2d": n(func(a []value.Value) (value.Value, error) {
			c, err := char("JumpCharacter2D", a)
			if err != nil {
				return value.Value{}, err
			}
			c.Jump()
			return z()
		}),
		"characteronground2d": n(func(a []value.Value) (value.Value, error) {
			c, err := char("CharacterOnGround2D", a)
			if err != nil {
				return value.Value{}, err
			}
			return bool01(c.OnGround()), nil
		}),
		"charactergroundbody2d": n(func(a []value.Value) (value.Value, error) {
			c, err := char("CharacterGroundBody2D", a)
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(c.GroundBody())), nil
		}),
		"setcharacterspeed2d":      setChar("SetCharacterSpeed2D", func(c *phys2d.Character, v float64) { c.Speed = v }),
		"setcharacterjump2d":       setChar("SetCharacterJump2D", func(c *phys2d.Character, v float64) { c.JumpHeight = v }),
		"setcharactergravity2d":    setChar("SetCharacterGravity2D", func(c *phys2d.Character, v float64) { c.GravityScale = v }),
		"setcharacteraircontrol2d": setChar("SetCharacterAirControl2D", func(c *phys2d.Character, v float64) { c.AirControl = v }),
		"setcharactercoyote2d":     setChar("SetCharacterCoyote2D", func(c *phys2d.Character, v float64) { c.Coyote = v }),
		"setcharacterjumpbuffer2d": setChar("SetCharacterJumpBuffer2D", func(c *phys2d.Character, v float64) { c.JumpBuffer = v }),
		"setcharactermaxslope2d":   setChar("SetCharacterMaxSlope2D", func(c *phys2d.Character, v float64) { c.MaxSlope = v }),
		"freecharacter2d": n(func(a []value.Value) (value.Value, error) {
			if w.phys2 != nil {
				w.phys2.Remove(argI(a, 0, 0))
			}
			return z()
		}),
	}
}
