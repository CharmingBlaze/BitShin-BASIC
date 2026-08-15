//go:build !nojolt && windows

package phys3d

import "bitshinbasic/internal/jolt"

func (w *joltWorld) GetRotation(id int) (float32, float32, float32, float32, bool) {
	b, ok := w.bodyOf(id)
	if !ok {
		return 0, 0, 0, 1, false
	}
	q := w.bi.GetRotation(b)
	return q.X, q.Y, q.Z, q.W, true
}

func (w *joltWorld) SetRotation(id int, x, y, z, qw float32) {
	if b, ok := w.bodyOf(id); ok {
		w.bi.SetRotation(b, jolt.Quat{X: x, Y: y, Z: z, W: qw})
	}
}

func (w *joltWorld) SetCCD(id int, on bool) int {
	b, ok := w.bodyOf(id)
	if !ok {
		return 0
	}
	w.bi.SetMotionQualityLinearCast(b, on)
	return 1
}

func (w *joltWorld) EnableContacts() {
	w.ps.EnableContactListener(true)
}

func (w *joltWorld) LookupBody(bodyValue uint32) int {
	if id, ok := w.bodyVal[bodyValue]; ok {
		return id
	}
	return 0
}

func (w *joltWorld) PollContacts(max int) []ContactEvent {
	raw := w.ps.PollContactEvents(max)
	if len(raw) == 0 {
		return nil
	}
	out := make([]ContactEvent, 0, len(raw))
	for i := 0; i < len(raw); i++ {
		ev := raw[i]
		a := w.LookupBody(ev.BodyA)
		b := w.LookupBody(ev.BodyB)
		if a == 0 && b == 0 {
			continue
		}
		out = append(out, ContactEvent{
			Kind: int(ev.Type),
			A:    a,
			B:    b,
			X:    ev.X, Y: ev.Y, Z: ev.Z,
			NX: ev.NX, NY: ev.NY, NZ: ev.NZ,
		})
	}
	return out
}
