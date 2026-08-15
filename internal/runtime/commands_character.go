package runtime

import "bitshinbasic/internal/value"

func (w *World) characterCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	blitzPos := func(id int) (float32, float32, float32) {
		e := w.ents[id]
		if e == nil {
			return 0, 0, 0
		}
		p := e.node.GetNode().Position()
		return fromG3N(p.X, p.Y, p.Z)
	}
	return map[string]cmd{
		"createcharactercontroller": need(func(a []value.Value) (value.Value, error) {
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
		"setcharactershape": need(func(a []value.Value) (value.Value, error) {
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
	}
}
