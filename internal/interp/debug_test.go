package interp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"moonbasic/internal/parse"
)

const debugProg = `Function Inner(n)
    m = n * 2
    Return m
End Function
Function Outer(a)
    b = Inner(a)
    c = b + 1
    Return c
End Function
x = 1
y = Outer(x)
z = y + 1
Print z
`

// debugRun runs src under the debugger with the given commands, one per line.
func debugRun(t *testing.T, src string, breaks []int, cmds ...string) (out, dbg string, in *Interp) {
	t.Helper()
	prog, err := parse.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	var o, d bytes.Buffer
	in = New(prog, nopHost{})
	in.Out = &o
	in.ErrOut = &o
	in.SetScript("prog.mb")
	in.SetDebug(breaks, strings.NewReader(strings.Join(cmds, "\n")+"\n"), &d)
	_ = in.Run()
	return o.String(), d.String(), in
}

func pausedLines(dbg string) []string {
	var lines []string
	for _, ln := range strings.Split(dbg, "\n") {
		ln = strings.TrimPrefix(strings.TrimSpace(ln), "debug> ")
		for strings.HasPrefix(ln, "debug> ") {
			ln = strings.TrimPrefix(ln, "debug> ")
		}
		if strings.HasPrefix(ln, "break ") {
			lines = append(lines, strings.TrimPrefix(ln, "break "))
		}
	}
	return lines
}

func TestDebugStepIntoOverOut(t *testing.T) {
	cases := []struct {
		name string
		cmds []string
		want string
	}{
		// s steps into each statement, so Outer and Inner bodies are visited.
		{"into", []string{"s", "s", "s", "c"}, "11 6 2 3"},
		// n stays in the main program: the calls run without pausing.
		{"over", []string{"n", "n", "c"}, "11 12 13"},
		// into Outer, into Inner, then out of Inner lands back in Outer on line 7.
		{"out", []string{"s", "s", "o", "c"}, "11 6 2 7"},
		// out of Outer from inside Inner's caller lands on line 12 in main.
		{"over-then-out", []string{"s", "n", "o", "c"}, "11 6 7 12"},
	}
	for _, tc := range cases {
		out, dbg, _ := debugRun(t, debugProg, []int{11}, tc.cmds...)
		got := strings.Join(pausedLines(dbg), " ")
		if got != tc.want {
			t.Errorf("%s: paused at %q want %q\n%s", tc.name, got, tc.want, dbg)
		}
		if strings.TrimSpace(out) != "4" {
			t.Errorf("%s: program did not finish: %q", tc.name, out)
		}
	}
}

func TestDebugLegacyStepAndBreak(t *testing.T) {
	_, dbg, _ := debugRun(t, "a = 1\nb = 2\nc = 3\n", []int{1}, "s", "c")
	if got := strings.Join(pausedLines(dbg), " "); got != "1 2" {
		t.Fatalf("paused at %q\n%s", got, dbg)
	}
	if !strings.Contains(dbg, "a = 1") {
		t.Fatalf("pause should list variables:\n%s", dbg)
	}
}

func TestDebugCallStack(t *testing.T) {
	_, dbg, _ := debugRun(t, debugProg, []int{2}, "bt", "c")
	want := "#0 Inner prog.mb:2\n#1 Outer prog.mb:6\n#2 (main) prog.mb:11\n"
	if !strings.Contains(dbg, want) {
		t.Fatalf("call stack missing %q:\n%s", want, dbg)
	}
}

func TestDebugCallStackMethodsAndCallStmt(t *testing.T) {
	src := `Type Pt
    Field x
    Method Bump()
        DebugBreak
        self.x = self.x + 1
    End Method
End Type
Function Go()
    p = Pt(1)
    p.Bump()
End Function
Go
`
	_, dbg, _ := debugRun(t, src, nil, "bt", "c")
	want := "#0 Pt.Bump prog.mb:4\n#1 Go prog.mb:10\n#2 (main) prog.mb:12\n"
	if !strings.Contains(dbg, want) {
		t.Fatalf("call stack missing %q:\n%s", want, dbg)
	}
}

func TestDebugVariablesAndWatch(t *testing.T) {
	src := `Type Pt
    Field x
    Field y
End Type
Dim arr(3)
arr(1) = 7
name$ = "moon"
p = Pt(1, 2)
Function F(k)
    local1 = k + 1
    Return local1
End Function
r = F(4)
`
	_, dbg, _ := debugRun(t, src, []int{11}, "locals", "globals", "p local1 * 10", "p name$ + \"!\"", "p arr(1)", "c")
	for _, want := range []string{
		"locals:\n  k = 4\n  local1 = 5\n",
		"arr = Array(3) [0, 7, 0, 0]",
		`name = "moon"`,
		"p = pt {x: 1, y: 2}",
		"local1 * 10 = 50",
		`name$ + "!" = "moon!"`,
		"arr(1) = 7",
	} {
		if !strings.Contains(dbg, want) {
			t.Errorf("missing %q in:\n%s", want, dbg)
		}
	}
	if strings.Contains(dbg, "  true = ") {
		t.Errorf("built-in constants should not be listed:\n%s", dbg)
	}
}

func TestDebugJSONPause(t *testing.T) {
	prog, err := parse.Parse(debugProg)
	if err != nil {
		t.Fatal(err)
	}
	var o, d bytes.Buffer
	in := New(prog, nopHost{})
	in.Out = &o
	in.SetScript("prog.mb")
	in.SetDebug([]int{3}, strings.NewReader("p m + 1\nc\n"), &d)
	in.SetDebugJSON(true)
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	var st debugState
	var ev map[string]string
	for _, ln := range strings.Split(d.String(), "\n") {
		if rest, ok := strings.CutPrefix(ln, "@debug "); ok {
			if err := json.Unmarshal([]byte(rest), &st); err != nil {
				t.Fatal(err)
			}
		}
		if rest, ok := strings.CutPrefix(ln, "@eval "); ok {
			if err := json.Unmarshal([]byte(rest), &ev); err != nil {
				t.Fatal(err)
			}
		}
	}
	if st.Line != 3 || st.Depth != 3 || len(st.Stack) != 3 || st.Stack[1].Name != "Outer" || st.Stack[1].Line != 6 {
		t.Fatalf("bad pause state %+v\n%s", st, d.String())
	}
	if len(st.Locals) != 2 || st.Locals[0].Name != "m" || st.Locals[0].Value != "2" {
		t.Fatalf("bad locals %+v", st.Locals)
	}
	if ev["expr"] != "m + 1" || ev["value"] != "3" {
		t.Fatalf("bad eval %v", ev)
	}
	if strings.Contains(d.String(), "debug> ") {
		t.Fatalf("JSON mode should not print a prompt:\n%s", d.String())
	}
}

func TestDebugLogOutput(t *testing.T) {
	prog, err := parse.Parse("DebugLog \"quiet\"\n")
	if err != nil {
		t.Fatal(err)
	}
	var o bytes.Buffer
	in := New(prog, nopHost{})
	in.Out, in.ErrOut, in.DebugLogs = &o, &o, false
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	if o.Len() != 0 {
		t.Fatalf("DebugLog should be quiet without -debug or MOON_DEBUG: %q", o.String())
	}

	prog, err = parse.Parse("x = 3\nDebugLog \"x is \" + x\nDebugLog(\"low\", \"warn\")\nDebugLog \"bad\", \"error\"\n")
	if err != nil {
		t.Fatal(err)
	}
	var info, errs bytes.Buffer
	in = New(prog, nopHost{})
	in.Out, in.ErrOut, in.DebugLogs = &info, &errs, true
	in.SetScript("game.mb")
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	if info.String() != "[debug] game.mb:2: x is 3\n" {
		t.Fatalf("info log %q", info.String())
	}
	if errs.String() != "[debug:warn] game.mb:3: low\n[debug:error] game.mb:4: bad\n" {
		t.Fatalf("warn/error log %q", errs.String())
	}
}

func TestAssertFailureNamesLine(t *testing.T) {
	src := `Function Check(v)
    Assert v > 0, "v must be positive"
    Return v
End Function
a = Check(2)
Assert a = 2
b = Check(-1)
Print "unreachable"
`
	prog, err := parse.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	var o bytes.Buffer
	in := New(prog, nopHost{})
	in.Out = &o
	in.SetScript("prog.mb")
	err = in.Run()
	if err == nil {
		t.Fatal("expected Assert failure")
	}
	if err.Error() != "prog.mb:2: Assert failed: v must be positive" {
		t.Fatalf("got %q", err.Error())
	}
	if strings.Contains(o.String(), "unreachable") {
		t.Fatal("program kept running after Assert failed")
	}

	prog, _ = parse.Parse("x = 0\nAssert x\n")
	in = New(prog, nopHost{})
	in.Out = &o
	if err := in.Run(); err == nil || err.Error() != "main:2: Assert failed: assertion failed" {
		t.Fatalf("got %v", err)
	}

	prog, _ = parse.Parse("Try\n    Assert 0, \"caught\"\nCatch e$\n    Print e$\nEnd Try\n")
	var c bytes.Buffer
	in = New(prog, nopHost{})
	in.Out = &c
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(c.String()) != "main:2: Assert failed: caught" {
		t.Fatalf("Try should catch Assert: %q", c.String())
	}
}

func TestAssertStopsLiveGame(t *testing.T) {
	prog, _ := parse.Parse(`Function F()
    Assert 1 = 2, "live"
End Function
While 1
    F()
Wend
`)
	var o bytes.Buffer
	in := New(prog, nopHost{})
	in.Out = &o
	in.MarkLive()
	_ = in.Run()
	in.ClearLiveError()
	if !in.Done() || in.Err() == nil || in.Err().Error() != "main:2: Assert failed: live" {
		t.Fatalf("Assert after Flip must stop the game: done=%v err=%v", in.Done(), in.Err())
	}
}

func TestAssertAndDebugBreakPauseWhenDebugging(t *testing.T) {
	_, dbg, _ := debugRun(t, "a = 5\nDebugBreak\nAssert a = 4, \"a is \" + a\n", nil, "p a", "c", "c")
	got := strings.Join(pausedLines(dbg), " ")
	if got != "2 3" {
		t.Fatalf("paused at %q\n%s", got, dbg)
	}
	if !strings.Contains(dbg, "a = 5") || !strings.Contains(dbg, "main:3: Assert failed: a is 5") && !strings.Contains(dbg, "prog.mb:3: Assert failed: a is 5") {
		t.Fatalf("pause output:\n%s", dbg)
	}
	// Without the debugger, DebugBreak does nothing.
	if out := run(t, "DebugBreak\nPrint \"ok\"\n"); strings.TrimSpace(out) != "ok" {
		t.Fatalf("got %q", out)
	}
}
