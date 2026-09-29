package interp

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"bitshinbasic/internal/parse"
	"bitshinbasic/internal/syntax"
	"bitshinbasic/internal/value"
)

type nopHost struct{}

func (nopHost) Call(name string, args []value.Value) (value.Value, error) {
	return value.Value{}, errUnknown(name)
}
func (nopHost) Yields(string) bool { return false }
func (nopHost) Holds() bool        { return false }

func errUnknown(name string) error { return errf(name) }

type unk struct{ n string }

func (e unk) Error() string { return "unknown command " + e.n }
func errf(n string) error   { return unk{n} }

func run(t *testing.T, src string) string {
	t.Helper()
	prog, err := parse.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	in := New(prog, nopHost{})
	in.Out = &buf
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestArithmeticAndPrint(t *testing.T) {
	out := run(t, `
Print 2 + 3 * 4
Print 2 ^ 3
Print "hi" + " " + "there"
`)
	if !strings.Contains(out, "14") || !strings.Contains(out, "8") || !strings.Contains(out, "hi there") {
		t.Fatalf("unexpected output %q", out)
	}
}

func TestForAndFunction(t *testing.T) {
	out := run(t, `
Function Double(n)
    Return n * 2
End Function
s = 0
For i = 1 To 4
    s = s + Double(i)
Next
Print s
`)
	if strings.TrimSpace(out) != "20" {
		t.Fatalf("got %q", out)
	}
}

func TestWhileIf(t *testing.T) {
	out := run(t, `
x = 0
While x < 3
    x = x + 1
Wend
If x = 3 Then Print "ok"
`)
	if !strings.Contains(out, "ok") {
		t.Fatalf("got %q", out)
	}
}

func TestSeedRndDeterministic(t *testing.T) {
	src := `
SeedRnd 12345
Print Rand(1, 1000)
Print RndSeed()
`
	a := run(t, src)
	b := run(t, src)
	if a != b {
		t.Fatalf("SeedRnd not deterministic:\n%s\nvs\n%s", a, b)
	}
	if !strings.Contains(a, "12345") {
		t.Fatalf("RndSeed missing: %q", a)
	}
}

func TestGameMath(t *testing.T) {
	out := run(t, `
Print Int(Sin(90))
Print Int(Cos(0))
Print Int(ATan2(0, 1))
Print Dist(0, 0, 3, 4)
Print Distance3D(0, 0, 0, 0, 3, 4)
Print Clamp(15, 0, 10)
Print Lerp(0, 10, 0.5)
Print InvLerp(0, 10, 5)
Print Int(EaseIn(0.5) * 100)
Print Int(SmoothStep(0, 1, 0.5) * 100)
Print WrapAngle(270)
Print AngleDelta(10, 350)
Print Approach(0, 10, 3)
Print Approach(8, 10, 5)
Print ApproachAngle(170, -170, 5)
Print Length2D(3, 4)
Print NormX(3, 4)
Print Int(DirX(90))
Print Int(DirZ(0))
Print MovePointX(0, 90, 10)
Print MovePointZ(0, 0, 10)
Print PointYaw(0, 0, 10, 0)
Print Int(PointPitch(0, 0, 0, 0, 1, 0))
Print Dot2D(1, 0, 0, 1)
Print CrossZ(1, 0, 0, 0, 1, 0)
Print ReflectX(1, -1, 0, 1)
Print ReflectY(1, -1, 0, 1)
Print Int(RotatedX(1, 0, 90))
Print Int(RotatedY(1, 0, 90))
Print Min(3, 1)
Print Max(3, 1)
Print Sgn(-2)
Print Pow(2, 3)
Print PointDistance(0, 0, 0, 6, 0, 8)
`)
	want := []string{
		"1", "1", "0",
		"5", "5",
		"10", "5", "0.5",
		"25", "50",
		"-90", "-20",
		"3", "10",
		"175",
		"5", "0.6",
		"1", "1",
		"10", "10",
		"90", "90",
		"0", "1",
		"1", "1",
		"0", "1",
		"1", "3",
		"-1", "8",
		"10",
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d\n%s", len(lines), len(want), out)
	}
	for i, w := range want {
		if strings.TrimSpace(lines[i]) != w {
			t.Fatalf("line %d: got %q want %q\n%s", i+1, lines[i], w, out)
		}
	}
}

func TestYesNoNull(t *testing.T) {
	out := run(t, `
If Yes Then Print "yes"
If No Then Print "nope"
x = Null
If x = Null Then Print "null"
If Not No Then Print "ok"
If True = Yes Then Print "same"
`)
	got := strings.TrimSpace(out)
	if !strings.Contains(got, "yes") || !strings.Contains(got, "null") || !strings.Contains(got, "ok") || !strings.Contains(got, "same") {
		t.Fatalf("got %q", out)
	}
	if strings.Contains(got, "nope") {
		t.Fatalf("No should be false: %q", out)
	}
}

func TestPhysicsMotionConstants(t *testing.T) {
	out := run(t, `
If STATIC = 0 Then Print "static"
If KINEMATIC = 1 Then Print "kinematic"
If DYNAMIC = 2 Then Print "dynamic"
If ON_GROUND = 0 Then Print "ground"
If IN_AIR = 3 Then Print "air"
`)
	for _, want := range []string{"static", "kinematic", "dynamic", "ground", "air"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestStructMethodImport(t *testing.T) {
	lib := `
Function Double(n)
    Return n * 2
End Function
`
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/lib.bb", []byte(lib), 0o644); err != nil {
		t.Fatal(err)
	}
	src := `
Import "lib.bb" As Lib
Struct Vec
    Field x
    Field y
    Method Length()
        Return Sqr(Self.x * Self.x + Self.y * Self.y)
    End Method
    Method Add(n)
        Self.x = Self.x + n
    End Method
End Struct
Namespace Util
    Function Sum(a, b)
        Return a + b
    End Function
End Namespace
v = Vec(3, 4)
Print Int(v.Length())
w = v
w.x = 0
Print v.x
v.Add(2)
Print v.x
Print Lib.Double(6)
Print Util.Sum(2, 3)
p = New Vec
p.x = 9
Print p.x
`
	prog, err := parse.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	prog, err = parse.ExpandIncludes(prog, dir)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	in := New(prog, nopHost{})
	in.Out = &buf
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	got := strings.Fields(buf.String())
	want := []string{"5", "0", "2", "12", "5", "9"}
	if len(got) != len(want) {
		t.Fatalf("got %q want %v", buf.String(), want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("item %d: got %q want %q (%q)", i, got[i], w, buf.String())
		}
	}
}

func TestNotAnd(t *testing.T) {
	out := run(t, `
If Not 0 And 1 Then Print "yes"
If 1 = 2 Then Print "no"
`)
	if strings.TrimSpace(out) != "yes" {
		t.Fatalf("got %q", out)
	}
}

type audioErr struct{}

func (audioErr) Error() string { return "no audio device" }

type flipBoomHost struct{}

func (flipBoomHost) Call(name string, args []value.Value) (value.Value, error) {
	if name == "playsound" {
		return value.Value{}, audioErr{}
	}
	return value.Num(0), nil
}
func (flipBoomHost) Yields(name string) bool { return name == "flip" }
func (flipBoomHost) Holds() bool             { return false }

func TestLiveCommandErrorDoesNotEnd(t *testing.T) {
	prog, err := parse.Parse(`
Flip
PlaySound(1)
Print "still here"
Flip
`)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	in := New(prog, flipBoomHost{})
	in.Out = &buf
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	if !in.live {
		t.Fatal("Flip should mark the program live")
	}
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	if in.Done() {
		t.Fatal("PlaySound error after Flip must not End the program")
	}
	out := buf.String()
	if !strings.Contains(out, "still here") {
		t.Fatalf("loop should continue, got %q", out)
	}
	if !strings.Contains(out, "no audio device") {
		t.Fatalf("expected printed command error, got %q", out)
	}
}

type alwaysQuitHost struct{}

func (alwaysQuitHost) Call(name string, args []value.Value) (value.Value, error) {
	switch name {
	case "keydown", "keyhit", "windowshouldclose":
		return value.Num(1), nil
	}
	return value.Num(0), nil
}
func (alwaysQuitHost) Yields(name string) bool { return name == "flip" }
func (alwaysQuitHost) Holds() bool             { return false }

func TestWhileKeyDownDoesNotEndBeforeFlip(t *testing.T) {
	prog, err := parse.Parse(`
While Not KeyDown(1)
    Flip
Wend
End
`)
	if err != nil {
		t.Fatal(err)
	}
	in := New(prog, alwaysQuitHost{})
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	if !in.live {
		t.Fatal("While Not KeyDown(1) must enter and Flip before End")
	}
	if in.Done() {
		t.Fatal("pre-Flip KeyDown/ShouldClose must not End the program")
	}
}

type qolHost struct {
	calls []string
	argc  []int
}

func (h *qolHost) Call(name string, args []value.Value) (value.Value, error) {
	h.calls = append(h.calls, name)
	h.argc = append(h.argc, len(args))
	switch name {
	case "createcube", "createcamera":
		return value.Num(7), nil
	case "setposition", "setscale", "entitycolor", "positionentity":
		return value.Num(0), nil
	case "setweather":
		return value.Str(syntax.WeatherMode(args[0])), nil
	}
	return value.Value{}, errUnknown(name)
}
func (h *qolHost) Yields(string) bool { return false }
func (h *qolHost) Holds() bool        { return false }

func TestDotMethodsChainVecHexWeather(t *testing.T) {
	src := `
box = CreateCube().Scale(2, 2, 2).Position([0, 2.2, -8]).Color($FFECc8)
Print box
cam = CreateCamera()
cam.Position(1, 2, 3)
PositionEntity box, 4, 5, 6
Print Hex("FF0000")
Print WEATHER_SNOW
SetWeather WEATHER_RAIN
Print WEATHER_CLEAR
`
	prog, err := parse.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	h := &qolHost{}
	in := New(prog, h)
	in.Out = &buf
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	out := strings.Fields(buf.String())
	if len(out) < 3 || out[0] != "7" || out[1] != "16711680" || out[2] != "snow" {
		t.Fatalf("output %q host %v", buf.String(), h.calls)
	}
	want := map[string]bool{"createcube": true, "setscale": true, "setposition": true, "entitycolor": true, "createcamera": true, "positionentity": true, "setweather": true}
	got := map[string]bool{}
	for _, c := range h.calls {
		got[c] = true
	}
	for n := range want {
		if !got[n] {
			t.Fatalf("missing host call %s in %v", n, h.calls)
		}
	}
	// vec/hex expand: Position([0,2.2,-8]) → 4 args; Color($hex) → 4 args
	for i, c := range h.calls {
		if c == "setposition" && h.argc[i] != 4 {
			t.Fatalf("setposition argc %d want 4 (vec/xyz expand)", h.argc[i])
		}
		if c == "entitycolor" && h.argc[i] != 4 {
			t.Fatalf("entitycolor argc %d want 4 (hex expand)", h.argc[i])
		}
		if c == "positionentity" && h.argc[i] != 4 {
			t.Fatalf("classic PositionEntity argc %d", h.argc[i])
		}
	}
}

func TestIfKeyHitEscapeDoesNotEndBeforeFlip(t *testing.T) {
	prog, err := parse.Parse(`
If KeyHit(1) Then End
Flip
Print "after"
`)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	in := New(prog, alwaysQuitHost{})
	in.Out = &buf
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	if !in.live || in.Done() {
		t.Fatal("If KeyHit(1) Then End must not fire before Flip")
	}
}

func TestWhileKeyDownQuitsAfterFlip(t *testing.T) {
	prog, err := parse.Parse(`
While Not KeyDown(1)
    Flip
Wend
End
`)
	if err != nil {
		t.Fatal(err)
	}
	in := New(prog, alwaysQuitHost{})
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	if !in.live || in.Done() {
		t.Fatal("must Flip once")
	}
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	if !in.Done() {
		t.Fatal("after Flip, KeyDown(1) must be able to End the loop")
	}
}

func TestCaptureQuitOrFramesDoesNotEndWithoutEsc(t *testing.T) {
	prog, err := parse.Parse(`
frames = 0
While True
    Flip
    frames = frames + 1
    If KeyHit(1) Or frames > 8 Then End
Wend
`)
	if err != nil {
		t.Fatal(err)
	}
	in := New(prog, keepOpenHost{})
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	if !in.live || in.Done() {
		t.Fatal("must yield at first Flip")
	}
	for i := 0; i < 20; i++ {
		if err := in.Run(); err != nil {
			t.Fatal(err)
		}
		if in.Done() {
			t.Fatalf("KeyHit(1) Or frames > 8 must not End without Escape (iter %d)", i)
		}
	}
}

type keepOpenHost struct{}

func (keepOpenHost) Call(string, []value.Value) (value.Value, error) { return value.Num(0), nil }
func (keepOpenHost) Yields(name string) bool                         { return name == "flip" }
func (keepOpenHost) Holds() bool                                     { return false }

func TestWhileTrueLoopsUntilEnd(t *testing.T) {
	prog, err := parse.Parse(`
n = 0
While True
    n = n + 1
    If n = 3 Then End
Wend
`)
	if err != nil {
		t.Fatal(err)
	}
	in := New(prog, nopHost{})
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	if !in.Done() {
		t.Fatal("While True must run until End")
	}
	v, ok := in.global.get("n")
	if !ok || v.Int() != 3 {
		t.Fatalf("n=%v ok=%v", v, ok)
	}
}

func TestWhileWindowShouldCloseDoesNotEndBeforeFlip(t *testing.T) {
	prog, err := parse.Parse(`
While Not WindowShouldClose()
    Flip
Wend
`)
	if err != nil {
		t.Fatal(err)
	}
	in := New(prog, alwaysQuitHost{})
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	if !in.live || in.Done() {
		t.Fatal("While Not WindowShouldClose must enter until Flip")
	}
}

type recHudHost struct {
	nopHost
	lines []string
}

func (h *recHudHost) Call(name string, args []value.Value) (value.Value, error) {
	if name == "hudprint" {
		line := ""
		if len(args) > 0 {
			line = args[0].String()
		}
		h.lines = append(h.lines, line)
		return value.Num(0), nil
	}
	return h.nopHost.Call(name, args)
}

func TestPrintAlsoCallsHud(t *testing.T) {
	prog, err := parse.Parse(`Print("Game loop started. Use ESC to quit.")`)
	if err != nil {
		t.Fatal(err)
	}
	host := &recHudHost{}
	buf := bytes.Buffer{}
	in := New(prog, host)
	in.Out = &buf
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Game loop started. Use ESC to quit.") {
		t.Fatalf("console Print missing: %q", buf.String())
	}
	if len(host.lines) != 1 || host.lines[0] != "Game loop started. Use ESC to quit." {
		t.Fatalf("on-screen Print missing: %#v", host.lines)
	}
}

func TestStrictMapsTryHandlesAndFast(t *testing.T) {
	src := `
Strict
Local hp% = 3.9
Local name$ = 12
Local m = CreateMap()
MapSet m, "hp", hp%
m("name") = name$
Print MapGet(m, "hp")
Print m("name")
Print hp%

Struct P
    Field x
End Struct
Local a = P(1)
Local b = a
b.x = 4
Local c = Copy(a)
c.x = 9
Print a.x
Print c.x

Function Double(n#)
    Return n# * 2
End Function
Print Double(4)
Print Callback(Double)

Try
    Boom
Catch err$
    Print "caught"
End Try

Local sm = CreateStateMachine()
AddState sm, "idle", "IdleTick"
Function IdleTick()
    Print "idle"
End Function
UpdateState sm
`
	out := run(t, src)
	fields := strings.Fields(out)
	want := []string{"3", "12", "3", "4", "9", "8", "double", "caught", "idle"}
	if len(fields) != len(want) {
		t.Fatalf("got %q want %v", out, want)
	}
	for i, w := range want {
		if fields[i] != w {
			t.Fatalf("item %d got %q want %q in %q", i, fields[i], w, out)
		}
	}
}

func TestFlipInsideCalledFunction(t *testing.T) {
	prog, err := parse.Parse(`
Function Tick()
    Print "a"
    Flip
    Print "b"
End Function
Tick
`)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	in := New(prog, flipBoomHost{})
	in.Out = &buf
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(buf.String()) != "a" {
		t.Fatalf("before flip: %q", buf.String())
	}
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(buf.String()) != "a b" && strings.TrimSpace(buf.String()) != "a\nb" {
		got := strings.Fields(buf.String())
		if len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Fatalf("after resume: %q", buf.String())
		}
	}
}

func TestFastPathUsed(t *testing.T) {
	prog, err := parse.Parse(`
Function Double(n#)
    Return n# * 2
End Function
Print Double(3)
`)
	if err != nil {
		t.Fatal(err)
	}
	in := New(prog, nopHost{})
	var buf bytes.Buffer
	in.Out = &buf
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	if in.fastHits == 0 {
		t.Fatal("numeric function should run on the bytecode path")
	}
	if strings.TrimSpace(buf.String()) != "6" {
		t.Fatalf("got %q", buf.String())
	}
}

func TestCollectionsDefaultsAndMotion(t *testing.T) {
	out := run(t, `
Type Pad
    Field x, y, z, w, d
End Type
Type Coin
    Field x, y
End Type
Dim pads(4) As Pad
pads(1).x = 0
pads(1).y = 1
pads(1).z = 0
pads(1).w = 2
pads(1).d = 2
pads(2) = Pad(4, 5, 0, 1, 1)
sum = 0
For p In pads
    p.x = p.x + 1
    sum = sum + p.x
Next
Print pads(1).x
Print pads(1).y
Print sum

Function Add(a, b = 10, c = b + 1)
    Return a + b + c
End Function
Print Add(1)
Print Add(1, 2, 3)

n = 1
i = 7
Function Bump()
    n = n + 1
    Return n
End Function
Function Inc()
    Global n
    n = n + 1
End Function
Function Loop()
    For i = 1 To 3
    Next
    Return i
End Function
Print Bump()
Print n
Inc()
Print n
Print Loop()
Print i

Data 10, 2
c = Coin()
Read c.x, c.y
Print c.x
Print c.y

Function Pair()
    Return 3, 4
End Function
a, b = Pair()
Print a
Print b
x, y = 8, 9
Print x
Print y

py, vy, g = Land(0, 1.2, 0, -1, pads)
Print py
Print vy
Print g
vx, vz = Accelerate(0, 0, 0, 1, 2, 0, 10, 1, 1)
Print vx
Print vz
Print TurnToward(0, 1, 0, 10)
wishX, wishZ = MoveWish(0)
Print wishX
Print wishZ

m = Material($C8A046, 3, 0.11)
Print m.r
Print m.g
Print m.b
Print m.tex
Print m.shine

lst = CreateList()
ListAdd lst, Coin(3, 4)
For item In lst
    item.x = item.x + 1
Next
Print ListGet(lst, 0).x
`)
	want := []string{
		"1", "1", "6",
		"22", "6",
		"2", "1", "2", "4", "7",
		"10", "2",
		"3", "4", "8", "9",
		"1", "0", "1",
		"0", "2", "10",
		"0", "0",
		"200", "160", "70", "3", "0.11",
		"4",
	}
	got := strings.Fields(out)
	if len(got) != len(want) {
		t.Fatalf("got %q\nwant %v", out, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("item %d got %q want %q in %q", i, got[i], w, out)
		}
	}
}

func TestStrictFunctionLocal(t *testing.T) {
	prog, err := parse.Parse(`
Strict
x = 1
Function F()
    x = 2
End Function
F()
`)
	if err != nil {
		t.Fatal(err)
	}
	in := New(prog, nopHost{})
	err = in.Run()
	if err == nil || !strings.Contains(err.Error(), "undefined") {
		t.Fatalf("got %v", err)
	}
}
