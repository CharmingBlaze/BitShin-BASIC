package runtime

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/AllenDang/cimgui-go/imgui"

	"bitshinbasic/internal/parse"
	"bitshinbasic/internal/value"
)

type animGraph struct {
	ent    int
	states map[string]int
	cur    string
}

type appState struct {
	actions   map[string][]int
	tables    map[string][]map[string]string
	graphs    map[int]*animGraph
	nextGraph int
	dropped   []string
	pick      string
	ikA       float64
	ikB       float64
}

func (w *World) extState() *appState {
	if w.ext == nil {
		w.ext = &appState{
			actions: map[string][]int{},
			tables:  map[string][]map[string]string{},
			graphs:  map[int]*animGraph{},
		}
	}
	return w.ext
}

func (w *World) SetScript(path string) { w.scriptPath = path }

func (w *World) appCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"graphicsapp": n(func(a []value.Value) (value.Value, error) {
			v, err := w.graphics3D(argI(a, 0, 960), argI(a, 1, 640), argI(a, 2, 0), argI(a, 3, 2))
			if err != nil {
				return v, err
			}
			w.spawnCamera(0)
			return v, nil
		}),
		"readtext": n(func(a []value.Value) (value.Value, error) {
			b, err := os.ReadFile(w.resolve(argS(a, 0)))
			if err != nil {
				return value.Str(""), err
			}
			return value.Str(string(b)), nil
		}),
		"writetext": n(func(a []value.Value) (value.Value, error) {
			path := w.resolve(argS(a, 0))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return value.Num(0), err
			}
			if err := os.WriteFile(path, []byte(argS(a, 1)), 0o644); err != nil {
				return value.Num(0), err
			}
			return value.Num(1), nil
		}),
		"listdir": n(func(a []value.Value) (value.Value, error) {
			entries, err := os.ReadDir(w.resolve(argS(a, 0)))
			if err != nil {
				return value.Str(""), err
			}
			names := make([]string, 0, len(entries))
			for _, e := range entries {
				name := e.Name()
				if e.IsDir() {
					name += "/"
				}
				names = append(names, name)
			}
			return value.Str(strings.Join(names, "\n")), nil
		}),
		"droppedfile": n(func(a []value.Value) (value.Value, error) {
			st := w.extState()
			if len(st.dropped) == 0 {
				return value.Str(""), nil
			}
			s := st.dropped[0]
			st.dropped = st.dropped[1:]
			return value.Str(s), nil
		}),
		"dropcount": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(len(w.extState().dropped))), nil
		}),
		"bindaction": n(func(a []value.Value) (value.Value, error) {
			name := strings.ToLower(argS(a, 0))
			key := argI(a, 1, 0)
			st := w.extState()
			st.actions[name] = append(st.actions[name], key)
			return z()
		}),
		"unbindaction": n(func(a []value.Value) (value.Value, error) {
			delete(w.extState().actions, strings.ToLower(argS(a, 0)))
			return z()
		}),
		"actiondown": n(func(a []value.Value) (value.Value, error) {
			if w.actionDown(argS(a, 0)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"actionhit": n(func(a []value.Value) (value.Value, error) {
			name := strings.ToLower(argS(a, 0))
			for _, k := range w.extState().actions[name] {
				if w.hits[k] {
					return value.Num(1), nil
				}
			}
			return value.Num(0), nil
		}),
		"savebinds": n(func(a []value.Value) (value.Value, error) {
			b, err := json.MarshalIndent(w.extState().actions, "", "  ")
			if err != nil {
				return value.Num(0), err
			}
			path := w.resolve(argS(a, 0))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return value.Num(0), err
			}
			return value.Num(1), os.WriteFile(path, b, 0o644)
		}),
		"loadbinds": n(func(a []value.Value) (value.Value, error) {
			b, err := os.ReadFile(w.resolve(argS(a, 0)))
			if err != nil {
				return value.Num(0), err
			}
			m := map[string][]int{}
			if err := json.Unmarshal(b, &m); err != nil {
				return value.Num(0), err
			}
			w.extState().actions = m
			return value.Num(1), nil
		}),
		"tablecreate": n(func(a []value.Value) (value.Value, error) {
			w.extState().tables[strings.ToLower(argS(a, 0))] = []map[string]string{}
			return z()
		}),
		"tableinsert": n(func(a []value.Value) (value.Value, error) {
			name := strings.ToLower(argS(a, 0))
			row := map[string]string{}
			if err := json.Unmarshal([]byte(argS(a, 1)), &row); err != nil {
				return value.Num(0), err
			}
			st := w.extState()
			st.tables[name] = append(st.tables[name], row)
			return value.Num(float64(len(st.tables[name]) - 1)), nil
		}),
		"tableget": n(func(a []value.Value) (value.Value, error) {
			rows := w.extState().tables[strings.ToLower(argS(a, 0))]
			i := argI(a, 1, 0)
			if i < 0 || i >= len(rows) {
				return value.Str(""), nil
			}
			return value.Str(rows[i][argS(a, 2)]), nil
		}),
		"tablecount": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(len(w.extState().tables[strings.ToLower(argS(a, 0))]))), nil
		}),
		"tablesave": n(func(a []value.Value) (value.Value, error) {
			rows := w.extState().tables[strings.ToLower(argS(a, 0))]
			b, err := json.MarshalIndent(rows, "", "  ")
			if err != nil {
				return value.Num(0), err
			}
			path := w.resolve(argS(a, 1))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return value.Num(0), err
			}
			return value.Num(1), os.WriteFile(path, b, 0o644)
		}),
		"tableload": n(func(a []value.Value) (value.Value, error) {
			b, err := os.ReadFile(w.resolve(argS(a, 1)))
			if err != nil {
				return value.Num(0), err
			}
			var rows []map[string]string
			if err := json.Unmarshal(b, &rows); err != nil {
				return value.Num(0), err
			}
			w.extState().tables[strings.ToLower(argS(a, 0))] = rows
			return value.Num(float64(len(rows))), nil
		}),
		"saveslot": n(func(a []value.Value) (value.Value, error) {
			path := w.slotPath(argI(a, 0, 0))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return value.Num(0), err
			}
			return value.Num(1), os.WriteFile(path, []byte(argS(a, 1)), 0o644)
		}),
		"loadslot": n(func(a []value.Value) (value.Value, error) {
			b, err := os.ReadFile(w.slotPath(argI(a, 0, 0)))
			if err != nil {
				return value.Str(""), nil
			}
			return value.Str(string(b)), nil
		}),
		"saveprefab": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 1, 0))
			if err != nil {
				return value.Value{}, err
			}
			pos := e.node.GetNode().Position()
			x, y, z := fromG3N(pos.X, pos.Y, pos.Z)
			body := map[string]any{
				"kind": e.kind, "src": e.src, "name": e.name,
				"x": x, "y": y, "z": z,
			}
			raw, err := json.MarshalIndent(body, "", "  ")
			if err != nil {
				return value.Num(0), err
			}
			path := filepath.Join(w.base, "prefabs", argS(a, 0)+".json")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return value.Num(0), err
			}
			return value.Num(1), os.WriteFile(path, raw, 0o644)
		}),
		"spawnprefab": need(func(a []value.Value) (value.Value, error) {
			path := filepath.Join(w.base, "prefabs", argS(a, 0)+".json")
			b, err := os.ReadFile(path)
			if err != nil {
				return value.Num(0), err
			}
			var body struct {
				Kind string  `json:"kind"`
				Src  string  `json:"src"`
				Name string  `json:"name"`
				X    float32 `json:"x"`
				Y    float32 `json:"y"`
				Z    float32 `json:"z"`
			}
			if err := json.Unmarshal(b, &body); err != nil {
				return value.Num(0), err
			}
			var id int
			if body.Src != "" {
				id, err = w.loadMeshFile(body.Src, 0)
				if err != nil {
					return value.Num(0), err
				}
			} else {
				id = w.createCubeMesh(nil)
			}
			e := w.ents[id]
			if e != nil {
				e.kind = body.Kind
				e.name = body.Name
				e.src = body.Src
				gx, gy, gz := toG3N(body.X, body.Y, body.Z)
				e.node.GetNode().SetPosition(gx, gy, gz)
			}
			return value.Num(float64(id)), nil
		}),
		"createanimgraph": need(func(a []value.Value) (value.Value, error) {
			st := w.extState()
			st.nextGraph++
			st.graphs[st.nextGraph] = &animGraph{ent: argI(a, 0, 0), states: map[string]int{}}
			return value.Num(float64(st.nextGraph)), nil
		}),
		"animstate": need(func(a []value.Value) (value.Value, error) {
			g := w.extState().graphs[argI(a, 0, 0)]
			if g == nil {
				return value.Num(0), fmt.Errorf("AnimState: no graph")
			}
			g.states[strings.ToLower(argS(a, 1))] = argI(a, 2, 0)
			if g.cur == "" {
				g.cur = strings.ToLower(argS(a, 1))
			}
			return z()
		}),
		"animgo": need(func(a []value.Value) (value.Value, error) {
			g := w.extState().graphs[argI(a, 0, 0)]
			if g == nil {
				return value.Num(0), fmt.Errorf("AnimGo: no graph")
			}
			name := strings.ToLower(argS(a, 1))
			seq, ok := g.states[name]
			if !ok {
				return value.Num(0), fmt.Errorf("AnimGo: no state %s", name)
			}
			g.cur = name
			e := w.ents[g.ent]
			if e == nil || e.anim == nil || seq < 0 || seq >= len(e.anim.clips) {
				return value.Num(0), nil
			}
			e.anim.seq = seq
			e.anim.mode = 1
			e.anim.speed = 1
			e.anim.dir = 1
			e.anim.time = e.anim.first[seq]
			e.anim.clips[seq].Reset()
			e.anim.clips[seq].SetPaused(false)
			return value.Num(1), nil
		}),
		"animupdate": need(func(a []value.Value) (value.Value, error) {
			g := w.extState().graphs[argI(a, 0, 0)]
			if g == nil {
				return value.Str(""), nil
			}
			return value.Str(g.cur), nil
		}),
		"reloadscripts": n(func(a []value.Value) (value.Value, error) {
			path := w.scriptPath
			if argS(a, 0) != "" {
				path = w.resolve(argS(a, 0))
			}
			if path == "" || w.runner == nil {
				return value.Num(0), fmt.Errorf("ReloadScripts: no script")
			}
			prog, err := parse.ParseFile(path)
			if err != nil {
				return value.Num(0), err
			}
			prog, err = parse.ExpandIncludes(prog, filepath.Dir(path))
			if err != nil {
				return value.Num(0), err
			}
			w.runner.Reload(prog)
			return value.Num(1), nil
		}),
		"solvetwobone": n(func(a []value.Value) (value.Value, error) {
			ax, ay := argN(a, 0, 0), argN(a, 1, 0)
			tx, ty := argN(a, 2, 0), argN(a, 3, 0)
			l1, l2 := argN(a, 4, 1), argN(a, 5, 1)
			if l1 < 0.0001 {
				l1 = 0.0001
			}
			if l2 < 0.0001 {
				l2 = 0.0001
			}
			dx, dy := tx-ax, ty-ay
			dist := math.Hypot(dx, dy)
			max := l1 + l2
			min := math.Abs(l1 - l2)
			if dist > max {
				dist = max
			}
			if dist < min {
				dist = min
			}
			cosA := (l1*l1 + dist*dist - l2*l2) / (2 * l1 * dist)
			cosB := (l1*l1 + l2*l2 - dist*dist) / (2 * l1 * l2)
			cosA = math.Max(-1, math.Min(1, cosA))
			cosB = math.Max(-1, math.Min(1, cosB))
			base := math.Atan2(dy, dx)
			a1 := (base - math.Acos(cosA)) * 180 / math.Pi
			a2 := (math.Pi - math.Acos(cosB)) * 180 / math.Pi
			st := w.extState()
			st.ikA, st.ikB = a1, a2
			return value.Num(a1), nil
		}),
		"ikbend": n(func(a []value.Value) (value.Value, error) {
			return value.Num(w.extState().ikB), nil
		}),
		"guipickfile": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Str(""), err
			}
			dir := argS(a, 0)
			if dir == "" {
				dir = "."
			}
			entries, err := os.ReadDir(w.resolve(dir))
			if err != nil {
				imgui.Text(err.Error())
				return value.Str(""), nil
			}
			st := w.extState()
			if imgui.BeginChildStr("bitshin-files") {
				for _, e := range entries {
					label := e.Name()
					if e.IsDir() {
						label += "/"
					}
					if imgui.SelectableBool(label) {
						st.pick = filepath.Join(dir, e.Name())
					}
				}
				imgui.EndChild()
			}
			return value.Str(st.pick), nil
		}),
	}
}

func (w *World) actionDown(name string) bool {
	for _, k := range w.extState().actions[strings.ToLower(name)] {
		if w.keys[k] {
			return true
		}
	}
	return false
}

func (w *World) slotPath(slot int) string {
	return filepath.Join(w.base, "saves", fmt.Sprintf("slot%d.txt", slot))
}
