package runtime

import (
	"fmt"
	"sort"
	"strings"

	"moonbasic/internal/value"
)

// A planner is a small GOAP search. Facts are comma-separated words.
// An action lists the facts it needs and the facts it adds.
type planAction struct {
	name string
	cost float32
	pre  []string
	add  []string
}

type planner struct {
	actions []planAction
	plan    []string
}

func (w *World) goapCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), _ func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"createplanner": n(func(a []value.Value) (value.Value, error) {
			if w.planners == nil {
				w.planners = map[int]*planner{}
			}
			id := w.takeHandle(&w.freePlan, &w.nextPlan)
			w.planners[id] = &planner{}
			return value.Num(float64(id)), nil
		}),
		"addplanaction": n(func(a []value.Value) (value.Value, error) {
			p := w.planners[argI(a, 0, 0)]
			if p == nil {
				return value.Value{}, fmt.Errorf("AddPlanAction: no planner")
			}
			cost := float32(argN(a, 2, 1))
			if cost <= 0 {
				cost = 1
			}
			p.actions = append(p.actions, planAction{
				name: argS(a, 1),
				cost: cost,
				pre:  factList(argS(a, 3)),
				add:  factList(argS(a, 4)),
			})
			return value.Num(float64(len(p.actions))), nil
		}),
		"plan": n(func(a []value.Value) (value.Value, error) {
			p := w.planners[argI(a, 0, 0)]
			if p == nil {
				return value.Str(""), nil
			}
			steps := planGOAP(p.actions, factList(argS(a, 1)), factList(argS(a, 2)))
			p.plan = steps
			if len(steps) == 0 {
				return value.Str(""), nil
			}
			return value.Str(steps[0]), nil
		}),
		"planlength": n(func(a []value.Value) (value.Value, error) {
			p := w.planners[argI(a, 0, 0)]
			if p == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(len(p.plan))), nil
		}),
		"planstep": n(func(a []value.Value) (value.Value, error) {
			p := w.planners[argI(a, 0, 0)]
			i := argI(a, 1, 0)
			if p == nil || i < 0 || i >= len(p.plan) {
				return value.Str(""), nil
			}
			return value.Str(p.plan[i]), nil
		}),
	}
}

func factList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func hasFacts(have, need []string) bool {
	for _, n := range need {
		ok := false
		for _, h := range have {
			if h == n {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

func planGOAP(actions []planAction, state, goal []string) []string {
	if hasFacts(state, goal) {
		return nil
	}
	type node struct {
		facts []string
		steps []string
		cost  float32
	}
	best := map[string]float32{}
	key := func(f []string) string {
		s := append([]string(nil), f...)
		sort.Strings(s)
		return strings.Join(s, ",")
	}
	q := []node{{facts: append([]string(nil), state...)}}
	best[key(state)] = 0
	var found []string
	foundCost := float32(1e9)
	guard := 0
	for len(q) > 0 && guard < 4000 {
		guard++
		cur := q[0]
		q = q[1:]
		if cur.cost >= foundCost {
			continue
		}
		for _, act := range actions {
			if !hasFacts(cur.facts, act.pre) {
				continue
			}
			next := append([]string(nil), cur.facts...)
			for _, add := range act.add {
				if !hasFacts(next, []string{add}) {
					next = append(next, add)
				}
			}
			cost := cur.cost + act.cost
			k := key(next)
			if old, ok := best[k]; ok && old <= cost {
				continue
			}
			best[k] = cost
			steps := append(append([]string(nil), cur.steps...), act.name)
			if hasFacts(next, goal) {
				if cost < foundCost {
					foundCost = cost
					found = steps
				}
				continue
			}
			q = append(q, node{facts: next, steps: steps, cost: cost})
		}
	}
	return found
}
