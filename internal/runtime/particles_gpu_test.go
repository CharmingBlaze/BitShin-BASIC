package runtime

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/g3n/engine/core"

	"moonbasic/internal/glslang"
	"moonbasic/internal/value"
)

func gpuTestWorld(t *testing.T) (*World, int) {
	t.Helper()
	w := New(".")
	w.scene = core.NewNode()
	id, err := w.newGPUEmitter(1000, 0)
	if err != nil {
		t.Fatal(err)
	}
	return w, id
}

func TestGPUParticleCommandsRegistered(t *testing.T) {
	m := New(".").commandTable()
	for _, name := range []string{
		"creategpuemitter", "isgpuemitter", "emitterblend", "emitterturbulence",
		"emitterground", "emitterweatherwind", "emittersizevar", "emittercount",
		"gpuparticlecount", "gpuparticlems", "gpuparticlesok",
		"emitterrate", "emitterspread", "setemitterrate", "positionemitter",
	} {
		if m[name] == nil {
			t.Errorf("missing %s", name)
		}
	}
}

func TestGPUEmitterNeeds3D(t *testing.T) {
	w := New(".")
	w.mode2D = true
	if _, err := w.newGPUEmitter(10, 0); err == nil || !strings.Contains(err.Error(), "CreateGPUEmitter") {
		t.Fatalf("want a CreateGPUEmitter error, got %v", err)
	}
}

// The CPU setter names reach the GPU emitter; a CPU id still reaches the CPU.
func TestGPUEmitterSettersDispatch(t *testing.T) {
	w, id := gpuTestWorld(t)
	m := w.commandTable()
	w.cmdMap = m
	call := func(name string, a ...float64) {
		t.Helper()
		if _, err := m[name](nums(a...)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	call("emitterrate", float64(id), 5000)
	call("emitterlife", float64(id), 1, 3)
	call("emitterspeed", float64(id), 7, 0.25)
	call("emittergravity", float64(id), 0, -9.8, 2)
	call("emitterwind", float64(id), 1, 0, 0)
	call("emitterdrag", float64(id), 0.4)
	call("emittercone", float64(id), 40)
	call("emitterspread", float64(id), 60) // alias by name
	call("emittercolor", float64(id), 255, 128, 0, 1, 0, 0, 255, 0)
	call("emittersize", float64(id), 0.3, 0.1)
	call("emittermax", float64(id), 200000)
	call("emitterburst", float64(id), 500)
	call("emitterturbulence", float64(id), 2, 0.3, 1)
	call("emitterground", float64(id), 1, 0.5, 0.4, 0.1)
	if _, err := m["emitterblend"]([]value.Value{value.Num(float64(id)), value.Str("add")}); err != nil {
		t.Fatal(err)
	}
	if _, err := m["emitterblend"]([]value.Value{value.Num(float64(id)), value.Str("sparkly")}); err == nil {
		t.Fatal("EmitterBlend should reject an unknown mode")
	}
	if _, err := m["emittershape"]([]value.Value{value.Num(float64(id)), value.Str("sphere"), value.Num(2)}); err != nil {
		t.Fatal(err)
	}
	e := w.gpuEmitterOf(id)
	switch {
	case e.rate != 5000, e.life0 != 1 || e.life1 != 3, e.speed != 7 || e.speedVar != 0.25:
		t.Fatalf("rate/life/speed %v %v %v %v %v", e.rate, e.life0, e.life1, e.speed, e.speedVar)
	case e.gravY != -9.8 || e.gravZ != 2 || e.windX != 1 || e.drag != 0.4:
		t.Fatalf("forces %v %v %v %v", e.gravY, e.gravZ, e.windX, e.drag)
	case e.cone != 60:
		t.Fatalf("cone %v (EmitterSpread alias)", e.cone)
	case e.max != 200000 || e.pendBurst != 500:
		t.Fatalf("max %d burst %d", e.max, e.pendBurst)
	case !e.additive || !e.ground || e.groundY != 0.5 || e.turb != 2:
		t.Fatalf("blend/ground/turb %v %v %v %v", e.additive, e.ground, e.groundY, e.turb)
	case e.shape != 2 || e.shapeX != 2:
		t.Fatalf("shape %d %v", e.shape, e.shapeX)
	case e.c0[0] != 1 || e.c1[2] != 1 || e.c1[3] != 0:
		t.Fatalf("colours %v %v", e.c0, e.c1)
	}

	// A CPU emitter in the same world keeps its own setter.
	cpu := w.newEmitter(0, false)
	call("emitterrate", float64(cpu), 3)
	if w.emitterOf(cpu).rate != 3 {
		t.Fatal("CPU EmitterRate did not reach the CPU emitter")
	}
	if e.rate != 5000 {
		t.Fatal("CPU EmitterRate changed the GPU emitter")
	}
	v, _ := m["isgpuemitter"](nums(float64(id)))
	c, _ := m["isgpuemitter"](nums(float64(cpu)))
	if v.Num != 1 || c.Num != 0 {
		t.Fatalf("IsGPUEmitter %v %v", v.Num, c.Num)
	}
	call("freeemitter", float64(id))
	if w.gpuEmitterOf(id) != nil || w.ents[id] != nil {
		t.Fatal("FreeEmitter left the GPU emitter or its entity")
	}
}

func TestGPUEmitterSpawnRing(t *testing.T) {
	e := defaultGPUEmitter(100)
	e.rate = 600 // 10 a frame at 60 Hz
	for i := 0; i < 3; i++ {
		e.advance(1.0/60, true)
	}
	start, count, spread := e.planSpawn()
	if start != 0 || count != 30 || math.Abs(spread-0.05) > 1e-9 {
		t.Fatalf("plan %d %d %v", start, count, spread)
	}
	e.burst(95)
	start, count, spread = e.planSpawn()
	if start != 30 || count != 95 || spread != 0 {
		t.Fatalf("burst plan %d %d %v (a burst pops at once)", start, count, spread)
	}
	if e.head != 25 {
		t.Fatalf("ring head %d, want wrap to 25", e.head)
	}
	e.burst(1000)
	if _, count, _ = e.planSpawn(); count != 100 {
		t.Fatalf("a burst over max spawns max, got %d", count)
	}
	// Hidden emitters owe nothing; a non-looping duration stops.
	e.advance(1, false)
	if e.pendRate != 0 {
		t.Fatal("hidden emitter spawned")
	}
	e.duration, e.looping = 0.5, false
	e.advance(1, true)
	if e.emitting {
		t.Fatal("duration without loop should stop emitting")
	}
}

func TestGPUEmitterAliveEstimate(t *testing.T) {
	e := defaultGPUEmitter(50)
	e.life0, e.life1 = 1, 2
	e.spawns = []gpuSpawnMark{{at: 0, count: 30}, {at: 1.5, count: 30}}
	e.clock = 2.5
	if n := e.alive(); n != 30 {
		t.Fatalf("alive %d, want the 30 younger than lifeMax", n)
	}
	e.clock = 1.9
	e.spawns = []gpuSpawnMark{{at: 0, count: 30}, {at: 1.5, count: 30}}
	if n := e.alive(); n != 50 {
		t.Fatalf("alive %d, want capped at max", n)
	}
}

func TestGPUSimParamsPacking(t *testing.T) {
	e := defaultGPUEmitter(10)
	e.vx, e.vy, e.vz = 0, 0, 2
	e.gravX, e.gravY, e.gravZ = 0, -10, 1
	e.windX, e.windZ = 2, 3
	e.cone = 90
	e.speed = 0
	e.life0, e.life1 = 3, 1
	e.areaX, e.areaY, e.areaZ = 1, 2, 3
	p := e.simParams(0.016, 4, 5, 0.01, 0.5, 0, 0.25)
	if p.Dir != [3]float32{0, 0, -1} {
		t.Fatalf("dir %v: script +z is scene -z", p.Dir)
	}
	if p.Accel != [3]float32{2.5, -10, -4.25} {
		t.Fatalf("accel %v", p.Accel)
	}
	if math.Abs(float64(p.ConeHalf)-math.Pi/4) > 1e-6 {
		t.Fatalf("cone half %v", p.ConeHalf)
	}
	if p.Speed != 1 {
		t.Fatalf("speed 0 should act as 1, like the CPU emitter: %v", p.Speed)
	}
	if p.Life != [2]float32{1, 3} {
		t.Fatalf("life %v (min, max)", p.Life)
	}
	if p.Shape != 0 || p.ShapeSize != [3]float32{1, 2, 3} {
		t.Fatalf("point shape jitters by EmitterArea: %d %v", p.Shape, p.ShapeSize)
	}
	if p.SpawnStart != 4 || p.SpawnCount != 5 || p.MaxCount != 10 {
		t.Fatalf("spawn %d %d %d", p.SpawnStart, p.SpawnCount, p.MaxCount)
	}
	if p.Turb[3] != 0 || p.Ground[0] != 0 {
		t.Fatal("turbulence and ground start off")
	}
	e.applyShape("sphere", nil)
	if p = e.simParams(0, 0, 0, 0, 0, 0, 0); p.Shape != 2 || p.ShapeSize[0] != 1 {
		t.Fatalf("sphere radius falls back to EmitterArea x: %v", p.ShapeSize)
	}
	e.applyShape("box", nums(0, 0, 4, 5, 6))
	if p = e.simParams(0, 0, 0, 0, 0, 0, 0); p.Shape != 0 || p.ShapeSize != [3]float32{4, 5, 6} {
		t.Fatalf("box %d %v", p.Shape, p.ShapeSize)
	}
	e.applyShape("streak", nums(0, 0, 0.2))
	if e.style != 1 || e.streak != 0.2 {
		t.Fatalf("streak %d %v", e.style, e.streak)
	}
	e.applyShape("snow", nil)
	if e.style != 2 || e.shape != 1 {
		t.Fatal("a look keeps the spawn shape")
	}
	e.ground, e.groundY, e.bounce = true, 1.5, 2
	e.turb = 1
	p = e.simParams(0, 0, 0, 0, 0, 0, 0)
	if p.Ground != [4]float32{1, 1.5, 1, float32(e.friction)} || p.Turb[3] != 1 {
		t.Fatalf("ground %v turb %v", p.Ground, p.Turb)
	}
	d := e.drawParams()
	if d.Style != 2 || d.Additive != 0 || d.UseTex != 0 || d.Color0[3] != 1 {
		t.Fatalf("draw %+v", d)
	}
}

func TestGPUWorldShiftMovesParticles(t *testing.T) {
	w, id := gpuTestWorld(t)
	e := w.gpuEmitterOf(id)
	w.noteGPUWorldShift() // first look only records the origin
	b := w.ensureBubble()
	b.originX += 64
	b.originZ += 32
	w.noteGPUWorldShift()
	if e.shift != [3]float32{64, 0, -32} {
		t.Fatalf("shift %v: particles move by -(64, -32) in scene space", e.shift)
	}
	if e.prevOrg != [3]float32{-64, 0, 32} {
		t.Fatalf("prev origin %v", e.prevOrg)
	}
}

func TestGPUParticleShadersGLSL330(t *testing.T) {
	for name, src := range map[string]string{
		"sim vert": mbGPUPartSimVertex, "sim frag": mbGPUPartSimFragment,
		"draw vert": mbGPUPartVertex, "draw frag": mbGPUPartFragment,
	} {
		if !strings.HasPrefix(src, "#version 330 core") {
			t.Errorf("%s must stay #version 330 core", name)
		}
		st := glslang.StageVert
		if strings.Contains(name, "frag") {
			st = glslang.StageFrag
		}
		if r := glslang.Validate(src, st); !r.OK {
			t.Errorf("%s: %s", name, r.Log)
		}
	}
	for _, v := range gpuPartVaryings {
		if !strings.Contains(mbGPUPartSimVertex, "out vec4 "+v+";") {
			t.Errorf("varying %s is not a vec4 out of the sim shader", v)
		}
	}
	if len(gpuPartVaryings)*4 != gpuPartFloats {
		t.Fatalf("%d varyings for a %d-float record", len(gpuPartVaryings), gpuPartFloats)
	}
	for _, u := range []string{"Shift", "Turb", "Ground", "Accel", "SpawnStart", "PrevOrigin"} {
		if !strings.Contains(mbGPUPartSimVertex, "uniform") || !strings.Contains(mbGPUPartSimVertex, " "+u+";") {
			t.Errorf("sim shader lacks uniform %s", u)
		}
	}
	if !strings.Contains(mbGPUPartFragment, "FogMode") {
		t.Error("GPU particles must fog like the scene")
	}
	if programUnits["mbgpupart"].samplers["PartTex"] != unitParticle {
		t.Error("PartTex is not registered in programUnits")
	}
}

// With glslangValidator on PATH (Vulkan SDK), check the GLSL as OpenGL 3.3
// source for real: no -V, since these are GL programs, not Vulkan ones.
func TestGPUParticleShadersCompile(t *testing.T) {
	exe, err := exec.LookPath("glslangValidator")
	if err != nil {
		t.Skip("glslangValidator not on PATH")
	}
	dir := t.TempDir()
	for name, src := range map[string]string{
		"sim.vert": mbGPUPartSimVertex, "sim.frag": mbGPUPartSimFragment,
		"draw.vert": mbGPUPartVertex, "draw.frag": mbGPUPartFragment,
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
		out, err := exec.Command(exe, path).CombinedOutput()
		if err != nil {
			t.Errorf("%s: %v\n%s", name, err, out)
		}
	}
}
