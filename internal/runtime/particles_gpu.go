package runtime

import (
	"fmt"
	"math"
	"strings"

	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/math32"
	"github.com/go-gl/gl/v3.3-core/gl"

	"moonbasic/internal/value"
)

// GPU emitters keep every particle in a GL buffer and step it with a vertex
// shader into transform feedback (GL 3.0 core, so the 3.3 baseline has it on
// every machine). Two buffers ping-pong: read one, capture into the other,
// then draw the new one as instanced camera-facing quads after the scene
// pass. The CPU only decides how many slots of a ring to respawn each frame,
// so 100k+ particles cost one draw and one feedback pass per emitter.
//
// A GPU emitter is an entity, like a 3D CPU emitter, so its id never clashes
// with one. The Emitter* setters look the id up here first and fall through
// to the CPU emitter otherwise: switching a script is changing one call.

const (
	gpuPartFloats  = 12 // pos+age, vel+life, misc: 3 vec4
	gpuPartDefault = 65536
	gpuPartCap     = 1 << 21
	gpuSpawnSpread = 0.1 // longest a frame's spawns are spread over (s)
)

type gpuSpawnMark struct {
	at    float64
	count int
}

type gpuEmitter struct {
	ent int
	max int

	rate, acc            float64
	life0, life1         float64
	speed, speedVar      float64
	vx, vy, vz           float64
	gravX, gravY, gravZ  float64
	windX, windY, windZ  float64
	weatherWind          bool
	drag                 float64
	cone                 float64
	areaX, areaY, areaZ  float64
	shape                int // 0 point (+area), 1 box, 2 sphere, 3 disc
	shapeX, shapeY       float64
	shapeZ               float64
	style                int // 0 soft, 1 streak, 2 flake
	streak               float64
	size0, size1, tall   float64
	sizeVar              float64
	c0, c1               [4]float64
	additive             bool
	turb, turbScale      float64
	turbSpeed            float64
	ground               bool
	groundY, bounce      float64
	friction             float64
	duration, elapsed    float64
	looping, emitting    bool
	tex                  int
	texPath              string
	glTex                uint32
	texTried             bool
	pendDt               float64
	pendRate, pendBurst  int
	head                 int
	simTime, clock       float64
	org, prevOrg         [3]float32
	hasOrg               bool
	shift                [3]float32
	spawns               []gpuSpawnMark
	seed                 uint32
	vbo, simVAO, drawVAO [2]uint32
	cur                  int
	glMax                int // particles the GL buffers hold; 0 none
	glFailed             bool
}

type gpuParticles struct {
	ems                 map[int]*gpuEmitter
	simProg, drawProg   uint32
	progOK, progTried   bool
	locs                map[uint32]map[string]int32
	simSerial           int
	originX, originZ    float32
	hasOrigin           bool
	queries             [2]uint32
	qPend               [2]bool
	qi                  int
	lastMs              float64
	trashBufs, trashVAO []uint32
	trashTex            []uint32
}

func defaultGPUEmitter(max int) *gpuEmitter {
	if max < 1 {
		max = gpuPartDefault
	}
	if max > gpuPartCap {
		max = gpuPartCap
	}
	// The CPU emitter's 3D defaults, so a script that only swaps the create
	// call looks the same.
	return &gpuEmitter{
		max:       max,
		rate:      20,
		life0:     1.2,
		life1:     1.2,
		speed:     4,
		vy:        1,
		gravY:     -6,
		drag:      0.15,
		cone:      180,
		size0:     0.18,
		size1:     0.06,
		tall:      1,
		streak:    0.05,
		c0:        [4]float64{1, 0.7, 0.25, 1},
		c1:        [4]float64{1, 0.35, 0.08, 0},
		turbScale: 0.5,
		turbSpeed: 0.6,
		bounce:    0.3,
		friction:  0.2,
		looping:   true,
		emitting:  true,
		seed:      0x9e3779b9,
	}
}

func (w *World) gpuEmitterOf(id int) *gpuEmitter {
	if w.gpuPart.ems == nil {
		return nil
	}
	return w.gpuPart.ems[id]
}

func (w *World) newGPUEmitter(max, parent int) (int, error) {
	if w.mode2D || w.scene == nil {
		return 0, fmt.Errorf("CreateGPUEmitter: needs Graphics3D (use CreateEmitter2D for 2D)")
	}
	if w.gpuPart.ems == nil {
		w.gpuPart.ems = map[int]*gpuEmitter{}
	}
	e := defaultGPUEmitter(max)
	id := w.addEntity(&Entity{node: core.NewNode()}, parent)
	e.ent = id
	e.seed ^= uint32(id) * 2246822519
	w.gpuPart.ems[id] = e
	return id, nil
}

// freeGPUEmitter queues its GL objects; they go at the next draw, where the
// context is known to be current.
func (w *World) freeGPUEmitter(id int, entityToo bool) {
	gp := &w.gpuPart
	e := gp.ems[id]
	if e == nil {
		return
	}
	w.dropGPUEmitterGL(e)
	if e.glTex != 0 {
		gp.trashTex = append(gp.trashTex, e.glTex)
		e.glTex = 0
	}
	delete(gp.ems, id)
	if entityToo && w.ents[id] != nil {
		w.freeEntityID(id)
	}
}

func (w *World) dropGPUEmitterGL(e *gpuEmitter) {
	gp := &w.gpuPart
	if e.glMax == 0 {
		return
	}
	gp.trashBufs = append(gp.trashBufs, e.vbo[0], e.vbo[1])
	gp.trashVAO = append(gp.trashVAO, e.simVAO[0], e.simVAO[1], e.drawVAO[0], e.drawVAO[1])
	e.vbo, e.simVAO, e.drawVAO = [2]uint32{}, [2]uint32{}, [2]uint32{}
	e.glMax = 0
	e.spawns = e.spawns[:0]
	e.head = 0
}

func (w *World) resetGPUEmitters() {
	for id := range w.gpuPart.ems {
		w.freeGPUEmitter(id, false)
	}
}

func (w *World) hasGPUEmitters() bool {
	return len(w.gpuPart.ems) > 0
}

// advance runs the emitter clock on the CPU: duration, loop, and how many
// particles the rate owes. The GPU step at the next render spawns them.
func (e *gpuEmitter) advance(dt float64, active bool) {
	if dt <= 0 {
		return
	}
	if e.duration > 0 {
		e.elapsed += dt
		if e.elapsed >= e.duration {
			if e.looping {
				e.elapsed = 0
			} else {
				e.emitting = false
			}
		}
	}
	if e.emitting && e.rate > 0 && active {
		e.acc += e.rate * dt
		n := int(e.acc)
		e.acc -= float64(n)
		e.pendRate += n
		if e.pendRate > e.max {
			e.pendRate = e.max
			e.acc = 0
		}
	}
	e.pendDt += dt
}

func (e *gpuEmitter) burst(n int) {
	if n < 1 {
		return
	}
	e.pendBurst += n
	if e.pendBurst > e.max {
		e.pendBurst = e.max
	}
}

func (e *gpuEmitter) lifeMax() float64 {
	return math.Max(e.life0, e.life1)
}

// planSpawn takes the frame's owed particles: the ring slots to respawn and
// how many seconds their births spread over (0 for a burst, so it pops).
func (e *gpuEmitter) planSpawn() (start, count int, spread float64) {
	count = e.pendRate + e.pendBurst
	if count > e.max {
		count = e.max
	}
	if count < 0 {
		count = 0
	}
	if e.pendBurst == 0 {
		spread = math.Min(e.pendDt, gpuSpawnSpread)
	}
	start = e.head
	if e.max > 0 {
		e.head = (e.head + count) % e.max
	}
	e.pendRate, e.pendBurst = 0, 0
	return
}

// alive estimates live particles from recent spawns: each lives at most
// lifeMax, and a slot respawned early only replaces one.
func (e *gpuEmitter) alive() int {
	lm := e.lifeMax()
	n := 0
	keep := e.spawns[:0]
	for _, s := range e.spawns {
		if e.clock-s.at < lm {
			keep = append(keep, s)
			n += s.count
		}
	}
	e.spawns = keep
	if n > e.max {
		n = e.max
	}
	return n
}

// gpuSimParams is one step's uniforms, in scene space (z flipped).
type gpuSimParams struct {
	Dt, SimTime               float32
	MaxCount                  int32
	SpawnStart, SpawnCount    int32
	SpawnSpread               float32
	Seed                      uint32
	Origin, PrevOrigin        [3]float32
	Shape                     int32
	ShapeSize                 [3]float32
	Dir                       [3]float32
	Speed, SpeedVar, ConeHalf float32
	Life                      [2]float32
	Accel                     [3]float32
	Drag                      float32
	Turb, Ground              [4]float32
	Shift                     [3]float32
	Style                     int32
}

type gpuDrawParams struct {
	Size                     [2]float32
	SizeVar, Tall, StreakLen float32
	Color0, Color1           [4]float32
	Style, Additive, UseTex  int32
}

func (e *gpuEmitter) simParams(dt float64, start, count int, spread float64, windX, windY, windZ float64) gpuSimParams {
	p := gpuSimParams{
		Dt:          float32(dt),
		SimTime:     float32(e.simTime),
		MaxCount:    int32(e.max),
		SpawnStart:  int32(start),
		SpawnCount:  int32(count),
		SpawnSpread: float32(spread),
		Seed:        e.seed,
		Origin:      e.org,
		PrevOrigin:  e.prevOrg,
		Style:       int32(e.style),
		Drag:        float32(math.Max(0, e.drag)),
		Shift:       e.shift,
	}
	dx, dy, dz := e.vx, e.vy, e.vz
	if dx == 0 && dy == 0 && dz == 0 {
		dy = 1
	}
	l := math.Sqrt(dx*dx + dy*dy + dz*dz)
	p.Dir = [3]float32{float32(dx / l), float32(dy / l), float32(-dz / l)}
	spd := e.speed
	if spd == 0 {
		spd = 1 // as coneVelocity
	}
	p.Speed = float32(spd)
	p.SpeedVar = float32(clampF(e.speedVar, 0, 1))
	p.ConeHalf = float32(clampF(e.cone*0.5*math.Pi/180, 0, math.Pi))
	l0, l1 := e.life0, e.life1
	if l1 <= 0 {
		l1 = l0
	}
	if l1 < l0 {
		l0, l1 = l1, l0
	}
	p.Life = [2]float32{float32(math.Max(l0, 0.001)), float32(math.Max(l1, 0.001))}
	ax := e.gravX + e.windX + windX
	ay := e.gravY + e.windY + windY
	az := e.gravZ + e.windZ + windZ
	p.Accel = [3]float32{float32(ax), float32(ay), float32(-az)}
	switch e.shape {
	case 1:
		sx, sy, sz := e.shapeX, e.shapeY, e.shapeZ
		if sx <= 0 && sy <= 0 && sz <= 0 {
			sx, sy, sz = e.areaX, e.areaY, e.areaZ
		}
		if sx <= 0 && sy <= 0 && sz <= 0 {
			sx, sy, sz = 0.5, 0.5, 0.5
		}
		p.Shape = 0
		p.ShapeSize = [3]float32{float32(sx), float32(sy), float32(sz)}
	case 2, 3:
		r := e.shapeX
		if r <= 0 {
			r = e.areaX
		}
		if r <= 0 {
			r = 0.5
		}
		p.Shape = int32(e.shape)
		p.ShapeSize = [3]float32{float32(r), float32(r), float32(r)}
	default:
		p.Shape = 0
		p.ShapeSize = [3]float32{float32(e.areaX), float32(e.areaY), float32(e.areaZ)}
	}
	if e.turb > 0 {
		sc := e.turbScale
		if sc <= 0 {
			sc = 0.5
		}
		p.Turb = [4]float32{float32(e.turb), float32(sc), float32(e.turbSpeed), 1}
	}
	if e.ground {
		p.Ground = [4]float32{1, float32(e.groundY), float32(clampF(e.bounce, 0, 1)), float32(clampF(e.friction, 0, 1))}
	}
	return p
}

func (e *gpuEmitter) drawParams() gpuDrawParams {
	d := gpuDrawParams{
		Size:      [2]float32{float32(math.Max(e.size0, 0)), float32(math.Max(e.size1, 0))},
		SizeVar:   float32(clampF(e.sizeVar, 0, 1)),
		Tall:      float32(math.Max(e.tall, 0.001)),
		StreakLen: float32(math.Max(e.streak, 0)),
		Style:     int32(e.style),
	}
	for i := 0; i < 4; i++ {
		d.Color0[i] = float32(e.c0[i])
		d.Color1[i] = float32(e.c1[i])
	}
	if e.additive {
		d.Additive = 1
	}
	if e.glTex != 0 {
		d.UseTex = 1
	}
	return d
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (e *gpuEmitter) setColor(a []float64) {
	var tmp emitter
	tmp.setColor(a)
	if len(a) < 3 {
		return
	}
	e.c0 = [4]float64{tmp.cr0, tmp.cg0, tmp.cb0, tmp.ca0}
	e.c1 = [4]float64{tmp.cr1, tmp.cg1, tmp.cb1, tmp.ca1}
}

// setSize follows EmitterSize: start, end, or the 5-arg w0, h0, w1, h1.
func (e *gpuEmitter) setSize(a []value.Value) {
	e.size0 = argN(a, 1, 0.2)
	e.size1 = argN(a, 2, e.size0*0.4)
	if len(a) >= 5 {
		e.size0 = argN(a, 1, 0.2)
		e.tall = argN(a, 2, 1) / mathMax(e.size0, 0.001)
		e.size1 = argN(a, 3, e.size0)
		if argN(a, 4, e.size0) > 0 && e.size1 > 0 {
			e.tall = argN(a, 4, e.size1) / e.size1
		}
	}
}

// applyShape takes a spawn shape (point, box, sphere, disc) or a look (soft,
// streak, flake). Extra args size the spawn shape or stretch a streak.
func (e *gpuEmitter) applyShape(name string, a []value.Value) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "point", "":
		e.shape = 0
	case "box", "cube", "area":
		e.shape = 1
		e.shapeX, e.shapeY, e.shapeZ = argN(a, 2, 0), argN(a, 3, argN(a, 2, 0)), argN(a, 4, argN(a, 2, 0))
	case "sphere", "ball":
		e.shape = 2
		e.shapeX = argN(a, 2, 0)
	case "disc", "disk", "circle", "ring":
		e.shape = 3
		e.shapeX = argN(a, 2, 0)
	case "streak", "rain", "spark", "sparks":
		e.style = 1
		if len(a) > 2 {
			e.streak = argN(a, 2, 0.05)
		}
	case "flake", "snow":
		e.style = 2
	default: // soft, mesh, 3d: GPU particles are always sprites
		e.style = 0
	}
}

func (w *World) tickGPUEmitters(dt float64) {
	gp := &w.gpuPart
	for id, e := range gp.ems {
		ent := w.ents[e.ent]
		if ent == nil || ent.node == nil {
			// FreeEntity on the emitter id.
			w.freeGPUEmitter(id, false)
			continue
		}
		e.advance(dt, ent.node.GetNode().Visible())
	}
}

func (w *World) gpuWeatherWind() (float64, float64, float64) {
	return w.wx.windX, w.wx.windY, w.wx.windZ
}

// noteWorldShift hands a floating-origin shift to every live particle.
func (w *World) noteGPUWorldShift() {
	gp := &w.gpuPart
	ox, oz := w.worldOrigin()
	if !gp.hasOrigin {
		gp.originX, gp.originZ, gp.hasOrigin = ox, oz, true
		return
	}
	dx, dz := ox-gp.originX, oz-gp.originZ
	if dx == 0 && dz == 0 {
		return
	}
	gp.originX, gp.originZ = ox, oz
	// Script x/z moved by -d; scene z is the script's -z.
	sx, _, sz := toG3N(dx, 0, dz)
	for _, e := range gp.ems {
		e.shift[0] += sx
		e.shift[2] += sz
		e.prevOrg[0] -= sx
		e.prevOrg[2] -= sz
	}
}

func (gp *gpuParticles) loc(prog uint32, name string) int32 {
	if gp.locs == nil {
		gp.locs = map[uint32]map[string]int32{}
	}
	m := gp.locs[prog]
	if m == nil {
		m = map[string]int32{}
		gp.locs[prog] = m
	}
	if l, ok := m[name]; ok {
		return l
	}
	l := gl.GetUniformLocation(prog, gl.Str(name+"\x00"))
	m[name] = l
	return l
}

func linkFeedbackProgram(vs, fs string, varyings []string) (uint32, error) {
	v, err := compileGLShader(gl.VERTEX_SHADER, vs)
	if err != nil {
		return 0, fmt.Errorf("sim vertex: %v", err)
	}
	f, err := compileGLShader(gl.FRAGMENT_SHADER, fs)
	if err != nil {
		gl.DeleteShader(v)
		return 0, fmt.Errorf("sim fragment: %v", err)
	}
	prog := gl.CreateProgram()
	gl.AttachShader(prog, v)
	gl.AttachShader(prog, f)
	names := make([]string, len(varyings))
	for i, s := range varyings {
		names[i] = s + "\x00"
	}
	cv, free := gl.Strs(names...)
	gl.TransformFeedbackVaryings(prog, int32(len(names)), cv, gl.INTERLEAVED_ATTRIBS)
	free()
	gl.LinkProgram(prog)
	gl.DeleteShader(v)
	gl.DeleteShader(f)
	var ok int32
	gl.GetProgramiv(prog, gl.LINK_STATUS, &ok)
	if ok == gl.FALSE {
		log := glInfoLog(prog, true)
		gl.DeleteProgram(prog)
		return 0, fmt.Errorf("sim link: %s", log)
	}
	return prog, nil
}

func (w *World) ensureGPUPartPrograms() bool {
	gp := &w.gpuPart
	if gp.progOK {
		return true
	}
	if gp.progTried {
		return false
	}
	gp.progTried = true
	m := w.ensureGLMod()
	if !m.caps.instance {
		w.skipGL("GPU particles", "OpenGL 3.1 instancing and transform feedback")
		return false
	}
	drainGL("before GPU particles")
	sim, err := linkFeedbackProgram(mbGPUPartSimVertex, mbGPUPartSimFragment, gpuPartVaryings)
	if err != nil {
		fmt.Println("GPU particles:", err)
		return false
	}
	draw, err := compileLink(
		struct {
			kind uint32
			src  string
		}{gl.VERTEX_SHADER, mbGPUPartVertex},
		struct {
			kind uint32
			src  string
		}{gl.FRAGMENT_SHADER, mbGPUPartFragment},
	)
	if err != nil {
		gl.DeleteProgram(sim)
		fmt.Println("GPU particles: draw program", err)
		return false
	}
	gp.simProg, gp.drawProg = sim, draw
	gl.GenQueries(2, &gp.queries[0])
	gp.progOK = true
	return true
}

func (w *World) ensureGPUEmitterGL(e *gpuEmitter) bool {
	if e.glFailed {
		return false
	}
	if e.glMax == e.max && e.vbo[0] != 0 {
		return true
	}
	w.dropGPUEmitterGL(e)
	w.flushGPUTrash()
	drainGL("before GPU emitter alloc")
	zero := make([]float32, e.max*gpuPartFloats)
	gl.GenBuffers(2, &e.vbo[0])
	gl.GenVertexArrays(2, &e.simVAO[0])
	gl.GenVertexArrays(2, &e.drawVAO[0])
	stride := int32(gpuPartFloats * 4)
	for i := 0; i < 2; i++ {
		gl.BindBuffer(gl.ARRAY_BUFFER, e.vbo[i])
		gl.BufferData(gl.ARRAY_BUFFER, len(zero)*4, gl.Ptr(zero), gl.DYNAMIC_COPY)
		for _, vao := range []uint32{e.simVAO[i], e.drawVAO[i]} {
			gl.BindVertexArray(vao)
			gl.BindBuffer(gl.ARRAY_BUFFER, e.vbo[i])
			for a := uint32(0); a < 3; a++ {
				gl.EnableVertexAttribArray(a)
				gl.VertexAttribPointerWithOffset(a, 4, gl.FLOAT, false, stride, uintptr(a*16))
				if vao == e.drawVAO[i] {
					gl.VertexAttribDivisor(a, 1)
				}
			}
		}
	}
	gl.BindVertexArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	e.glMax = e.max
	e.cur = 0
	e.head = 0
	e.spawns = e.spawns[:0]
	if err := gl.GetError(); err != gl.NO_ERROR {
		for gl.GetError() != gl.NO_ERROR {
		}
		fmt.Printf("GPU particles: buffer alloc failed (%s, %d particles)\n", glErrName(err), e.max)
		w.dropGPUEmitterGL(e)
		e.glFailed = true
		return false
	}
	return true
}

func (w *World) flushGPUTrash() {
	gp := &w.gpuPart
	for _, b := range gp.trashBufs {
		if b != 0 {
			gl.DeleteBuffers(1, &b)
		}
	}
	for _, v := range gp.trashVAO {
		if v != 0 {
			gl.DeleteVertexArrays(1, &v)
		}
	}
	for _, t := range gp.trashTex {
		if t != 0 {
			gl.DeleteTextures(1, &t)
		}
	}
	gp.trashBufs, gp.trashVAO, gp.trashTex = gp.trashBufs[:0], gp.trashVAO[:0], gp.trashTex[:0]
}

func (w *World) ensureGPUEmitterTex(e *gpuEmitter) {
	if e.tex == 0 || e.texTried {
		return
	}
	e.texTried = true
	t := w.texs[e.tex]
	if t == nil || t.path == "" {
		return
	}
	img, err := loadRGBAFile(t.path)
	if err != nil {
		fmt.Println("GPU particles: EmitterParticle", err)
		return
	}
	e.glTex, _, _ = uploadGLTexture(img)
}

// drawGPUEmitters runs after the scene pass for each camera. The first call
// in a frame also steps every emitter, so split screens simulate once.
func (w *World) drawGPUEmitters(cam *camera.Camera, view, viewProj *math32.Matrix4) {
	gp := &w.gpuPart
	if len(gp.ems) == 0 || w.app == nil {
		return
	}
	if !w.ensureGPUPartPrograms() {
		return
	}
	gs := w.app.Gls()
	w.flushGPUTrash()
	first := gp.simSerial != w.renderSerial
	timing := false
	if first {
		gp.simSerial = w.renderSerial
		timing = gp.beginTimer()
		w.simulateGPUEmitters()
	}
	w.renderGPUEmitters(gs, cam, view, viewProj)
	if timing {
		gl.EndQuery(gl.TIME_ELAPSED)
		gp.qPend[gp.qi] = true
		gp.qi ^= 1
	}
	drainGL("GPU particles")
}

// beginTimer reads the query from two frames back (no stall) and starts one.
func (gp *gpuParticles) beginTimer() bool {
	q := gp.queries[gp.qi]
	if q == 0 {
		return false
	}
	if gp.qPend[gp.qi] {
		var avail int32
		gl.GetQueryObjectiv(q, gl.QUERY_RESULT_AVAILABLE, &avail)
		if avail == 0 {
			return false
		}
		var ns uint64
		gl.GetQueryObjectui64v(q, gl.QUERY_RESULT, &ns)
		gp.lastMs = float64(ns) / 1e6
		gp.qPend[gp.qi] = false
	}
	gl.BeginQuery(gl.TIME_ELAPSED, q)
	return true
}

func (w *World) simulateGPUEmitters() {
	gp := &w.gpuPart
	w.noteGPUWorldShift()
	var wwx, wwy, wwz float64
	prog := gp.simProg
	gl.Enable(gl.RASTERIZER_DISCARD)
	gl.UseProgram(prog)
	for _, e := range gp.ems {
		dt := e.pendDt
		if dt > 0.1 {
			dt = 0.1 // a hitch must not fling everything at once
		}
		ent := w.ents[e.ent]
		if ent != nil && ent.node != nil {
			p := worldPos(ent.node.GetNode())
			e.org = [3]float32{p.X, p.Y, p.Z}
			if !e.hasOrg {
				e.prevOrg = e.org
				e.hasOrg = true
			}
		}
		owed := e.pendRate+e.pendBurst > 0
		if !owed && (dt <= 0 || e.alive() == 0) {
			e.pendDt = 0
			e.prevOrg = e.org
			e.shift = [3]float32{}
			continue
		}
		if !w.ensureGPUEmitterGL(e) {
			e.pendRate, e.pendBurst, e.pendDt = 0, 0, 0
			continue
		}
		e.clock += dt
		e.simTime += dt
		start, count, spread := e.planSpawn()
		if count > 0 {
			e.spawns = append(e.spawns, gpuSpawnMark{at: e.clock, count: count})
		}
		e.seed = e.seed*1664525 + 1013904223
		wx, wy, wz := 0.0, 0.0, 0.0
		if e.weatherWind {
			if wwx == 0 && wwy == 0 && wwz == 0 {
				wwx, wwy, wwz = w.gpuWeatherWind()
			}
			wx, wy, wz = wwx, wwy, wwz
		}
		p := e.simParams(dt, start, count, spread, wx, wy, wz)
		w.setSimUniforms(prog, &p)
		src, dst := e.cur, 1-e.cur
		gl.BindVertexArray(e.simVAO[src])
		gl.BindBufferBase(gl.TRANSFORM_FEEDBACK_BUFFER, 0, e.vbo[dst])
		gl.BeginTransformFeedback(gl.POINTS)
		gl.DrawArrays(gl.POINTS, 0, int32(e.max))
		gl.EndTransformFeedback()
		e.cur = dst
		e.pendDt = 0
		e.prevOrg = e.org
		e.shift = [3]float32{}
	}
	gl.BindBufferBase(gl.TRANSFORM_FEEDBACK_BUFFER, 0, 0)
	gl.BindVertexArray(0)
	gl.Disable(gl.RASTERIZER_DISCARD)
}

func (w *World) setSimUniforms(prog uint32, p *gpuSimParams) {
	gp := &w.gpuPart
	u1f := func(n string, v float32) {
		if l := gp.loc(prog, n); l >= 0 {
			gl.Uniform1f(l, v)
		}
	}
	u1i := func(n string, v int32) {
		if l := gp.loc(prog, n); l >= 0 {
			gl.Uniform1i(l, v)
		}
	}
	u3 := func(n string, v [3]float32) {
		if l := gp.loc(prog, n); l >= 0 {
			gl.Uniform3f(l, v[0], v[1], v[2])
		}
	}
	u4 := func(n string, v [4]float32) {
		if l := gp.loc(prog, n); l >= 0 {
			gl.Uniform4f(l, v[0], v[1], v[2], v[3])
		}
	}
	u1f("Dt", p.Dt)
	u1f("SimTime", p.SimTime)
	u1i("MaxCount", p.MaxCount)
	u1i("SpawnStart", p.SpawnStart)
	u1i("SpawnCount", p.SpawnCount)
	u1f("SpawnSpread", p.SpawnSpread)
	if l := gp.loc(prog, "Seed"); l >= 0 {
		gl.Uniform1ui(l, p.Seed)
	}
	u3("Origin", p.Origin)
	u3("PrevOrigin", p.PrevOrigin)
	u1i("Shape", p.Shape)
	u3("ShapeSize", p.ShapeSize)
	u3("Dir", p.Dir)
	u1f("Speed", p.Speed)
	u1f("SpeedVar", p.SpeedVar)
	u1f("ConeHalf", p.ConeHalf)
	if l := gp.loc(prog, "Life"); l >= 0 {
		gl.Uniform2f(l, p.Life[0], p.Life[1])
	}
	u3("Accel", p.Accel)
	u1f("Drag", p.Drag)
	u4("Turb", p.Turb)
	u4("Ground", p.Ground)
	u3("Shift", p.Shift)
	u1i("Style", p.Style)
}

func (w *World) renderGPUEmitters(gs *gls.GLS, cam *camera.Camera, view, viewProj *math32.Matrix4) {
	gp := &w.gpuPart
	prog := gp.drawProg
	right := math32.Vector3{X: view[0], Y: view[4], Z: view[8]}
	up := math32.Vector3{X: view[1], Y: view[5], Z: view[9]}
	right.Normalize()
	up.Normalize()
	cp := worldPos(cam.GetNode())
	// Through G3N's cached state calls, so its next material setup is right.
	gs.Enable(gls.BLEND)
	gs.BlendEquation(gls.FUNC_ADD)
	gs.BlendFunc(gls.ONE, gls.ONE_MINUS_SRC_ALPHA)
	gs.Enable(gls.DEPTH_TEST)
	gs.DepthFunc(gls.LEQUAL)
	gs.DepthMask(false)
	gs.Disable(gls.CULL_FACE)
	gl.UseProgram(prog)
	if l := gp.loc(prog, "ViewProj"); l >= 0 {
		gl.UniformMatrix4fv(l, 1, false, &viewProj[0])
	}
	if l := gp.loc(prog, "CamRight"); l >= 0 {
		gl.Uniform3f(l, right.X, right.Y, right.Z)
	}
	if l := gp.loc(prog, "CamUp"); l >= 0 {
		gl.Uniform3f(l, up.X, up.Y, up.Z)
	}
	if l := gp.loc(prog, "CamPos"); l >= 0 {
		gl.Uniform3f(l, cp.X, cp.Y, cp.Z)
	}
	if l := gp.loc(prog, "FogMode"); l >= 0 {
		gl.Uniform1i(l, int32(w.fogMode))
	}
	if l := gp.loc(prog, "FogColor"); l >= 0 {
		gl.Uniform3f(l, w.fogRGB.R, w.fogRGB.G, w.fogRGB.B)
	}
	if l := gp.loc(prog, "FogNear"); l >= 0 {
		gl.Uniform1f(l, w.fogNear)
	}
	if l := gp.loc(prog, "FogFar"); l >= 0 {
		gl.Uniform1f(l, w.fogFar)
	}
	if l := gp.loc(prog, "FogDensity"); l >= 0 {
		gl.Uniform1f(l, w.fogDensity)
	}
	if l := gp.loc(prog, "PartTex"); l >= 0 {
		gl.Uniform1i(l, unitParticle)
	}
	boundTex := false
	for _, e := range gp.ems {
		if e.glMax == 0 || e.alive() == 0 {
			continue
		}
		w.ensureGPUEmitterTex(e)
		d := e.drawParams()
		if d.UseTex == 1 {
			gs.ActiveTexture(gls.TEXTURE0 + unitParticle)
			gl.BindTexture(gl.TEXTURE_2D, e.glTex)
			boundTex = true
		}
		if l := gp.loc(prog, "Size"); l >= 0 {
			gl.Uniform2f(l, d.Size[0], d.Size[1])
		}
		if l := gp.loc(prog, "SizeVar"); l >= 0 {
			gl.Uniform1f(l, d.SizeVar)
		}
		if l := gp.loc(prog, "Tall"); l >= 0 {
			gl.Uniform1f(l, d.Tall)
		}
		if l := gp.loc(prog, "StreakLen"); l >= 0 {
			gl.Uniform1f(l, d.StreakLen)
		}
		if l := gp.loc(prog, "Color0"); l >= 0 {
			gl.Uniform4f(l, d.Color0[0], d.Color0[1], d.Color0[2], d.Color0[3])
		}
		if l := gp.loc(prog, "Color1"); l >= 0 {
			gl.Uniform4f(l, d.Color1[0], d.Color1[1], d.Color1[2], d.Color1[3])
		}
		if l := gp.loc(prog, "Style"); l >= 0 {
			gl.Uniform1i(l, d.Style)
		}
		if l := gp.loc(prog, "Additive"); l >= 0 {
			gl.Uniform1i(l, d.Additive)
		}
		if l := gp.loc(prog, "UseTex"); l >= 0 {
			gl.Uniform1i(l, d.UseTex)
		}
		gl.BindVertexArray(e.drawVAO[e.cur])
		gl.DrawArraysInstanced(gl.TRIANGLE_STRIP, 0, 4, int32(e.glMax))
	}
	gl.BindVertexArray(0)
	if boundTex {
		gs.ActiveTexture(gls.TEXTURE0 + unitParticle)
		gl.BindTexture(gl.TEXTURE_2D, 0)
	}
	gs.ActiveTexture(gls.TEXTURE0)
	gs.DepthMask(true)
	gs.Disable(gls.BLEND)
}

// gpuParticleCount is the estimated live total over every GPU emitter.
func (w *World) gpuParticleCount() int {
	n := 0
	for _, e := range w.gpuPart.ems {
		n += e.alive()
	}
	return n
}

// withGPUEmitters wraps the CPU emitter setters: a GPU emitter id is handled
// here, anything else goes to the CPU command unchanged.
func (w *World) withGPUEmitters(m map[string]cmd, z func() (value.Value, error)) map[string]cmd {
	set := map[string]func(e *gpuEmitter, id int, a []value.Value) (value.Value, error){
		"emitterrate": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			e.rate = argN(a, 1, 20)
			return z()
		},
		"emittermax": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			n := argI(a, 1, gpuPartDefault)
			if n < 1 {
				n = 1
			}
			if n > gpuPartCap {
				n = gpuPartCap
			}
			e.max = n // the buffers are rebuilt at the next draw
			return z()
		},
		"emitterlife": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			e.life0 = argN(a, 1, 1)
			e.life1 = argN(a, 2, e.life0)
			return z()
		},
		"emitterspeed": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			e.speed = argN(a, 1, 4)
			e.speedVar = argN(a, 2, 0)
			return z()
		},
		"emittersize": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			e.setSize(a)
			return z()
		},
		"emittercolor": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			vals := make([]float64, 0, 8)
			for i := 1; i < len(a) && i <= 8; i++ {
				vals = append(vals, argN(a, i, 0))
			}
			e.setColor(vals)
			return z()
		},
		"emittervelocity": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			e.vx, e.vy, e.vz = argN(a, 1, 0), argN(a, 2, 1), argN(a, 3, 0)
			return z()
		},
		"emittergravity": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			if len(a) >= 4 {
				e.gravX, e.gravY, e.gravZ = argN(a, 1, 0), argN(a, 2, -6), argN(a, 3, 0)
			} else {
				e.gravY = argN(a, 1, -6)
			}
			return z()
		},
		"emitterwind": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			e.windX, e.windY, e.windZ = argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0)
			return z()
		},
		"emittercone": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			e.cone = argN(a, 1, 30)
			return z()
		},
		"emittershape": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			e.applyShape(argS(a, 1), a)
			return z()
		},
		"emitterdrag": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			e.drag = argN(a, 1, 0)
			return z()
		},
		"emitterduration": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			e.duration = argN(a, 1, 0)
			e.elapsed = 0
			e.emitting = true
			return z()
		},
		"emitterloop": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			e.looping = argI(a, 1, 1) != 0
			if e.looping {
				e.emitting = true
			}
			return z()
		},
		"emitterarea": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			e.areaX, e.areaY, e.areaZ = argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0)
			return z()
		},
		"emitterparticle": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			if e.glTex != 0 {
				w.gpuPart.trashTex = append(w.gpuPart.trashTex, e.glTex)
				e.glTex = 0
			}
			e.tex = argI(a, 1, 0)
			e.texTried = false
			return z()
		},
		"positionemitter": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			if ent := w.ents[e.ent]; ent != nil && ent.node != nil {
				gx, gy, gz := toG3N(float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)))
				ent.node.GetNode().SetPosition(gx, gy, gz)
			}
			return z()
		},
		"parentemitter": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			ent := w.ents[e.ent]
			pid := argI(a, 1, 0)
			if ent == nil || ent.node == nil || pid == id {
				return z()
			}
			pn := w.parentNode(pid)
			if pn == nil {
				return value.Value{}, fmt.Errorf("ParentEmitter: invalid parent %d", pid)
			}
			cn := ent.node.GetNode()
			if p := cn.Parent(); p != nil {
				p.GetNode().Remove(ent.node)
			}
			pn.GetNode().Add(ent.node)
			ent.parent = pid
			return z()
		},
		"emit": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			e.burst(maxInt(argI(a, 1, 1), 1))
			return z()
		},
		"emitterburst": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			e.burst(maxInt(argI(a, 1, 16), 1))
			return z()
		},
		"freeemitter": func(e *gpuEmitter, id int, a []value.Value) (value.Value, error) {
			w.freeGPUEmitter(id, true)
			return z()
		},
	}
	for name, fn := range set {
		cpu := m[name]
		fn := fn
		m[name] = func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			if e := w.gpuEmitterOf(id); e != nil {
				return fn(e, id, a)
			}
			if cpu == nil {
				return z()
			}
			return cpu(a)
		}
	}
	return m
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// gpuParticleCommands are the GPU-only extras. Each takes a CPU emitter id
// too and does nothing with it, so a script can keep the calls either way.
func (w *World) gpuParticleCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	gpu := func(fn func(*gpuEmitter, []value.Value) (value.Value, error)) cmd {
		return n(func(a []value.Value) (value.Value, error) {
			e := w.gpuEmitterOf(argI(a, 0, 0))
			if e == nil {
				return z()
			}
			return fn(e, a)
		})
	}
	return map[string]cmd{
		"creategpuemitter": need(func(a []value.Value) (value.Value, error) {
			id, err := w.newGPUEmitter(argI(a, 0, gpuPartDefault), argI(a, 1, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(id)), nil
		}),
		"isgpuemitter": n(func(a []value.Value) (value.Value, error) {
			if w.gpuEmitterOf(argI(a, 0, 0)) != nil {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"emitterblend": gpu(func(e *gpuEmitter, a []value.Value) (value.Value, error) {
			if len(a) > 1 && a[1].Kind == value.KindStr {
				switch strings.ToLower(strings.TrimSpace(argS(a, 1))) {
				case "add", "additive", "glow":
					e.additive = true
				case "alpha", "blend", "normal":
					e.additive = false
				default:
					return value.Value{}, fmt.Errorf("EmitterBlend: mode %q (use \"add\" or \"alpha\")", argS(a, 1))
				}
				return z()
			}
			e.additive = argI(a, 1, 1) != 0
			return z()
		}),
		"emitterturbulence": gpu(func(e *gpuEmitter, a []value.Value) (value.Value, error) {
			e.turb = argN(a, 1, 1)
			e.turbScale = argN(a, 2, 0.5)
			e.turbSpeed = argN(a, 3, 0.6)
			return z()
		}),
		"emitterground": gpu(func(e *gpuEmitter, a []value.Value) (value.Value, error) {
			e.ground = argI(a, 1, 1) != 0
			e.groundY = argN(a, 2, e.groundY)
			e.bounce = argN(a, 3, e.bounce)
			e.friction = argN(a, 4, e.friction)
			return z()
		}),
		"emitterweatherwind": gpu(func(e *gpuEmitter, a []value.Value) (value.Value, error) {
			e.weatherWind = argI(a, 1, 1) != 0
			return z()
		}),
		"emittersizevar": gpu(func(e *gpuEmitter, a []value.Value) (value.Value, error) {
			e.sizeVar = argN(a, 1, 0.3)
			return z()
		}),
		"emittercount": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			if e := w.gpuEmitterOf(id); e != nil {
				return value.Num(float64(e.alive())), nil
			}
			if e := w.emitterOf(id); e != nil {
				return value.Num(float64(len(e.parts))), nil
			}
			return value.Num(0), nil
		}),
		"gpuparticlecount": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.gpuParticleCount())), nil
		}),
		"gpuparticlems": n(func(a []value.Value) (value.Value, error) {
			return value.Num(w.gpuPart.lastMs), nil
		}),
		"gpuparticlesok": n(func(a []value.Value) (value.Value, error) {
			if w.app == nil {
				return value.Num(0), nil
			}
			w.detectModernGL()
			if w.glmod.caps.instance && (!w.gpuPart.progTried || w.gpuPart.progOK) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
	}
}
