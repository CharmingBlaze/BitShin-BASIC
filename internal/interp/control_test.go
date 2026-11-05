package interp

import (
	"strings"
	"testing"

	"moonbasic/internal/parse"
)

func TestGotoGosubRemAndComments(t *testing.T) {
	out := run(t, `
n = 0
Goto skip
n = 9
skip:
Print n
Gosub bump
Print n
Rem this stays a comment Print 7
x = 1 /* hidden */ + 2
Print x
End
bump:
n = n + 3
Return
`)
	if out != "0\n3\n3\n" && out != "0\r\n3\r\n3\r\n" {
		t.Fatalf("output %q", out)
	}
}

func TestLocalAsKeepsTheType(t *testing.T) {
	out := run(t, `
Local n As Integer = 3.9
n = n + 1.2
Print n
Local s As String = 12
Print s
Local b As Bool = 4
Print b
Function Hit()
  Static k As Integer = 0.9
  k = k + 1.2
  Return k
End Function
Print Hit()
Print Hit()
`)
	want := "4\n12\n1\n1\n2\n"
	if out != want && out != "4\r\n12\r\n1\r\n1\r\n2\r\n" {
		t.Fatalf("output %q", out)
	}
}

func TestFunctionReturnStillReturnsValue(t *testing.T) {
	out := run(t, `
Function Add(a, b)
  Return a + b
End Function
Print Add(2, 3)
`)
	if out != "5\n" && out != "5\r\n" {
		t.Fatalf("output %q", out)
	}
}

func TestTransitionsAndUtility(t *testing.T) {
	out := run(t, `
Function Ready()
  Return 1
End Function
Function Idle()
End Function
Function Go()
End Function
m = CreateStateMachine()
AddState m, "idle", "Idle"
AddState m, "go", "Go"
AddTransition m, "idle", "go", "Ready"
UpdateState m
Print StateName(m)
u = CreateUtility()
UtilityConsider u, "idle", 1
UtilityConsider u, "attack", 5
Print UtilityChoose(u)
`)
	if out != "go\nattack\n" && out != "go\r\nattack\r\n" {
		t.Fatalf("output %q", out)
	}
}

func TestFuncStatsRecordsCall(t *testing.T) {
	prog, err := parse.Parse(`
Function Work()
  x = 1
End Function
Work()
`)
	if err != nil {
		t.Fatal(err)
	}
	in := New(prog, nopHost{})
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, s := range in.FuncStats() {
		if s.Name == "work" && s.Calls == 1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("stats %+v", in.FuncStats())
	}
}

func TestDimWithATypeSuffixIsStillAnArray(t *testing.T) {
	out := run(t, `Dim a#(3)
a(1) = 1.5
a#(2) = 2.5
Dim n$(1)
n(0) = "hi"
Print a(1) + a(2)
Print n$(0)
`)
	if !strings.Contains(out, "4") || !strings.Contains(out, "hi") {
		t.Fatalf("got %q", out)
	}
}
