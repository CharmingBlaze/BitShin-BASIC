package runtime

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/value"
)

// playState is the outdoor/indoor game layer: time of day, rooms, doors,
// inventory, dialogue, and a three-state actor. It is not a second renderer.
type playState struct {
	hour     float32
	sunScale float32
	skyBand  string
	outAmb   math32.Color
	ambSet   bool

	nextRoom int
	room     int
	rooms    map[int]*playRoom
	member   map[int]int
	links    map[int][]int

	doors    map[int]*playDoor
	items    map[int]string
	held     map[int]bool
	inv      []string
	lines    map[int][]string
	talkEnt  int
	talkLine int

	actors map[int]*playActor
	player int

	clock    float64
	lastX    float32
	lastZ    float32
	havePos  bool
	stepDist float32
	lastStep float64
	echoAt   float64
	echoSnd  int
	echoVol  float64

	visited map[string]bool
}

type playRoom struct {
	amb    math32.Color
	hasAmb bool
	step   int
	reverb float32
}

type playDoor struct {
	a, b  int
	sound int
	open  bool
	base  float32
	yaw   float32
}

type playActor struct {
	aggro float32
	speed float32
	state int // 0 idle, 1 chase, 2 attack
}

type gameSave struct {
	Hour      float32    `json:"hour"`
	Player    [3]float32 `json:"player"`
	PlayerID  int        `json:"playerID"`
	Room      int        `json:"room"`
	Inventory []string   `json:"inventory"`
	Doors     []saveDoor `json:"doors"`
	Visited   []string   `json:"visited"`
	TalkEnt   int        `json:"talkEnt"`
	TalkLine  int        `json:"talkLine"`
}

type saveDoor struct {
	ID   int  `json:"id"`
	Open bool `json:"open"`
}

func (w *World) ensurePlay() *playState {
	if w.play == nil {
		w.play = &playState{
			sunScale: 1,
			outAmb:   math32.Color{0.45, 0.48, 0.52},
			nextRoom: 1,
			rooms:    map[int]*playRoom{},
			member:   map[int]int{},
			links:    map[int][]int{},
			doors:    map[int]*playDoor{},
			items:    map[int]string{},
			held:     map[int]bool{},
			lines:    map[int][]string{},
			actors:   map[int]*playActor{},
			visited:  map[string]bool{},
		}
	}
	return w.play
}

func (w *World) playCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	_ = need
	m := w.scatterCommands(n, z)
	m["settimeofday"] = n(func(a []value.Value) (value.Value, error) {
		w.setPlayHour(float32(argN(a, 0, 12)))
		return z()
	})
	m["gettimeofday"] = n(func(a []value.Value) (value.Value, error) {
		return value.Num(float64(w.ensurePlay().hour)), nil
	})
	m["createroom"] = n(func(a []value.Value) (value.Value, error) {
		g := w.ensurePlay()
		id := g.nextRoom
		g.nextRoom++
		g.rooms[id] = &playRoom{}
		return value.Num(float64(id)), nil
	})
	m["setroom"] = n(func(a []value.Value) (value.Value, error) {
		w.assignRoom(argI(a, 0, 0), argI(a, 1, 0))
		return z()
	})
	m["roomadd"] = n(func(a []value.Value) (value.Value, error) {
		w.assignRoom(argI(a, 1, 0), argI(a, 0, 0))
		return z()
	})
	m["roomlink"] = n(func(a []value.Value) (value.Value, error) {
		w.linkRooms(argI(a, 0, 0), argI(a, 1, 0))
		w.applyRoomVis()
		return z()
	})
	m["roomambient"] = n(func(a []value.Value) (value.Value, error) {
		g := w.ensurePlay()
		r := g.rooms[argI(a, 0, 0)]
		if r == nil {
			return z()
		}
		c := rgb(argN(a, 1, 40), argN(a, 2, 40), argN(a, 3, 48))
		r.amb = *c
		r.hasAmb = true
		if g.room == argI(a, 0, 0) {
			w.applyRoomLight()
		}
		return z()
	})
	m["roomaudio"] = n(func(a []value.Value) (value.Value, error) {
		g := w.ensurePlay()
		r := g.rooms[argI(a, 0, 0)]
		if r == nil && argI(a, 0, 0) == 0 {
			r = &playRoom{}
			g.rooms[0] = r
		}
		if r == nil {
			return z()
		}
		r.step = argI(a, 1, 0)
		r.reverb = float32(argN(a, 2, 0))
		if r.reverb < 0 {
			r.reverb = 0
		}
		if r.reverb > 1 {
			r.reverb = 1
		}
		return z()
	})
	m["enterroom"] = n(func(a []value.Value) (value.Value, error) {
		w.enterRoom(argI(a, 0, 0))
		return z()
	})
	m["currentroom"] = n(func(a []value.Value) (value.Value, error) {
		if w.play == nil {
			return value.Num(0), nil
		}
		return value.Num(float64(w.play.room)), nil
	})
	m["createdoor"] = n(func(a []value.Value) (value.Value, error) {
		id := argI(a, 0, 0)
		g := w.ensurePlay()
		base := float32(0)
		if e := w.ents[id]; e != nil {
			base = e.yaw
		}
		g.doors[id] = &playDoor{
			a: argI(a, 1, 0), b: argI(a, 2, 0), sound: argI(a, 3, 0),
			base: base, yaw: base,
		}
		if g.doors[id].a != 0 || g.doors[id].b != 0 {
			w.linkRooms(g.doors[id].a, g.doors[id].b)
		}
		return value.Num(float64(id)), nil
	})
	m["use"] = n(func(a []value.Value) (value.Value, error) {
		id := argI(a, 0, w.pickID)
		if w.useEntity(id) {
			return value.Num(1), nil
		}
		return value.Num(0), nil
	})
	m["setitem"] = n(func(a []value.Value) (value.Value, error) {
		w.ensurePlay().items[argI(a, 0, 0)] = argS(a, 1)
		return z()
	})
	m["setdialogue"] = n(func(a []value.Value) (value.Value, error) {
		text := argS(a, 1)
		text = strings.ReplaceAll(text, "\r\n", "\n")
		parts := strings.FieldsFunc(text, func(r rune) bool { return r == '|' || r == '\n' })
		lines := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				lines = append(lines, p)
			}
		}
		w.ensurePlay().lines[argI(a, 0, 0)] = lines
		return z()
	})
	m["inventoryhas"] = n(func(a []value.Value) (value.Value, error) {
		if w.inventoryHas(argS(a, 0)) {
			return value.Num(1), nil
		}
		return value.Num(0), nil
	})
	m["inventorycount"] = n(func(a []value.Value) (value.Value, error) {
		if w.play == nil {
			return value.Num(0), nil
		}
		return value.Num(float64(len(w.play.inv))), nil
	})
	m["inventoryitem"] = n(func(a []value.Value) (value.Value, error) {
		g := w.play
		i := argI(a, 0, 0)
		if g == nil || i < 0 || i >= len(g.inv) {
			return value.Str(""), nil
		}
		return value.Str(g.inv[i]), nil
	})
	m["inventoryremove"] = n(func(a []value.Value) (value.Value, error) {
		w.inventoryRemove(argS(a, 0))
		return z()
	})
	m["dialogueon"] = n(func(a []value.Value) (value.Value, error) {
		g := w.play
		if g == nil || g.talkEnt == 0 {
			return value.Num(0), nil
		}
		return value.Num(1), nil
	})
	m["dialogueline"] = n(func(a []value.Value) (value.Value, error) {
		return value.Str(w.dialogueLine()), nil
	})
	m["dialogueadvance"] = n(func(a []value.Value) (value.Value, error) {
		w.advanceDialogue()
		return z()
	})
	m["savegame"] = n(func(a []value.Value) (value.Value, error) {
		if err := w.saveGame(argS(a, 0)); err != nil {
			return value.Num(0), err
		}
		return value.Num(1), nil
	})
	m["loadgame"] = n(func(a []value.Value) (value.Value, error) {
		if err := w.loadGame(argS(a, 0)); err != nil {
			return value.Num(0), err
		}
		return value.Num(1), nil
	})
	m["createactor"] = n(func(a []value.Value) (value.Value, error) {
		id := argI(a, 0, 0)
		aggro := float32(argN(a, 1, 8))
		speed := float32(argN(a, 2, 3.2))
		if aggro <= 0 {
			aggro = 8
		}
		if speed <= 0 {
			speed = 3.2
		}
		w.ensurePlay().actors[id] = &playActor{aggro: aggro, speed: speed}
		return value.Num(float64(id)), nil
	})
	m["setplayer"] = n(func(a []value.Value) (value.Value, error) {
		id := argI(a, 0, 0)
		w.ensurePlay().player = id
		w.armWorldBubble(id)
		return z()
	})
	m["actorstate"] = n(func(a []value.Value) (value.Value, error) {
		g := w.play
		if g == nil {
			return value.Num(0), nil
		}
		if ac := g.actors[argI(a, 0, 0)]; ac != nil {
			return value.Num(float64(ac.state)), nil
		}
		return value.Num(0), nil
	})
	return m
}

func (w *World) setPlayHour(hours float32) {
	if hours < 0 {
		hours = 0
	}
	hours = float32(math.Mod(float64(hours), 24))
	g := w.ensurePlay()
	g.hour = hours
	elev := float32(math.Sin(float64((hours - 6) / 24 * 2 * math.Pi)))
	pitch := elev * 58
	if pitch < 8 {
		pitch = 8
	}
	yaw := (hours - 6) * 15
	sun := elev
	if sun < 0.08 {
		sun = 0.08
	}
	if sun > 1 {
		sun = 1
	}
	g.sunScale = sun
	for _, e := range w.ents {
		if e == nil || e.lgtKind != 1 || e.node == nil {
			continue
		}
		w.setLightDir(e, float64(pitch), float64(yaw), 0)
		if e.lgtI != nil && g.room == 0 {
			e.lgtI.SetIntensity(sun)
		}
	}
	band := "day"
	if hours < 5.5 || hours >= 19.5 {
		band = "night"
	} else if hours < 8 || hours >= 17 {
		band = "sunset"
	}
	if band != g.skyBand {
		g.skyBand = band
		w.paintTimeSky(band)
	}
	if g.room == 0 {
		w.applyRoomLight()
	}
}

func (w *World) paintTimeSky(band string) {
	g := w.ensurePlay()
	switch band {
	case "night":
		w.skyTop = math32.Color{0.03, 0.04, 0.09}
		w.skyBot = math32.Color{0.08, 0.09, 0.14}
		w.fogRGB = math32.Color{0.05, 0.06, 0.09}
		g.outAmb = math32.Color{0.08, 0.09, 0.14}
	case "sunset":
		w.skyTop = math32.Color{177.0 / 255, 174.0 / 255, 119.0 / 255}
		w.skyBot = math32.Color{234.0 / 255, 125.0 / 255, 125.0 / 255}
		w.fogRGB = math32.Color{85.0 / 255, 97.0 / 255, 120.0 / 255}
		g.outAmb = math32.Color{0.55, 0.32, 0.22}
	default:
		w.skyTop = math32.Color{0.525, 0.735, 0.84}
		w.skyBot = math32.Color{0.9, 0.9, 0.95}
		w.fogRGB = math32.Color{0.5, 0.6, 0.7}
		g.outAmb = math32.Color{0.45, 0.48, 0.52}
	}
	g.ambSet = true
	if w.app == nil {
		return
	}
	sx, sy, sz := w.proceduralSkySun()
	id, err := w.createSkyBoxFromFaces(generateSkyFaces(w.skyTop, w.skyBot, sx, sy, sz))
	if err == nil {
		w.skyProc = true
		w.setSkyBox(id)
	}
}

func (w *World) assignRoom(ent, room int) {
	g := w.ensurePlay()
	if room == 0 {
		delete(g.member, ent)
	} else {
		if g.rooms[room] == nil {
			g.rooms[room] = &playRoom{}
		}
		g.member[ent] = room
	}
	w.applyRoomVis()
}

func (w *World) linkRooms(a, b int) {
	if a == b {
		return
	}
	g := w.ensurePlay()
	add := func(from, to int) {
		for _, l := range g.links[from] {
			if l == to {
				return
			}
		}
		g.links[from] = append(g.links[from], to)
	}
	add(a, b)
	add(b, a)
}

func (w *World) enterRoom(id int) {
	g := w.ensurePlay()
	if id != 0 && g.rooms[id] == nil {
		g.rooms[id] = &playRoom{}
	}
	g.room = id
	w.applyRoomVis()
	w.applyRoomLight()
}

func (w *World) roomShown(id int) bool {
	g := w.play
	if g == nil || g.room == 0 {
		return id == 0
	}
	if id == g.room {
		return true
	}
	for _, l := range g.links[g.room] {
		if l == id {
			return true
		}
	}
	return false
}

func (w *World) applyRoomVis() {
	g := w.play
	if g == nil {
		return
	}
	for ent, room := range g.member {
		e := w.ents[ent]
		if e == nil || e.node == nil {
			continue
		}
		e.node.GetNode().SetVisible(w.roomShown(room))
	}
}

func (w *World) applyRoomLight() {
	g := w.play
	if g == nil || w.ambient == nil {
		return
	}
	col := g.outAmb
	inten := float32(1)
	if g.room != 0 {
		if r := g.rooms[g.room]; r != nil && r.hasAmb {
			col = r.amb
			inten = 0.7
		}
	}
	w.ambient.SetColor(&col)
	w.ambient.SetIntensity(inten)
	sun := g.sunScale
	if sun <= 0 {
		sun = 1
	}
	if g.room != 0 {
		sun *= 0.22
	}
	for _, e := range w.ents {
		if e != nil && e.lgtKind == 1 && e.lgtI != nil {
			e.lgtI.SetIntensity(sun)
		}
	}
}

func (w *World) useEntity(id int) bool {
	if id == 0 {
		return false
	}
	g := w.ensurePlay()
	if d := g.doors[id]; d != nil {
		d.open = !d.open
		if d.sound != 0 {
			_, _ = w.startClip(d.sound, false)
		}
		return true
	}
	if name := g.items[id]; name != "" && !g.held[id] {
		g.inv = append(g.inv, name)
		g.held[id] = true
		if e := w.ents[id]; e != nil && e.node != nil {
			e.node.GetNode().SetVisible(false)
		}
		return true
	}
	if lines := g.lines[id]; len(lines) > 0 {
		g.talkEnt = id
		g.talkLine = 0
		return true
	}
	if ac := g.actors[id]; ac != nil {
		ac.state = 1
		return true
	}
	return false
}

func (w *World) inventoryHas(name string) bool {
	g := w.play
	if g == nil || name == "" {
		return false
	}
	for _, s := range g.inv {
		if strings.EqualFold(s, name) {
			return true
		}
	}
	return false
}

func (w *World) inventoryRemove(name string) {
	g := w.play
	if g == nil || name == "" {
		return
	}
	src := append([]string(nil), g.inv...)
	out := make([]string, 0, len(src))
	dropped := false
	for _, s := range src {
		if !dropped && strings.EqualFold(s, name) {
			dropped = true
			continue
		}
		out = append(out, s)
	}
	g.inv = out
}

func (w *World) dialogueLine() string {
	g := w.play
	if g == nil || g.talkEnt == 0 {
		return ""
	}
	lines := g.lines[g.talkEnt]
	if g.talkLine < 0 || g.talkLine >= len(lines) {
		return ""
	}
	return lines[g.talkLine]
}

func (w *World) advanceDialogue() {
	g := w.play
	if g == nil || g.talkEnt == 0 {
		return
	}
	g.talkLine++
	if g.talkLine >= len(g.lines[g.talkEnt]) {
		g.talkEnt = 0
		g.talkLine = 0
	}
}

func (w *World) scriptPos(id int) (x, y, z float32, ok bool) {
	e := w.ents[id]
	if e == nil || e.node == nil {
		return 0, 0, 0, false
	}
	p := e.node.GetNode().Position()
	x, y, z = fromG3N(p.X, p.Y, p.Z)
	return x, y, z, true
}

func (w *World) setScriptPos(id int, x, y, z float32) {
	e := w.ents[id]
	if e == nil || e.node == nil {
		return
	}
	gx, gy, gz := toG3N(x, y, z)
	e.node.GetNode().SetPosition(gx, gy, gz)
}

func (w *World) tickPlay(dt float32) {
	g := w.play
	if g == nil {
		return
	}
	if dt <= 0 {
		dt = 1.0 / 60.0
	}
	g.clock += float64(dt)
	for id, d := range g.doors {
		e := w.ents[id]
		if e == nil || e.node == nil {
			continue
		}
		target := d.base
		if d.open {
			target = d.base + 100
		}
		delta := target - d.yaw
		step := 140 * dt
		if delta > step {
			delta = step
		} else if delta < -step {
			delta = -step
		}
		d.yaw += delta
		e.yaw = d.yaw
		w.applyRot(e)
	}
	if g.echoAt > 0 && g.clock >= g.echoAt && g.echoSnd != 0 {
		_, _ = w.startClipWithVol(g.echoSnd, false, g.echoVol)
		g.echoAt = 0
	}
	if g.player != 0 {
		if x, _, z, ok := w.scriptPos(g.player); ok {
			if g.havePos {
				dx, dz := x-g.lastX, z-g.lastZ
				g.stepDist += float32(math.Hypot(float64(dx), float64(dz)))
			}
			g.lastX, g.lastZ = x, z
			g.havePos = true
			if g.stepDist > 1.1 && g.clock-g.lastStep > 0.38 {
				if r := g.rooms[g.room]; r != nil && r.step != 0 {
					_, _ = w.startClip(r.step, false)
					if r.reverb > 0 {
						g.echoAt = g.clock + 0.12
						g.echoSnd = r.step
						g.echoVol = float64(r.reverb) * 0.35
					}
				}
				g.stepDist = 0
				g.lastStep = g.clock
			}
			if t := w.terrains[w.curTerrain]; t != nil {
				c := chunkOf(x, z, t.chunkWorld())
				g.visited[visitedKey(c)] = true
			}
		}
	}
	px, py, pz, havePlayer := w.scriptPos(g.player)
	for id, ac := range g.actors {
		e := w.ents[id]
		if e == nil || e.node == nil {
			continue
		}
		if !havePlayer || g.player == 0 || g.player == id {
			ac.state = 0
			continue
		}
		ax, ay, az, ok := w.scriptPos(id)
		if !ok {
			continue
		}
		dx, dz := px-ax, pz-az
		dist := float32(math.Hypot(float64(dx), float64(dz)))
		switch {
		case dist > ac.aggro:
			ac.state = 0
		case dist < 1.5:
			ac.state = 2
		default:
			ac.state = 1
			if dist > 1e-4 {
				step := ac.speed * dt
				if step > dist-1.2 {
					step = dist - 1.2
				}
				if step < 0 {
					step = 0
				}
				nx, nz := dx/dist, dz/dist
				ny := ay
				if t := w.terrains[w.curTerrain]; t != nil {
					ny = w.terrainHeight(ax+nx*step, az+nz*step)
					if ny == 0 && py != 0 {
						ny = ay
					}
				}
				w.setScriptPos(id, ax+nx*step, ny, az+nz*step)
				e.yaw = float32(math.Atan2(float64(nx), float64(nz)) * 180 / math.Pi)
				w.applyRot(e)
			}
		}
	}
}

func visitedKey(k chunkKey) string {
	return strconvI(k.X) + "," + strconvI(k.Z)
}

func strconvI(n int) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(
		jsonNumber(n), "+", ""), " ", ""))
}

func jsonNumber(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func (w *World) gamePath(path string) string {
	if path == "" {
		path = "savegame.json"
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(w.base, path)
}

func (w *World) saveGame(path string) error {
	g := w.ensurePlay()
	sv := gameSave{
		Hour: g.hour, PlayerID: g.player, Room: g.room,
		Inventory: append([]string(nil), g.inv...),
		TalkEnt:   g.talkEnt, TalkLine: g.talkLine,
	}
	if x, y, z, ok := w.scriptPos(g.player); ok {
		sv.Player = [3]float32{x, y, z}
	}
	ids := make([]int, 0, len(g.doors))
	for id := range g.doors {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		sv.Doors = append(sv.Doors, saveDoor{ID: id, Open: g.doors[id].open})
	}
	for k, on := range g.visited {
		if on {
			sv.Visited = append(sv.Visited, k)
		}
	}
	sort.Strings(sv.Visited)
	raw, err := json.MarshalIndent(sv, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(w.gamePath(path), raw, 0o644)
}

func (w *World) loadGame(path string) error {
	raw, err := os.ReadFile(w.gamePath(path))
	if err != nil {
		return err
	}
	var sv gameSave
	if err := json.Unmarshal(raw, &sv); err != nil {
		return err
	}
	g := w.ensurePlay()
	w.setPlayHour(sv.Hour)
	g.inv = append([]string(nil), sv.Inventory...)
	g.talkEnt = sv.TalkEnt
	g.talkLine = sv.TalkLine
	g.player = sv.PlayerID
	g.visited = map[string]bool{}
	for _, k := range sv.Visited {
		g.visited[k] = true
	}
	for _, d := range sv.Doors {
		door := g.doors[d.ID]
		if door == nil {
			continue
		}
		door.open = d.Open
		door.yaw = door.base
		if d.Open {
			door.yaw = door.base + 100
		}
		if e := w.ents[d.ID]; e != nil {
			e.yaw = door.yaw
			if e.node != nil {
				w.applyRot(e)
			}
		}
	}
	w.setScriptPos(g.player, sv.Player[0], sv.Player[1], sv.Player[2])
	w.enterRoom(sv.Room)
	return nil
}
