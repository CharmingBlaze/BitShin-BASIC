package runtime

import (
	"math"
	"testing"

	"moonbasic/internal/value"
)

func near2(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func newWorld2D(t *testing.T) (*World, map[string]cmd) {
	t.Helper()
	w := New(".")
	w.ready = true
	w.mode2D = true
	w.scrW, w.scrH = 800, 600
	return w, w.commandTable()
}

func call(t *testing.T, cmds map[string]cmd, name string, args ...float64) value.Value {
	t.Helper()
	vs := make([]value.Value, len(args))
	for i, a := range args {
		vs[i] = value.Num(a)
	}
	v, err := cmds[name](vs)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return v
}

func TestCamera2DDefaultKeepsTopLeft(t *testing.T) {
	w, cmds := newWorld2D(t)
	call(t, cmds, "setcamera2d", 100, 50)
	if x := call(t, cmds, "camera2dx").Num; x != 100 {
		t.Fatalf("Camera2DX %v", x)
	}
	// At zoom 1 a world point moves by exactly the camera offset, as before.
	sx := call(t, cmds, "worldtoscreenx2d", 0, 130, 70).Num
	sy := call(t, cmds, "worldtoscreeny2d", 0, 130, 70).Num
	if !near2(sx, 30) || !near2(sy, 20) {
		t.Fatalf("world to screen %v,%v", sx, sy)
	}
	call(t, cmds, "movecamera2d", 10, 0)
	if x := call(t, cmds, "camera2dx").Num; x != 110 {
		t.Fatalf("after MoveCamera2D %v", x)
	}
	_ = w
}

func TestCamera2DZoomRotateRoundTrip(t *testing.T) {
	_, cmds := newWorld2D(t)
	cam := call(t, cmds, "createcamera2d", 400, 0, 400, 300).Num
	call(t, cmds, "positioncamera2d", cam, 1000, 500)
	call(t, cmds, "camerazoom2d", cam, 2)
	call(t, cmds, "camerarotation2d", cam, 30)
	// The centre of the viewport sees the camera's centre.
	if x := call(t, cmds, "screentoworldx2d", cam, 600, 150).Num; !near2(x, 1000) {
		t.Fatalf("centre x %v", x)
	}
	// Zoom 2 halves world distances on screen and the mapping inverts.
	sx := call(t, cmds, "worldtoscreenx2d", cam, 1037, 512).Num
	sy := call(t, cmds, "worldtoscreeny2d", cam, 1037, 512).Num
	wx := call(t, cmds, "screentoworldx2d", cam, sx, sy).Num
	wy := call(t, cmds, "screentoworldy2d", cam, sx, sy).Num
	if !near2(wx, 1037) || !near2(wy, 512) {
		t.Fatalf("round trip %v,%v", wx, wy)
	}
	if d := math.Hypot(sx-600, sy-150); !near2(d, 2*math.Hypot(37, 12)) {
		t.Fatalf("zoomed distance %v", d)
	}
}

func TestCamera2DFollowDeadZoneAndBounds(t *testing.T) {
	c := &camera2D{zoom: 1, deadW: 100, deadH: 0}
	c.followStep(30, 0, 1.0/60) // inside the 50 px half dead zone: stays
	if c.cx != 0 {
		t.Fatalf("moved inside dead zone: %v", c.cx)
	}
	c.followStep(80, 0, 1.0/60) // 30 px past the edge, no smoothing: closes it
	if !near2(c.cx, 30) {
		t.Fatalf("dead zone edge %v", c.cx)
	}
	c = &camera2D{zoom: 1, smooth: 0.5}
	for i := 0; i < 30; i++ { // half a second at 60 fps closes 95%
		c.followStep(100, 0, 1.0/60)
	}
	if !near2(math.Round(c.cx*1000)/1000, 95) {
		t.Fatalf("smoothing after 0.5 s %v", c.cx)
	}

	w, cmds := newWorld2D(t)
	call(t, cmds, "camerabounds2d", 0, 0, 0, 1000, 700)
	call(t, cmds, "positioncamera2d", 0, -500, 900)
	w.tickCameras2D()
	c = w.cam2D(0)
	if c.cx != 400 || c.cy != 400 { // half view is 400 x 300
		t.Fatalf("bounds clamp %v,%v", c.cx, c.cy)
	}
	call(t, cmds, "camerazoom2d", 0, 0.5) // view 1600 wide: wider than the level
	w.tickCameras2D()
	if c.cx != 500 {
		t.Fatalf("centred in a small level %v", c.cx)
	}
}

func TestCamera2DShakeFades(t *testing.T) {
	w, cmds := newWorld2D(t)
	call(t, cmds, "camerashake2d", 0, 10, 0.2)
	w.delta = 0.05
	w.tickCameras2D()
	c := w.cam2D(0)
	if c.shakeX == 0 && c.shakeY == 0 {
		t.Fatal("no shake")
	}
	if math.Abs(c.shakeX) > 10 || math.Abs(c.shakeY) > 10 {
		t.Fatalf("shake past amplitude %v,%v", c.shakeX, c.shakeY)
	}
	w.delta = 0.3
	w.tickCameras2D()
	if c.shakeX != 0 || c.shakeY != 0 {
		t.Fatalf("shake did not end %v,%v", c.shakeX, c.shakeY)
	}
}

func TestSpriteParentPivotBlendRect(t *testing.T) {
	w, cmds := newWorld2D(t)
	parent := &ebiSprite{id: 1, x: 100, y: 100, rot: 90, sx: 2, sy: 2, visible: true, alpha: 1}
	child := &ebiSprite{id: 2, x: 10, y: 0, sx: 1, sy: 1, visible: true, alpha: 1}
	w.sprites[1], w.sprites[2] = parent, child
	call(t, cmds, "setspriteparent", 2, 1)
	// 10 px along the parent's x, scaled 2 and turned 90 degrees: straight down.
	x := call(t, cmds, "spriteworldx", 2).Num
	y := call(t, cmds, "spriteworldy", 2).Num
	if !near2(x, 100) || !near2(y, 120) {
		t.Fatalf("child world %v,%v", x, y)
	}
	// keepWorld: re-parenting to nothing leaves it in place.
	call(t, cmds, "setspriteparent", 2, 0, 1)
	if !near2(child.x, 100) || !near2(child.y, 120) || child.parent != 0 {
		t.Fatalf("detach keepWorld %v,%v parent %d", child.x, child.y, child.parent)
	}
	if _, err := cmds["setspriteparent"]([]value.Value{value.Num(1), value.Num(1)}); err == nil {
		t.Error("a sprite became its own parent")
	}

	call(t, cmds, "setspritepivot", 2, 0.5, 1)
	if child.pvx != 0 || child.pvy != 0.5 {
		t.Fatalf("pivot %v,%v", child.pvx, child.pvy)
	}
	if _, err := cmds["setspriteblend"]([]value.Value{value.Num(2), value.Str("Add")}); err != nil || child.blend != blendAdd {
		t.Fatalf("blend %d %v", child.blend, err)
	}
	if _, err := cmds["setspriteblend"]([]value.Value{value.Num(2), value.Str("glow")}); err == nil {
		t.Error("unknown blend accepted")
	}
	call(t, cmds, "setspriterect", 2, 4, 8, 16, 32)
	if child.uw != 16 || child.uh != 32 {
		t.Fatalf("rect %d x %d", child.uw, child.uh)
	}
	call(t, cmds, "setspriterect", 2)
	if child.uw != 0 {
		t.Fatal("rect not cleared")
	}
}
