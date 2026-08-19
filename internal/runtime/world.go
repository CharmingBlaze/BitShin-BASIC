// Package runtime implements Blitz-style commands on G3N (3D) and Ebiten (2D).
package runtime

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/light"
	"github.com/g3n/engine/loader/collada"
	"github.com/g3n/engine/loader/gltf"
	"github.com/g3n/engine/loader/obj"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/renderer"
	"github.com/g3n/engine/texture"
	"github.com/g3n/engine/window"
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"

	bsaudio "bitshinbasic/internal/audio"
	"bitshinbasic/internal/interp"
	"bitshinbasic/internal/netenet"
	"bitshinbasic/internal/phys2d"
	"bitshinbasic/internal/phys3d"
	"bitshinbasic/internal/syntax"
	"bitshinbasic/internal/value"
)

type Entity struct {
	node                               core.INode
	mesh                               *graphic.Mesh
	mat                                *material.Standard
	cam                                *camera.Camera
	lgt                                interface{ SetColor(color *math32.Color) }
	lgtI                               interface{ SetIntensity(float32) }
	pbr                                *material.Physical
	pbrWrap                            *pbrMat
	usePBR                             bool
	tint                               math32.Color4
	etype                              int
	radius                             float32
	pitch                              float32
	yaw                                float32
	roll                               float32
	parent                             int
	name                               string
	kind                               string // cube, sphere, mesh, …
	src                                string // LoadMesh path
	collided                           []int
	hitID                              int
	hitX                               float32
	hitY                               float32
	hitZ                               float32
	bodyType                           int // 0 none, 1 dynamic, 2 static, 3 kinematic
	cloth                              bool
	buoyancy                           float32 // 0 auto 1.1 in water, <0 off
	boxX, boxY, boxZ                   float32
	driveVX, driveVY, driveVZ          float32
	lastDriveX, lastDriveY, lastDriveZ float32
	driveTrack                         bool
	vpX, vpY, vpW, vpH                 int
	anim                               *animState
	lgtKind                            int // 0 ambient/none, 1 directional, 2 point, 3 spot
	castShadow                         bool
	camFollowed                        bool
	meshNoCast                         bool
	meshNoRecv                         bool
	albedoTex                          *texture.Texture2D
	shadowRes                          int
	shadowGeomDirty                    bool // vertex/morph edits: skip shadow cache until redrawn
	sky                                bool
	windSway                           bool
	windPhase                          float32
	onClick                            string
	onChange                           string
	ushader                            int
	hitbox                             bool
	collKind                           int // 0 none, 1 box, 2 sphere, 3 capsule, 4 cylinder, 5 sensor, 6 compound
}

type texSlot struct {
	tex  *texture.Texture2D
	path string
}

type collideRule struct {
	src, dest, method, response int
}

type World struct {
	app                    *g3nHost
	scene                  *core.Node
	cam                    *camera.Camera
	ambient                *light.Ambient
	ents                   map[int]*Entity
	nextID                 int
	title                  string
	clear                  math32.Color
	base                   string
	keys                   map[int]bool
	prev                   map[int]bool
	hits                   map[int]bool
	mouse                  [8]bool
	mx, my                 float32
	delta                  float64
	started                time.Time
	lastFlip               time.Time
	fogMode                int
	fogRGB                 math32.Color
	fogNear                float32
	fogFar                 float32
	fogDensity             float32
	texts                  []*gui.Label
	hudLines               []string
	hudLabs                []*gui.Label
	textRGB                math32.Color
	texs                   map[int]*texSlot
	nextTex                int
	rules                  []collideRule
	wire                   bool
	ready                  bool
	presented              bool
	presentOK              bool
	escapeLatch            bool
	heldAtFlip             map[int]bool
	closeReady             bool
	loopFrames             int
	quit                   bool
	shadow                 shadowMap
	shaderUnis             map[string]shaderUni
	tweens                 []nodeTween
	cmdMap                 map[string]cmd
	phys3                  phys3d.World
	vehCtrls               map[int]*vehCtrl
	ropes                  map[int]*ropeSystem
	nextRope               int
	phys2                  *phys2d.Space
	net                    netenet.Host
	nets                   map[int]netenet.Host
	netQ                   map[int][]netenet.Event
	nextNet                int
	netInbox               []netenet.Event
	netMsg                 string
	netPeer                int
	netKind                int
	netIP                  string
	sess                   *netSess
	mode2D                 bool
	scrW                   int
	scrH                   int
	images                 map[int]*ebiImage
	sprites                map[int]*ebiSprite
	draws                  []drawOp
	drawRGB                [3]uint8
	clsRGB                 [3]uint8
	nextImg                int
	prevMX                 float32
	prevMY                 float32
	mxs                    float32
	mys                    float32
	mouseInited            bool
	waitHold               bool
	delayUntil             time.Time
	timers                 map[int]*blitzTimer
	nextTimer              int
	namedT                 map[string]*namedTimer
	pickID                 int
	pickX, pickY, pickZ    float32
	tformX, tformY, tformZ float32
	sounds                 map[int]*sndSlot
	nextSnd                int
	listenEnt              int
	musicID                int
	mouseHits              [8]bool
	mz                     float32
	prevMZ                 float32
	mzs                    float32
	clipLocal              string
	guiReady               bool
	guiFrame               bool
	guiUsed                bool
	guiWantM               bool
	guiWantK               bool
	guiChars               []rune
	guiCol                 [3]float32
	guiPrevKeys            map[window.Key]bool
	imgFilter              int // 0 nearest, 1 linear
	shotPath               string
	emitters               map[int]*emitter
	nextEmit               int
	partGeom               *geometry.Geometry
	partPool               []*partVis
	skies                  map[int]*skySlot
	nextSky                int
	skyID                  int
	wx                     weatherState
	thunderClip            *bsaudio.Clip
	thunderVoice           *bsaudio.Voice
	rainProg               uint32
	rainVAO, rainVBO       uint32
	rainBlitOK             bool
	tiles                  map[int]*tileMap
	nextTile               int
	fonts                  map[int]*fontSlot
	nextFont               int
	curFont                int
	pak                    *pakFS
	runner                 *interp.Interp
	gpDead                 float64
	onResize               string
	onKeyFn                string
	onMouseFn              string
	ecs                    ecsHost
	docs                   map[int]*dataDoc
	freeDocs               []int
	nextDoc                int
	pools                  map[int]*userPool
	freePools              []int
	nextPool               int
	banks                  map[int]*bankSlot
	freeBanks              []int
	nextBank               int
	freeIDs                []int
	navs                   map[int]*navMesh
	freeNavs               []int
	nextNav                int
	curNav                 int
	agents                 map[int]*navAgent
	navStepped             bool
	grids                  map[int]*gridMap
	freeGrids              []int
	nextGrid               int
	paths                  map[int]*gridPath
	freePaths              []int
	nextPath               int
	guiSlide               map[string]*float32
	guiCheck               map[string]*bool
	guiInput               map[string]*string
	guiWinOpen             bool
	wins                   map[int]*extraWin
	freeWins               []int
	nextWin                int
	renderWin              int
	focusWin               int
	winBlitProg            uint32
	jobs                   *jobPool
	jobWorkers             int
	stream                 *worldStream
	instances              map[int]*instancedMesh
	probes                 map[int]*lightProbe
	freeProbes             []int
	nextProbe              int
	terrains               map[int]*terrain
	freeTerrains           []int
	nextTerrain            int
	curTerrain             int
	hmaps                  map[int]*heightMap
	freeHMaps              []int
	nextHMap               int
	curHMap                int
	geo                    geoFrame
	waters                 map[int]*waterBody
	cloths                 map[int]*clothSheet
	grabs                  map[int]grabHold
	projectiles            map[int]*projFly
	pathFollows            map[int]*pathFollow
	physDebug              bool
	physDebugMesh          *graphic.Mesh
	physHitDebugMesh       *graphic.Mesh
	beams                  map[int]*beamLink
	boneAttaches           []boneAttach
	freeWaters             []int
	nextWater              int
	curWater               int
	crowds                 map[int]*crowd
	freeCrowds             []int
	nextCrowd              int
	curCrowd               int
	physAsync              bool
	physThreads            int
	navMaxSlope            float32
	pbrLib                 map[int]*pbrMat
	nextPBR                int
	iblOn                  bool
	iblIntensity           float32
	glmod                  glModern
	clouds                 map[int]*cloudLayer
	freeClouds             []int
	nextCloud              int
	curCloud               int
	skyTop                 math32.Color
	skyBot                 math32.Color
	skyProc                bool
	skySun                 math32.Vector3
	skySunOK               bool
	atmo                   *atmoDome
	fogHeight              float32
	fogHFall               float32
	wetness                float32
	post                   postFX
	ushaders               map[int]*userShader
	freeUSh                []int
	nextUSh                int
	timeScale              float32
	shakeTrauma            float32
	shakeDuration          float32
	shakeElapsed           float32
	shakeFreq              float32
	fpsControllers         map[int]*fpsCtrl
	tpsControllers         map[int]*tpsCtrl
	topDownControllers     map[int]*topDownCtrl
	platformerControllers  map[int]*platformerCtrl
	actTweens              []*activeTween
}

type blitzTimer struct {
	hz       float64
	start    time.Time
	consumed int64
}

type namedTimer struct {
	interval time.Duration
	next     time.Time
}

type sndSlot struct {
	path  string
	clip  *bsaudio.Clip
	voice *bsaudio.Voice
	vol   float64
	pitch float64
	music bool
}

func New(base string) *World {
	return &World{
		ents:                  map[int]*Entity{},
		texs:                  map[int]*texSlot{},
		keys:                  map[int]bool{},
		prev:                  map[int]bool{},
		hits:                  map[int]bool{},
		heldAtFlip:            map[int]bool{},
		nextID:                1,
		nextTex:               1,
		scene:                 core.NewNode(),
		shadow:                shadowMap{on: true, size: 2048, extent: 32, distance: 250, zPad: 60, cascades: 4, pcf: 3, bias: 0.0025, normalBias: 1, lightSize: 0.04, evsmC: 8, fadeNear: 220, fadeFar: 250, primed: map[*geometry.Geometry]bool{}},
		shaderUnis:            map[string]shaderUni{},
		title:                 "BitShin BASIC",
		clear:                 math32.Color{0.15, 0.16, 0.2},
		textRGB:               math32.Color{1, 1, 1},
		fogRGB:                math32.Color{0.6, 0.65, 0.75},
		fogNear:               10,
		fogFar:                100,
		fogDensity:            0.025,
		delta:                 1.0 / 60.0,
		started:               time.Now(),
		base:                  base,
		images:                map[int]*ebiImage{},
		sprites:               map[int]*ebiSprite{},
		nextImg:               1,
		nextTimer:             1,
		nextSnd:               1,
		timers:                map[int]*blitzTimer{},
		namedT:                map[string]*namedTimer{},
		sounds:                map[int]*sndSlot{},
		emitters:              map[int]*emitter{},
		skies:                 map[int]*skySlot{},
		nextSky:               1,
		tiles:                 map[int]*tileMap{},
		fonts:                 map[int]*fontSlot{},
		nextEmit:              1,
		nextTile:              1,
		nextFont:              1,
		imgFilter:             1,
		drawRGB:               [3]uint8{255, 255, 255},
		clsRGB:                [3]uint8{20, 22, 32},
		scrW:                  800,
		scrH:                  600,
		guiPrevKeys:           map[window.Key]bool{},
		gpDead:                0.15,
		sess:                  newNetSess(),
		nets:                  map[int]netenet.Host{},
		netQ:                  map[int][]netenet.Event{},
		nextNet:               1,
		docs:                  map[int]*dataDoc{},
		nextDoc:               1,
		pools:                 map[int]*userPool{},
		nextPool:              1,
		banks:                 map[int]*bankSlot{},
		nextBank:              1,
		navs:                  map[int]*navMesh{},
		nextNav:               1,
		agents:                map[int]*navAgent{},
		grids:                 map[int]*gridMap{},
		nextGrid:              1,
		paths:                 map[int]*gridPath{},
		nextPath:              1,
		guiSlide:              map[string]*float32{},
		guiCheck:              map[string]*bool{},
		guiInput:              map[string]*string{},
		wins:                  map[int]*extraWin{},
		nextWin:               1,
		instances:             map[int]*instancedMesh{},
		probes:                map[int]*lightProbe{},
		nextProbe:             1,
		terrains:              map[int]*terrain{},
		nextTerrain:           1,
		hmaps:                 map[int]*heightMap{},
		nextHMap:              1,
		waters:                map[int]*waterBody{},
		nextWater:             1,
		crowds:                map[int]*crowd{},
		nextCrowd:             1,
		pbrLib:                map[int]*pbrMat{},
		nextPBR:               1,
		iblOn:                 true,
		iblIntensity:          1,
		clouds:                map[int]*cloudLayer{},
		nextCloud:             1,
		skyTop:                math32.Color{0.525, 0.735, 0.84},
		skyBot:                math32.Color{0.9, 0.9, 0.95},
		ushaders:              map[int]*userShader{},
		nextUSh:               1,
		post:                  postFX{exposure: 1, contrast: 1, sat: 1, tint: math32.Color{1, 1, 1}},
		timeScale:             1.0,
		fpsControllers:        map[int]*fpsCtrl{},
		tpsControllers:        map[int]*tpsCtrl{},
		topDownControllers:    map[int]*topDownCtrl{},
		platformerControllers: map[int]*platformerCtrl{},
	}
}

func (w *World) Bind(in *interp.Interp) { w.runner = in }

func (w *World) HasGraphics() bool { return w.ready }

func (w *World) HasPresented() bool { return w.presented }

func (w *World) Yields(name string) bool {
	switch name {
	case "flip", "waittimer", "waitkey", "delay":
		return true
	}
	return false
}

func (w *World) Holds() bool { return w.waitHold }

func (w *World) Call(name string, args []value.Value) (value.Value, error) {
	if v, ok := namedKeys[name]; ok {
		return value.Num(float64(v)), nil
	}
	if s, ok := syntax.WeatherConstants[name]; ok {
		return value.Str(s), nil
	}
	if n, ok := syntax.NetConstants[name]; ok {
		return value.Num(float64(n)), nil
	}
	if w.cmdMap == nil {
		w.cmdMap = w.commandTable()
	}
	fn, ok := w.cmdMap[name]
	if !ok {
		return value.Value{}, fmt.Errorf("unknown command %s", name)
	}
	args = syntax.ExpandCommandArgs(name, args)
	v, err := fn(args)
	if err != nil {
		return v, err
	}
	if syntax.ReturnsEntity(name) && len(args) > 0 {
		return args[0], nil
	}
	return v, nil
}

type cmd func([]value.Value) (value.Value, error)

func (w *World) require() error {
	if !w.ready {
		return fmt.Errorf("Graphics3D or Graphics2D must be called first")
	}
	return nil
}

func (w *World) require3D() error {
	if !w.ready || w.mode2D {
		return fmt.Errorf("Graphics3D must be called first")
	}
	return nil
}

func (w *World) graphics3D(width, height, depth, mode int) (value.Value, error) {
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}
	host, err := startG3N(w.title, width, height)
	if err != nil {
		return value.Value{}, err
	}
	w.app = host
	if gw, ok := w.app.IWindow.(*window.GlfwWindow); ok {
		gw.SetSize(width, height)
		gw.SetTitle(w.title)
	}
	w.scene = core.NewNode()
	gui.Manager().Set(w.scene)
	w.ambient = light.NewAmbient(&math32.Color{0.28, 0.30, 0.36}, 1)
	w.scene.Add(w.ambient)
	w.app.Gls().ClearColor(w.clear.R, w.clear.G, w.clear.B, 1)
	w.app.Subscribe(window.OnWindowSize, func(evname string, ev interface{}) {
		ww, hh := w.app.GetSize()
		w.scrW, w.scrH = ww, hh
		if ww > 0 && hh > 0 {
			gl.Viewport(0, 0, int32(ww), int32(hh))
		}
		if w.cam != nil {
			w.cam.SetAspect(float32(ww) / float32(hh))
		}
		w.fireHook(w.onResize, value.Num(float64(ww)), value.Num(float64(hh)))
	})
	w.app.Subscribe(window.OnKeyDown, func(evname string, ev interface{}) {
		ke := ev.(*window.KeyEvent)
		w.setKey(ke.Key, true)
		w.guiForwardKey(ke.Key, ke.Mods, true)
		w.fireHook(w.onKeyFn, value.Num(float64(w.blitzKey(ke.Key))), value.Num(1))
	})
	w.app.Subscribe(window.OnKeyUp, func(evname string, ev interface{}) {
		ke := ev.(*window.KeyEvent)
		w.setKey(ke.Key, false)
		w.guiForwardKey(ke.Key, ke.Mods, false)
	})
	w.app.Subscribe(window.OnMouseDown, func(evname string, ev interface{}) {
		e := ev.(*window.MouseEvent)
		b := int(e.Button) + 1
		if b >= 0 && b < len(w.mouse) {
			w.mouse[b] = true
			w.mouseHits[b] = true
		}
		w.guiForwardMouse(int(e.Button), true)
	})
	w.app.Subscribe(window.OnMouseUp, func(evname string, ev interface{}) {
		e := ev.(*window.MouseEvent)
		w.mouse[int(e.Button)+1] = false
		w.guiForwardMouse(int(e.Button), false)
	})
	w.app.Subscribe(window.OnCursor, func(evname string, ev interface{}) {
		e := ev.(*window.CursorEvent)
		w.mx, w.my = e.Xpos, e.Ypos
		w.fireHook(w.onMouseFn, value.Num(float64(e.Xpos)), value.Num(float64(e.Ypos)))
	})
	w.app.Subscribe(window.OnScroll, func(evname string, ev interface{}) {
		e := ev.(*window.ScrollEvent)
		w.mz += e.Yoffset
		w.guiForwardScroll(float64(e.Xoffset), float64(e.Yoffset))
	})
	w.app.Subscribe(window.OnChar, func(evname string, ev interface{}) {
		e := ev.(*window.CharEvent)
		w.guiChars = append(w.guiChars, e.Char)
	})
	if mode == 1 {
		if gw, ok := w.app.IWindow.(*window.GlfwWindow); ok {
			gw.SetFullscreen(true)
		}
	}
	w.ready = true
	w.mode2D = false
	w.scrW, w.scrH = width, height
	w.focusWin = 0
	w.renderWin = 0
	if gw, ok := w.app.IWindow.(*window.GlfwWindow); ok {
		gw.SetShouldClose(false)
		gw.SetFocusCallback(func(_ *glfw.Window, focused bool) {
			if focused {
				w.focusWin = 0
			}
		})
	}
	w.loopFrames = 0
	w.app.beforeDestroy = w.destroyExtraWindows
	w.detectModernGL()
	_ = depth
	// Blitz3D-style: shadows are on after Graphics3D. Authors call EnableShadows False.
	w.ensureShadowOn()
	if w.runner != nil {
		w.runner.MarkLive()
	}
	return value.Num(0), nil
}

func (w *World) setKey(k window.Key, down bool) {
	// Window creation / first focus often injects phantom Escape and Space.
	// Recording them before Flip makes While Not KeyDown(1) quit and
	// KeyDown(KEY_SPACE) grip on the first presented frames.
	if !w.presented && (k == window.KeyEscape || k == window.KeySpace) {
		return
	}
	if k == window.KeyEscape && w.escapeLatch {
		if !down {
			w.escapeLatch = false
			w.clearHeldKey(KeyEscape)
		}
		return
	}
	for code, gk := range dikToKey {
		if gk != k {
			continue
		}
		if w.applyHeldLatch(code, down) {
			continue
		}
		w.keys[code] = down
		if down {
			w.hits[code] = true
		}
	}
	if k == window.KeyEscape {
		if w.applyHeldLatch(KeyEscape, down) {
			return
		}
		w.keys[KeyEscape] = down
		if down {
			w.hits[KeyEscape] = true
		}
	}
}

func (w *World) applyHeldLatch(code int, down bool) bool {
	if w.heldAtFlip == nil || !w.heldAtFlip[code] {
		return false
	}
	if !down {
		w.clearHeldKey(code)
	}
	return true
}

func (w *World) clearHeldKey(code int) {
	if w.heldAtFlip != nil {
		delete(w.heldAtFlip, code)
	}
	w.keys[code] = false
	w.hits[code] = false
}

// latchHeldKeys ignores keys that were already down at first present
// until each one is released. Matches the Escape phantom-focus fix.
func (w *World) latchHeldKeys() {
	if w.heldAtFlip == nil {
		w.heldAtFlip = map[int]bool{}
	}
	for code, down := range w.keys {
		if !down {
			continue
		}
		w.heldAtFlip[code] = true
		w.keys[code] = false
		w.hits[code] = false
	}
	if gw := w.glfwWin(); gw != nil {
		for code, gk := range dikToKey {
			if gw.GetKey(glfw.Key(gk)) != glfw.Press {
				continue
			}
			w.heldAtFlip[code] = true
			w.keys[code] = false
			w.hits[code] = false
		}
	}
}

func (w *World) tickKeys() {
	// rising-edge hits already recorded on key down; clear unused? KeyHit consumes.
	w.prev = map[int]bool{}
	for k, v := range w.keys {
		w.prev[k] = v
	}
}

func (w *World) keyDown(code int) bool {
	if w.heldAtFlip[code] {
		return false
	}
	// Escape must work after Flip even if imgui wants the keyboard.
	if code == KeyEscape || code == KeySpace {
		if !w.presented {
			return false
		}
		if code == KeyEscape {
			return w.keys[code]
		}
	}
	if w.guiCapturesKeyboard() {
		return false
	}
	if w.keys[code] {
		return true
	}
	return w.pollHeldKey(code)
}

// windowWantsClose is WindowShouldClose(). Phantom GLFW close from
// window creation is swallowed until close has been seen false after
// the first Flip (or a few frames). A later X click still quits.
func (w *World) windowWantsClose() bool {
	gw := w.glfwWin()
	if gw == nil {
		return false
	}
	closing := gw.ShouldClose()
	if !w.presented {
		if closing {
			gw.SetShouldClose(false)
		}
		return false
	}
	if !w.closeReady {
		if closing {
			gw.SetShouldClose(false)
		} else {
			w.closeReady = true
		}
		if w.loopFrames >= 12 {
			if gw.ShouldClose() {
				gw.SetShouldClose(false)
			}
			w.closeReady = true
		}
		return false
	}
	return closing
}

func (w *World) keyHit(code int) bool {
	if w.heldAtFlip[code] {
		return false
	}
	if code == KeyEscape || code == KeySpace {
		if !w.presented {
			w.hits[code] = false
			return false
		}
		if code == KeySpace {
			if w.guiCapturesKeyboard() {
				return false
			}
			if w.hits[code] {
				w.hits[code] = false
				return true
			}
			return false
		}
		if w.hits[code] {
			w.hits[code] = false
			return true
		}
		return false
	}
	if w.guiCapturesKeyboard() {
		return false
	}
	if w.hits[code] {
		w.hits[code] = false
		return true
	}
	return false
}

func (w *World) addEntity(e *Entity, parent int) int {
	id := w.takeHandle(&w.freeIDs, &w.nextID)
	w.ents[id] = e
	e.radius = 1
	e.boxX, e.boxY, e.boxZ = 1, 1, 1
	e.tint = math32.Color4{1, 1, 1, 1}
	e.parent = parent
	p := w.parentNode(parent)
	if p != nil && p.GetNode() != nil && e.node != nil {
		p.GetNode().Add(e.node)
	}
	return id
}

func (w *World) parentNode(id int) core.INode {
	if id == 0 {
		return w.scene
	}
	if e := w.ents[id]; e != nil {
		return e.node
	}
	return w.scene
}

func (w *World) ent(id int) (*Entity, error) {
	e := w.ents[id]
	if e == nil {
		return nil, fmt.Errorf("invalid entity %d", id)
	}
	return e, nil
}

func (w *World) nodeOf(id int) (*core.Node, error) {
	e, err := w.ent(id)
	if err != nil {
		return nil, err
	}
	return e.node.GetNode(), nil
}

func toG3N(x, y, z float32) (float32, float32, float32) {
	return x, y, -z
}

func fromG3N(x, y, z float32) (float32, float32, float32) {
	return x, y, -z
}

// rgb converts script RGB to G3N 0–1. Canonical scale is 0–255; if every
// channel is ≤ 1 the values are treated as already normalized 0–1.
func rgb(r, g, b float64) *math32.Color {
	if r > 1 || g > 1 || b > 1 {
		return &math32.Color{float32(r) / 255, float32(g) / 255, float32(b) / 255}
	}
	return &math32.Color{float32(r), float32(g), float32(b)}
}

func rgbBytes(r, g, b float64) [3]uint8 {
	c := rgb(r, g, b)
	return [3]uint8{u8f(float64(c.R)), u8f(float64(c.G)), u8f(float64(c.B))}
}

func (w *World) newMat() *material.Standard {
	m := material.NewStandard(&math32.Color{0.82, 0.84, 0.88})
	m.SetShininess(8)
	m.SetSpecularColor(&math32.Color{0.11, 0.11, 0.11})
	m.SetEmissiveColor(&math32.Color{0, 0, 0})
	if w.shadow.on || w.fogMode != 0 {
		m.SetShader("bsshadow")
	}
	if w.wire {
		m.SetWireframe(true)
	}
	return m
}

func (w *World) meshEnt(geom *geometry.Geometry, parent int) int {
	mat := w.newMat()
	mesh := graphic.NewMesh(geom, newLitMat(w, mat))
	mesh.SetCullable(false)
	id := w.addEntity(&Entity{node: mesh, mesh: mesh, mat: mat}, parent)
	if geom != nil {
		if w.shadow.primed == nil {
			w.shadow.primed = map[*geometry.Geometry]bool{}
		}
		w.shadow.primed[geom] = true
	}
	return id
}

func (w *World) Loop(in *interp.Interp) error {
	if w.mode2D {
		return w.loop2D(in)
	}
	first := true
	w.app.onFirstPresent = func() {
		w.latchHeldKeys()
		w.drainPhantomQuit()
		if w.heldAtFlip[KeyEscape] || w.keys[KeyEscape] {
			w.escapeLatch = true
			w.clearHeldKey(KeyEscape)
			w.heldAtFlip[KeyEscape] = true
		}
		w.presentOK = true
	}
	w.app.Run(func(rend *renderer.Renderer, dt time.Duration) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Flip: panic", r)
			}
		}()
		w.tickDelta(dt)
		w.noteMouse()
		w.loopFrames++
		if !w.presentOK {
			w.drainPhantomQuit()
		}
		if !first {
			w.clearFrameText()
			if err := in.Run(); err != nil {
				fmt.Println(err)
				if w.presented {
					in.ClearLiveError()
				}
			}
		}
		first = false
		if w.windowWantsClose() {
			w.quit = true
		}
		if w.quit || in.Done() {
			w.app.Exit()
			return
		}
		w.tickFX()
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("RenderWorld: panic", r)
				}
			}()
			w.render(rend)
		}()
	})
	return in.Err()
}

func (w *World) RenderOnce() {
	if w.mode2D {
		_ = w.loop2D(nil)
		return
	}
	if w.app == nil {
		return
	}
	w.app.Run(func(rend *renderer.Renderer, dt time.Duration) {
		w.tickDelta(dt)
		w.render(rend)
		w.app.Exit()
	})
}

// tickDelta sets World.delta to seconds since the last Flip (or last frame).
// Both the G3N and Ebiten loops call this before resuming the interpreter.
func (w *World) tickDelta(hint time.Duration) {
	now := time.Now()
	switch {
	case !w.lastFlip.IsZero():
		w.delta = now.Sub(w.lastFlip).Seconds()
	case hint > 0:
		w.delta = hint.Seconds()
	default:
		w.delta = 1.0 / 60.0
	}
	if w.delta < 1e-6 {
		w.delta = 1e-6
	}
	if w.delta > 0.25 {
		w.delta = 0.25
	}
	w.lastFlip = now
	w.navStepped = false
	w.ecs.stepped = false
}

func (w *World) markFlip() {
	w.lastFlip = time.Now()
	if !w.presented {
		w.keys[KeyEscape] = false
		w.hits[KeyEscape] = false
		w.keys[KeySpace] = false
		w.hits[KeySpace] = false
	}
	w.presented = true
}

// drainPhantomQuit drops creation leftovers (Ebiten GLFW class unregister /
// first-focus Escape / GLFW ShouldClose) until the first SwapBuffers+poll.
// After presentOK it must not run — that trapped the window X.
func (w *World) drainPhantomQuit() {
	if w.presentOK {
		return
	}
	w.keys[KeyEscape] = false
	w.hits[KeyEscape] = false
	w.keys[KeySpace] = false
	w.hits[KeySpace] = false
	if gw := w.glfwWin(); gw != nil && gw.ShouldClose() {
		gw.SetShouldClose(false)
	}
}

func (w *World) clearFrameText() {
	for _, t := range w.texts {
		if t == nil {
			continue
		}
		if p := t.Parent(); p != nil {
			p.GetNode().Remove(t)
		}
		t.SetVisible(false)
	}
	w.texts = w.texts[:0]
}

func (w *World) tickFX() {
	dt := float32(w.delta)
	if dt <= 0 {
		dt = 1.0 / 60.0
	}
	if w.jobs != nil {
		w.jobs.flushGL()
	}
	w.tickStream()
	w.tickTerrain()
	w.tickWater()
	w.tickClouds(dt)
	w.tickInstances()
	w.applyProbes()
	w.tickAnims(dt)
	w.tickWeather(dt)
	w.tickEmitters(dt)
	w.syncSkyboxes()
	w.netUpdate()
	w.updateNav()
	w.tickCrowds()
	w.tickClouds(dt)
	w.ecsProgress(float64(dt))
}

func (w *World) render(rend *renderer.Renderer) {
	w.syncSkyboxes()
	cr, cg, cb := w.clear.R, w.clear.G, w.clear.B
	if w.fogMode != 0 && !w.skyVisible() && !w.atmoVisible() {
		cr, cg, cb = w.fogRGB.R, w.fogRGB.G, w.fogRGB.B
	}
	if fr, fg, fb, ok := w.weatherClearMix(); ok {
		cr, cg, cb = cr+fr, cg+fg, cb+fb
	}
	ww, hh := w.scrW, w.scrH
	if w.app != nil {
		if aw, ah := w.app.GetSize(); aw > 0 && ah > 0 {
			ww, hh = aw, ah
			w.scrW, w.scrH = aw, ah
		}
	}
	if ww <= 0 {
		ww = 800
	}
	if hh <= 0 {
		hh = 600
	}
	w.flushHudPrint()
	usedPost := w.beginPostTarget(ww, hh)
	drainGL("render")
	gl.Viewport(0, 0, int32(ww), int32(hh))
	w.app.Gls().ClearColor(cr, cg, cb, 1)
	w.app.Gls().Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)
	w.renderSceneCams(rend, ww, hh)
	if usedPost {
		w.endPostTarget(w.app.Gls(), ww, hh)
	}
	w.guiPresent()
	w.presentExtraWindows(rend)
}

func (w *World) resolve(path string) string {
	path = filepath.FromSlash(strings.ReplaceAll(path, "\\", "/"))
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(w.base, path)
}

// openPath returns a real filesystem path, extracting from OpenPak if needed.
func (w *World) openPath(rel string) (string, error) {
	disk := w.resolve(rel)
	if st, err := os.Stat(disk); err == nil && !st.IsDir() {
		return disk, nil
	}
	if w.pak != nil {
		return w.pak.extract(rel)
	}
	return "", fmt.Errorf("file not found: %s", rel)
}

func (w *World) ensurePhys2() {
	if w.phys2 == nil {
		w.phys2 = phys2d.New()
	}
}

func worldPos(n *core.Node) math32.Vector3 {
	var p math32.Vector3
	n.WorldPosition(&p)
	return p
}

func worldQuat(n *core.Node) math32.Quaternion {
	q := math32.Quaternion{}
	n.WorldQuaternion(&q)
	return q
}

func (w *World) refreshWorldMatrices() {
	if w.scene != nil {
		w.scene.UpdateMatrixWorld()
	}
}

func (w *World) ensurePhys3() {
	if w.phys3 == nil {
		w.phys3 = phys3d.New()
		w.phys3.EnableContacts()
	}
}

func (w *World) processPhysicsContacts() {
	if w.phys3 == nil {
		return
	}
	evs := w.phys3.PollContacts(512)
	for i := 0; i < len(evs); i++ {
		ev := evs[i]
		if ev.Kind == phys3d.ContactRemoved {
			continue
		}
		if a := w.ents[ev.A]; a != nil && ev.B != 0 {
			a.collided = append(a.collided, ev.B)
			a.hitID = ev.B
			a.hitX, a.hitY, a.hitZ = ev.X, ev.Y, ev.Z
		}
		if b := w.ents[ev.B]; b != nil && ev.A != 0 {
			b.collided = append(b.collided, ev.A)
			if b.hitID == 0 {
				b.hitID = ev.A
				b.hitX, b.hitY, b.hitZ = ev.X, ev.Y, ev.Z
			}
		}
	}
}

func (w *World) syncPhys3Pose() {
	if w.phys3 == nil {
		return
	}
	for id, e := range w.ents {
		if e.parent != 0 {
			continue
		}
		if x, y, z, ok := w.phys3.GetPosition(id); ok {
			gx, gy, gz := toG3N(x, y, z)
			e.node.GetNode().SetPosition(gx, gy, gz)
		}
		if qx, qy, qz, qw, ok := w.phys3.GetRotation(id); ok {
			q := math32.Quaternion{X: qx, Y: qy, Z: -qz, W: qw}
			e.node.GetNode().SetQuaternion(q.X, q.Y, q.Z, q.W)
			r := e.node.GetNode().Rotation()
			e.pitch = r.X * 180 / math32.Pi
			e.yaw = -r.Y * 180 / math32.Pi
			e.roll = -r.Z * 180 / math32.Pi
		}
	}
}

func (w *World) driveParentedBodies(dt float32) {
	if w.phys3 == nil {
		return
	}
	if dt <= 0 {
		dt = 1.0 / 60.0
	}
	for id, e := range w.ents {
		if e == nil || e.parent != 0 || e.node == nil || e.bodyType != 3 {
			continue
		}
		vx, vy, vz, ok := w.phys3.GetVelocity(id)
		if !ok {
			continue
		}
		px, py, pz, ok := w.phys3.GetPosition(id)
		if !ok {
			continue
		}
		gx, gy, gz := toG3N(px+vx*dt, py+vy*dt, pz+vz*dt)
		e.node.GetNode().SetPosition(gx, gy, gz)
		e.node.GetNode().UpdateMatrixWorld()
		e.driveVX, e.driveVY, e.driveVZ = vx, vy, vz
	}
	w.refreshWorldMatrices()
	for id, e := range w.ents {
		if e == nil || e.parent == 0 || e.node == nil {
			continue
		}
		if _, _, _, ok := w.phys3.GetPosition(id); !ok {
			continue
		}
		n := e.node.GetNode()
		p := worldPos(n)
		x, y, z := fromG3N(p.X, p.Y, p.Z)
		wq := worldQuat(n)
		if e.driveTrack {
			e.driveVX = (x - e.lastDriveX) / dt
			e.driveVY = (y - e.lastDriveY) / dt
			e.driveVZ = (z - e.lastDriveZ) / dt
		}
		e.lastDriveX, e.lastDriveY, e.lastDriveZ = x, y, z
		e.driveTrack = true
		w.phys3.MoveKinematic(id, x, y, z, wq.X, wq.Y, -wq.Z, wq.W, dt)
	}
	w.wakeDynamicsNearDrivers()
}

func (w *World) meshPhysPose(e *Entity) (x, y, z, qx, qy, qz, qw float32, ok bool) {
	if e == nil || e.node == nil {
		return 0, 0, 0, 0, 0, 0, 1, false
	}
	n := e.node.GetNode()
	p := worldPos(n)
	x, y, z = fromG3N(p.X, p.Y, p.Z)
	wq := worldQuat(n)
	return x, y, z, wq.X, wq.Y, -wq.Z, wq.W, true
}

func (w *World) wakeDynamicsNearDrivers() {
	if w.phys3 == nil {
		return
	}
	for id, e := range w.ents {
		if e == nil || e.bodyType != 1 {
			continue
		}
		px, py, pz, ok := w.phys3.GetPosition(id)
		if !ok {
			continue
		}
		for _, drv := range w.ents {
			if drv == nil || drv.bodyType == 0 || drv.bodyType == 1 {
				continue
			}
			dx, dy, dz, _, _, _, _, pok := w.meshPhysPose(drv)
			if !pok {
				continue
			}
			reach := drv.boxX + drv.boxY + drv.boxZ + 1.2
			ox, oy, oz := px-dx, py-dy, pz-dz
			if ox*ox+oy*oy+oz*oz <= reach*reach {
				w.phys3.Wake(id)
				break
			}
		}
	}
}

func (w *World) resolveDrivenOverlaps() {
	if w.phys3 == nil {
		return
	}
	w.refreshWorldMatrices()
	type pose struct {
		id             int
		x, y, z        float32
		hx, hy, hz     float32
		qx, qy, qz, qw float32
		vx, vy, vz     float32
	}
	drivers := make([]pose, 0, 8)
	prizes := make([]pose, 0, 16)
	for id, e := range w.ents {
		if e == nil || e.bodyType == 0 {
			continue
		}
		if _, _, _, ok := w.phys3.GetPosition(id); !ok {
			continue
		}
		hx, hy, hz := e.boxX, e.boxY, e.boxZ
		if hx <= 0 {
			hx = 0.5
		}
		if hy <= 0 {
			hy = 0.5
		}
		if hz <= 0 {
			hz = 0.5
		}
		if e.bodyType == 1 {
			x, y, z, ok := w.phys3.GetPosition(id)
			if !ok {
				continue
			}
			qx, qy, qz, qw, rok := w.phys3.GetRotation(id)
			if !rok {
				qx, qy, qz, qw = 0, 0, 0, 1
			}
			prizes = append(prizes, pose{id: id, x: x, y: y, z: z, hx: hx, hy: hy, hz: hz, qx: qx, qy: qy, qz: qz, qw: qw})
			continue
		}
		if e.bodyType != 3 && e.parent == 0 {
			continue
		}
		x, y, z, qx, qy, qz, qw, ok := w.meshPhysPose(e)
		if !ok {
			continue
		}
		drivers = append(drivers, pose{
			id: id, x: x, y: y, z: z, hx: hx, hy: hy, hz: hz,
			qx: qx, qy: qy, qz: qz, qw: qw,
			vx: e.driveVX, vy: e.driveVY, vz: e.driveVZ,
		})
	}
	for i := 0; i < len(drivers); i++ {
		d := drivers[i]
		for j := 0; j < len(prizes); j++ {
			p := prizes[j]
			nx, ny, nz, depth, hit := phys3d.BoxOverlapMTV(
				d.x, d.y, d.z, d.hx, d.hy, d.hz, d.qx, d.qy, d.qz, d.qw,
				p.x, p.y, p.z, p.hx, p.hy, p.hz, p.qx, p.qy, p.qz, p.qw,
			)
			if !hit || depth <= 0 {
				continue
			}
			p.x += nx * depth
			p.y += ny * depth
			p.z += nz * depth
			prizes[j] = p
			w.phys3.SetPosition(p.id, p.x, p.y, p.z)
			w.phys3.Wake(p.id)
			kick := float32(1.6)
			w.phys3.SetVelocity(p.id, d.vx+nx*kick, d.vy+ny*kick, d.vz+nz*kick)
		}
	}
}

func (w *World) updateWorld() {
	ts := w.timeScale
	if ts <= 0 {
		ts = 1.0
	}
	dt := float32(w.delta) * ts
	if dt <= 0 {
		dt = (1.0 / 60.0) * ts
	}
	w.updateTweens(dt)
	w.updatePathFollows(dt)
	for _, e := range w.ents {
		e.collided = e.collided[:0]
	}
	if w.phys3 != nil {
		w.driveParentedBodies(dt)
		w.applyRopeForces()
		w.applyClothWind()
		w.tickProjectiles(dt)
		if w.physAsync {
			w.ensureJobs().submit(func() { w.phys3.Step(dt) })
			w.jobs.waitAll()
		} else {
			w.phys3.Step(dt)
		}
		w.resolveDrivenOverlaps()
		w.syncPhys3Pose()
		w.drawPhysicsDebug()
		w.tickCloth()
		w.tickBeams()
		w.simulateRopes(dt)
		w.tickRopes()
		w.processPhysicsContacts()
	}
	if w.phys2 != nil {
		w.phys2.Step(float64(dt))
		for id, e := range w.ents {
			if x, y, ok := w.phys2.GetPosition(id); ok {
				e.node.GetNode().SetPosition(float32(x), float32(y), e.node.GetNode().Position().Z)
			}
		}
		for id, s := range w.sprites {
			if x, y, ok := w.phys2.GetPosition(id); ok {
				s.x, s.y = x, y
			}
		}
	}
	w.noteMouse()
	w.netUpdate()
	w.updateNav()
	w.ecsProgress(float64(dt))
	if w.phys2 != nil {
		for id, e := range w.ents {
			hits := w.phys2.Hits(id)
			if len(hits) == 0 {
				continue
			}
			e.collided = append(e.collided, hits...)
			e.hitID = hits[0]
		}
	}
	for _, rule := range w.rules {
		for idA, a := range w.ents {
			if a.etype != rule.src {
				continue
			}
			pa := worldPos(a.node.GetNode())
			for idB, b := range w.ents {
				if idA == idB || b.etype != rule.dest {
					continue
				}
				pb := worldPos(b.node.GetNode())
				dx := pa.X - pb.X
				dy := pa.Y - pb.Y
				dz := pa.Z - pb.Z
				dist := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
				min := a.radius + b.radius
				if dist < min && dist > 0 {
					a.collided = append(a.collided, idB)
					a.hitID = idB
					a.hitX, a.hitY, a.hitZ = fromG3N((pa.X+pb.X)/2, (pa.Y+pb.Y)/2, (pa.Z+pb.Z)/2)
					if rule.response == 1 || rule.response == 2 {
						n := min - dist
						a.node.GetNode().SetPosition(
							pa.X+dx/dist*n,
							pa.Y+dy/dist*n,
							pa.Z+dz/dist*n,
						)
					}
				}
			}
		}
	}
}

func (w *World) glfwWin() *window.GlfwWindow {
	if w.app == nil {
		return nil
	}
	gw, _ := w.app.IWindow.(*window.GlfwWindow)
	return gw
}

// pollHeldKey reads the live GLFW key so WASD still drives if a frame
// missed OnKeyDown. heldAtFlip / gui capture already returned above.
func (w *World) pollHeldKey(code int) bool {
	gw := w.glfwWin()
	if gw == nil {
		return false
	}
	gk, ok := dikToKey[code]
	if !ok {
		return false
	}
	return gw.GetKey(glfw.Key(gk)) == glfw.Press
}

func (w *World) loadMeshFile(path string, parent int) (int, error) {
	realPath, err := w.openPath(path)
	if err != nil {
		return 0, err
	}
	ext := strings.ToLower(filepath.Ext(realPath))
	var id int
	switch ext {
	case ".gltf":
		id, err = w.loadGLTFAt(realPath, parent, false)
	case ".glb":
		id, err = w.loadGLTFAt(realPath, parent, true)
	case ".dae":
		id, err = w.loadColladaAt(realPath, parent)
	default:
		id, err = w.loadObjAt(realPath, parent)
	}
	if err != nil {
		return 0, err
	}
	if e := w.ents[id]; e != nil {
		e.kind = "mesh"
		e.src = path
	}
	return id, nil
}

func (w *World) loadGLTF(path string, parent int, bin bool) (int, error) {
	real, err := w.openPath(path)
	if err != nil {
		return 0, err
	}
	return w.loadGLTFAt(real, parent, bin)
}

func (w *World) loadGLTFAt(path string, parent int, bin bool) (int, error) {
	var (
		g   *gltf.GLTF
		err error
	)
	if bin {
		g, err = gltf.ParseBin(path)
	} else {
		g, err = gltf.ParseJSON(path)
	}
	if err != nil {
		return 0, err
	}
	var node core.INode
	if len(g.Scenes) > 0 {
		idx := 0
		if g.Scene != nil {
			idx = *g.Scene
		}
		node, err = g.LoadScene(idx)
	} else if len(g.Meshes) > 0 {
		node, err = g.LoadMesh(0)
	} else {
		return 0, fmt.Errorf("LoadMesh: no scene or mesh in %s", path)
	}
	if err != nil {
		return 0, err
	}
	id := w.addEntity(&Entity{node: node, radius: 1}, parent)
	w.attachLoadedPBR(w.ents[id])
	return id, nil
}

func (w *World) loadCollada(path string, parent int) (int, error) {
	real, err := w.openPath(path)
	if err != nil {
		return 0, err
	}
	return w.loadColladaAt(real, parent)
}

func (w *World) loadColladaAt(path string, parent int) (int, error) {
	dec, err := collada.Decode(path)
	if err != nil {
		return 0, err
	}
	node, err := dec.NewScene()
	if err != nil {
		return 0, err
	}
	id := w.addEntity(&Entity{node: node, radius: 1}, parent)
	w.attachLoadedPBR(w.ents[id])
	return id, nil
}

func (w *World) loadObj(path string, parent int) (int, error) {
	real, err := w.openPath(path)
	if err != nil {
		return 0, err
	}
	return w.loadObjAt(real, parent)
}

func (w *World) loadObjAt(path string, parent int) (int, error) {
	matpath := strings.TrimSuffix(path, filepath.Ext(path)) + ".mtl"
	dec, err := obj.Decode(path, matpath)
	if err != nil {
		dec, err = obj.Decode(path, "")
		if err != nil {
			return 0, err
		}
	}
	group, err := dec.NewGroup()
	if err != nil {
		return 0, err
	}
	id := w.addEntity(&Entity{node: group, radius: 1}, parent)
	w.attachLoadedPBR(w.ents[id])
	return id, nil
}

func (w *World) noteMouse() {
	if !w.presented || !w.mouseInited {
		w.prevMX, w.prevMY = w.mx, w.my
		w.mouseInited = true
		w.mxs, w.mys, w.mzs = 0, 0, 0
		w.prevMZ = w.mz
		return
	}
	w.mxs = w.mx - w.prevMX
	w.mys = w.my - w.prevMY
	w.mzs = w.mz - w.prevMZ
	w.prevMX, w.prevMY = w.mx, w.my
	w.prevMZ = w.mz
}

func (w *World) pos2D(id int) (float64, float64) {
	if s := w.sprites[id]; s != nil {
		return s.x, s.y
	}
	if e := w.ents[id]; e != nil {
		p := e.node.GetNode().Position()
		return float64(p.X), float64(p.Y)
	}
	return 0, 0
}

func argN(a []value.Value, i int, def float64) float64 {
	if i >= len(a) {
		return def
	}
	return a[i].Number()
}

func argI(a []value.Value, i int, def int) int {
	if i >= len(a) {
		return def
	}
	return a[i].Int()
}

func argS(a []value.Value, i int) string {
	if i >= len(a) {
		return ""
	}
	return a[i].String()
}
