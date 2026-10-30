package runtime

import (
	"testing"

	"moonbasic/internal/value"
)

func TestFileRoundTrip(t *testing.T) {
	w := New(t.TempDir())
	missing, err := w.Call("readfile", []value.Value{value.Str("missing.txt")})
	if err != nil {
		t.Fatal(err)
	}
	if missing.Int() != 0 {
		t.Fatalf("missing file handle %v", missing)
	}

	f, err := w.Call("writefile", []value.Value{value.Str("notes.txt")})
	if err != nil || f.Int() == 0 {
		t.Fatalf("writefile %v %v", f, err)
	}
	if _, err := w.Call("writeline", []value.Value{f, value.Str("hello")}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("writeint", []value.Value{f, value.Num(42)}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("writefloat", []value.Value{f, value.Num(1.5)}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("writestring", []value.Value{f, value.Str("ok")}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("closefile", []value.Value{f}); err != nil {
		t.Fatal(err)
	}

	size, err := w.Call("filesize", []value.Value{value.Str("notes.txt")})
	if err != nil || size.Int() <= 0 {
		t.Fatalf("filesize %v %v", size, err)
	}

	r, err := w.Call("readfile", []value.Value{value.Str("notes.txt")})
	if err != nil || r.Int() == 0 {
		t.Fatalf("readfile %v %v", r, err)
	}
	line, err := w.Call("readline", []value.Value{r})
	if err != nil || line.String() != "hello" {
		t.Fatalf("line %q %v", line.String(), err)
	}
	n, err := w.Call("readint", []value.Value{r})
	if err != nil || n.Int() != 42 {
		t.Fatalf("int %v %v", n, err)
	}
	fl, err := w.Call("readfloat", []value.Value{r})
	if err != nil || fl.Number() != 1.5 {
		t.Fatalf("float %v %v", fl, err)
	}
	s, err := w.Call("readstring", []value.Value{r})
	if err != nil || s.String() != "ok" {
		t.Fatalf("string %q %v", s.String(), err)
	}
	eof, err := w.Call("eof", []value.Value{r})
	if err != nil || eof.Int() != 1 {
		t.Fatalf("eof %v %v", eof, err)
	}
	if _, err := w.Call("seekfile", []value.Value{r, value.Num(0)}); err != nil {
		t.Fatal(err)
	}
	again, err := w.Call("readline", []value.Value{r})
	if err != nil || again.String() != "hello" {
		t.Fatalf("seek line %q %v", again.String(), err)
	}
	if _, err := w.Call("closefile", []value.Value{r}); err != nil {
		t.Fatal(err)
	}
}
