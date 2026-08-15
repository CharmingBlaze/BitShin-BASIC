package runtime

import (
	"fmt"

	"github.com/SanderMertens/flecs-go"

	"bitshinbasic/internal/value"
)

type ecsHost struct {
	world   *flecs.World
	comps   map[string]flecs.Component
	queries map[int]*ecsQuery
	nextQ   int
	freeQ   []int
	stepped bool
}

type ecsQuery struct {
	q    *flecs.Query
	ents []flecs.Entity
}

func (w *World) ecsEnsure() *flecs.World {
	if w.ecs.world == nil {
		w.ecs.world = flecs.NewWorld()
		w.ecs.comps = map[string]flecs.Component{}
		w.ecs.queries = map[int]*ecsQuery{}
		w.ecs.nextQ = 1
	}
	return w.ecs.world
}

func (w *World) ecsProgress(dt float64) {
	if w.ecs.world == nil || w.ecs.stepped {
		return
	}
	w.ecs.stepped = true
	if dt <= 0 {
		dt = w.delta
	}
	if dt <= 0 {
		dt = 1.0 / 60.0
	}
	w.ecs.world.Progress(float32(dt))
	w.ecsIntegrate(float32(dt))
}

func (w *World) ecsIntegrate(dt float32) {
	if w.ecs.world == nil || dt <= 0 {
		return
	}
	pos := w.ecs.comps["Position"]
	vel := w.ecs.comps["Velocity"]
	if pos == 0 || vel == 0 {
		return
	}
	q := w.ecs.world.Query("Position, Velocity")
	if q == nil {
		return
	}
	for _, e := range q.Each() {
		p, okp := w.ecs.world.Get(e, pos)
		v, okv := w.ecs.world.Get(e, vel)
		if !okp || !okv {
			continue
		}
		p.X += v.X * dt
		p.Y += v.Y * dt
		p.Z += v.Z * dt
		w.ecs.world.Set(e, pos, p)
	}
}

func (w *World) ecsComp(name string) flecs.Component {
	world := w.ecsEnsure()
	if name == "" {
		return 0
	}
	if c, ok := w.ecs.comps[name]; ok {
		return c
	}
	c := world.Component(name)
	w.ecs.comps[name] = c
	return c
}

func (w *World) ecsCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		"ecsworld": n(func(a []value.Value) (value.Value, error) {
			w.ecsEnsure()
			return value.Num(1), nil
		}),
		"ecsentity": n(func(a []value.Value) (value.Value, error) {
			e := w.ecsEnsure().Entity(argS(a, 0))
			return value.Num(float64(e)), nil
		}),
		"ecscomponent": n(func(a []value.Value) (value.Value, error) {
			name := argS(a, 0)
			if name == "" {
				return value.Value{}, fmt.Errorf("EcsComponent: name required")
			}
			return value.Num(float64(w.ecsComp(name))), nil
		}),
		"ecsset": n(func(a []value.Value) (value.Value, error) {
			world := w.ecsEnsure()
			e := flecs.Entity(argI(a, 0, 0))
			c := w.ecsCompID(a, 1)
			if e == 0 || c == 0 {
				return z()
			}
			v := flecs.Value{
				X: float32(argN(a, 2, 0)),
				Y: float32(argN(a, 3, 0)),
				Z: float32(argN(a, 4, 0)),
				W: float32(argN(a, 5, 0)),
			}
			if len(a) > 2 && a[2].Kind == value.KindStr {
				v.S = a[2].String()
			} else if len(a) > 6 {
				v.S = argS(a, 6)
			}
			world.Set(e, c, v)
			return z()
		}),
		"ecsget": n(func(a []value.Value) (value.Value, error) {
			v, ok := w.ecsEnsure().Get(flecs.Entity(argI(a, 0, 0)), w.ecsCompID(a, 1))
			if !ok {
				return value.Num(0), nil
			}
			return value.Num(float64(v.X)), nil
		}),
		"ecsgetx": n(func(a []value.Value) (value.Value, error) {
			v, _ := w.ecsEnsure().Get(flecs.Entity(argI(a, 0, 0)), w.ecsCompID(a, 1))
			return value.Num(float64(v.X)), nil
		}),
		"ecsgety": n(func(a []value.Value) (value.Value, error) {
			v, _ := w.ecsEnsure().Get(flecs.Entity(argI(a, 0, 0)), w.ecsCompID(a, 1))
			return value.Num(float64(v.Y)), nil
		}),
		"ecsgetz": n(func(a []value.Value) (value.Value, error) {
			v, _ := w.ecsEnsure().Get(flecs.Entity(argI(a, 0, 0)), w.ecsCompID(a, 1))
			return value.Num(float64(v.Z)), nil
		}),
		"ecsgetw": n(func(a []value.Value) (value.Value, error) {
			v, _ := w.ecsEnsure().Get(flecs.Entity(argI(a, 0, 0)), w.ecsCompID(a, 1))
			return value.Num(float64(v.W)), nil
		}),
		"ecsgets": n(func(a []value.Value) (value.Value, error) {
			v, _ := w.ecsEnsure().Get(flecs.Entity(argI(a, 0, 0)), w.ecsCompID(a, 1))
			return value.Str(v.S), nil
		}),
		"ecshas": n(func(a []value.Value) (value.Value, error) {
			if w.ecsEnsure().Has(flecs.Entity(argI(a, 0, 0)), w.ecsCompID(a, 1)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"ecsadd": n(func(a []value.Value) (value.Value, error) {
			w.ecsEnsure().Add(flecs.Entity(argI(a, 0, 0)), w.ecsCompID(a, 1))
			return z()
		}),
		"ecsremove": n(func(a []value.Value) (value.Value, error) {
			w.ecsEnsure().Remove(flecs.Entity(argI(a, 0, 0)), w.ecsCompID(a, 1))
			return z()
		}),
		"ecsdelete": n(func(a []value.Value) (value.Value, error) {
			w.ecsEnsure().Delete(flecs.Entity(argI(a, 0, 0)))
			return z()
		}),
		"ecsalive": n(func(a []value.Value) (value.Value, error) {
			if w.ecsEnsure().Alive(flecs.Entity(argI(a, 0, 0))) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"ecsvalid": n(func(a []value.Value) (value.Value, error) {
			if w.ecsEnsure().Valid(flecs.Entity(argI(a, 0, 0))) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"ecslookup": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.ecsEnsure().Lookup(argS(a, 0)))), nil
		}),
		"ecsname": n(func(a []value.Value) (value.Value, error) {
			e := flecs.Entity(argI(a, 0, 0))
			if len(a) > 1 {
				w.ecsEnsure().SetName(e, argS(a, 1))
			}
			return value.Str(w.ecsEnsure().Name(e)), nil
		}),
		"ecsquery": n(func(a []value.Value) (value.Value, error) {
			expr := argS(a, 0)
			q := w.ecsEnsure().Query(expr)
			if q == nil {
				return value.Value{}, fmt.Errorf("EcsQuery: invalid expression %q", expr)
			}
			id := w.takeHandle(&w.ecs.freeQ, &w.ecs.nextQ)
			if w.ecs.queries == nil {
				w.ecs.queries = map[int]*ecsQuery{}
			}
			ents := q.Each()
			w.ecs.queries[id] = &ecsQuery{q: q, ents: ents}
			return value.Num(float64(id)), nil
		}),
		"ecsquerycount": n(func(a []value.Value) (value.Value, error) {
			q := w.ecs.queries[argI(a, 0, 0)]
			if q == nil {
				return value.Num(0), nil
			}
			q.ents = q.q.Each()
			return value.Num(float64(len(q.ents))), nil
		}),
		"ecsqueryentity": n(func(a []value.Value) (value.Value, error) {
			q := w.ecs.queries[argI(a, 0, 0)]
			i := argI(a, 1, 0)
			if q == nil || i < 0 || i >= len(q.ents) {
				return value.Num(0), nil
			}
			return value.Num(float64(q.ents[i])), nil
		}),
		"ecsprogress": n(func(a []value.Value) (value.Value, error) {
			dt := argN(a, 0, w.delta)
			w.ecs.stepped = false
			w.ecsProgress(dt)
			return z()
		}),
		"ecscount": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.ecsEnsure().Count(w.ecsCompID(a, 0)))), nil
		}),
		"ecsparent": n(func(a []value.Value) (value.Value, error) {
			w.ecsEnsure().SetParent(flecs.Entity(argI(a, 0, 0)), flecs.Entity(argI(a, 1, 0)))
			return z()
		}),
		"ecsgetparent": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.ecsEnsure().Parent(flecs.Entity(argI(a, 0, 0))))), nil
		}),
		"ecsversion": n(func(a []value.Value) (value.Value, error) {
			return value.Str(flecs.Version), nil
		}),
	}
}

func (w *World) ecsCompID(a []value.Value, i int) flecs.Component {
	if i >= len(a) {
		return 0
	}
	if a[i].Kind == value.KindStr {
		return w.ecsComp(a[i].String())
	}
	id := flecs.Component(a[i].Int())
	if id != 0 {
		return id
	}
	return 0
}
