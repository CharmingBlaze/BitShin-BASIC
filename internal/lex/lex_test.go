package lex

import "testing"

func TestCommentsAndKeys(t *testing.T) {
	lx := New(`; comment
x = 1 // hi
' also
If x <> 2 Then`)
	var kinds []Kind
	for {
		tkn := lx.Next()
		if tkn.Kind == EOF {
			break
		}
		if tkn.Kind != Newline {
			kinds = append(kinds, tkn.Kind)
		}
	}
	if len(kinds) < 6 {
		t.Fatalf("too few tokens: %v", kinds)
	}
}

func TestDotField(t *testing.T) {
	lx := New(`v.x = 1.5`)
	var kinds []Kind
	for {
		tkn := lx.Next()
		if tkn.Kind == EOF {
			break
		}
		kinds = append(kinds, tkn.Kind)
	}
	if len(kinds) < 5 || kinds[1] != Dot || kinds[4] != Number {
		t.Fatalf("tokens %v", kinds)
	}
}

func TestHexLiteral(t *testing.T) {
	lx := New(`c = $FFECc8`)
	var kinds []Kind
	var hexLit string
	for {
		tkn := lx.Next()
		if tkn.Kind == EOF {
			break
		}
		if tkn.Kind == Hex {
			hexLit = tkn.Lit
		}
		if tkn.Kind != Newline {
			kinds = append(kinds, tkn.Kind)
		}
	}
	if hexLit != "FFECc8" {
		t.Fatalf("hex lit %q kinds %v", hexLit, kinds)
	}
}

func TestIdentKey(t *testing.T) {
	if IdentKey("Yaw#") != "yaw" {
		t.Fatal(IdentKey("Yaw#"))
	}
	if IdentKey("Name$") != "name" {
		t.Fatal(IdentKey("Name$"))
	}
}
