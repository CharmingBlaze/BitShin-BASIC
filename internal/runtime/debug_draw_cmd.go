package runtime

import (
	"fmt"
	"strings"

	"moonbasic/internal/value"
)

func dbgOverlayList() string {
	names := make([]string, 0, len(dbgOverlayNames)+1)
	for _, o := range dbgOverlayNames {
		names = append(names, o.name)
	}
	return strings.Join(append(names, "all"), ", ")
}

// dbgSeconds reads the optional lifetime after the colour.
func dbgSeconds(a []value.Value, i int) float32 {
	if len(a) > i {
		return float32(argN(a, i, 0))
	}
	return 0
}

func (w *World) debugDrawCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		// DebugDraw what$ [, on]
		"debugdraw": n(func(a []value.Value) (value.Value, error) {
			bit, ok := dbgOverlayBit(argS(a, 0))
			if !ok {
				return value.Value{}, fmt.Errorf("DebugDraw: unknown overlay %q (use %s)", argS(a, 0), dbgOverlayList())
			}
			w.setDebugFlags(bit, argI(a, 1, 1) != 0)
			return z()
		}),
		// DebugDrawEnabled(what$)
		"debugdrawenabled": n(func(a []value.Value) (value.Value, error) {
			bit, ok := dbgOverlayBit(argS(a, 0))
			if !ok || w.dbg == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(b01(w.dbg.flags&bit == bit))), nil
		}),
		// DebugDrawOnTop mode: 0 depth-tested, 1 always on top, 2 x-ray.
		"debugdrawontop": n(func(a []value.Value) (value.Value, error) {
			m := argI(a, 0, 1)
			if m < 0 || m > 2 {
				m = 1
			}
			w.debugState().mode = m
			return z()
		}),
		// DebugDrawNavMesh [nav [, on]]: nav 0 draws every baked mesh.
		"debugdrawnavmesh": n(func(a []value.Value) (value.Value, error) {
			w.debugState().navOnly = argI(a, 0, 0)
			w.setDebugFlags(dbgNav, argI(a, 1, 1) != 0)
			return z()
		}),
		// DebugRayTime seconds: how long picks stay drawn.
		"debugraytime": n(func(a []value.Value) (value.Value, error) {
			t := float32(argN(a, 0, dbgDefaultRayLife))
			if t < 0 {
				t = 0
			}
			w.debugState().rayLife = t
			return z()
		}),
		// SetPhysicsDebug flags: 1 shapes, 2 contacts, 4 velocity, 8 joints.
		"setphysicsdebug": n(func(a []value.Value) (value.Value, error) {
			w.setPhysicsDebugMask(uint32(argI(a, 0, int(dbgPhysicsMask))))
			return z()
		}),
		"getphysicsdebug": n(func(a []value.Value) (value.Value, error) {
			if w.dbg == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(w.dbg.flags & dbgPhysicsMask)), nil
		}),
		// DebugLine x1,y1,z1, x2,y2,z2 [, r,g,b [, seconds]]
		"debugline": n(func(a []value.Value) (value.Value, error) {
			p, q, c := dbgBlitz(a, 0), dbgBlitz(a, 3), dbgArgColor(a, 6)
			w.debugUserShape(dbgSeconds(a, 9), func(b *lineBatch) { b.line(p, q, c) })
			return z()
		}),
		// DebugArrow x1,y1,z1, x2,y2,z2 [, r,g,b [, seconds]]
		"debugarrow": n(func(a []value.Value) (value.Value, error) {
			p, q, c := dbgBlitz(a, 0), dbgBlitz(a, 3), dbgArgColor(a, 6)
			w.debugUserShape(dbgSeconds(a, 9), func(b *lineBatch) { b.arrow(p, q, c) })
			return z()
		}),
		// DebugSphere x,y,z, radius [, r,g,b [, seconds]]
		"debugsphere": n(func(a []value.Value) (value.Value, error) {
			p, r, c := dbgBlitz(a, 0), float32(argN(a, 3, 0.5)), dbgArgColor(a, 4)
			w.debugUserShape(dbgSeconds(a, 7), func(b *lineBatch) { b.wireSphere(identityXform(p), r, dbgCircleSegs, c) })
			return z()
		}),
		// DebugBox x,y,z, hx,hy,hz [, r,g,b [, seconds]]
		"debugbox": n(func(a []value.Value) (value.Value, error) {
			p, c := dbgBlitz(a, 0), dbgArgColor(a, 6)
			hx, hy, hz := float32(argN(a, 3, 0.5)), float32(argN(a, 4, 0.5)), float32(argN(a, 5, 0.5))
			w.debugUserShape(dbgSeconds(a, 9), func(b *lineBatch) { b.wireBox(identityXform(p), hx, hy, hz, c) })
			return z()
		}),
		// DebugClear: drop the script's queued lines, rays and sounds.
		"debugclear": n(func(a []value.Value) (value.Value, error) {
			if s := w.dbg; s != nil {
				s.user, s.rays, s.sounds, s.contacts = s.user[:0], s.rays[:0], s.sounds[:0], s.contacts[:0]
			}
			return z()
		}),
		// DebugDrawLines(): lines drawn last frame.
		"debugdrawlines": n(func(a []value.Value) (value.Value, error) {
			if w.dbg == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(w.dbg.lastLines)), nil
		}),
	}
}
