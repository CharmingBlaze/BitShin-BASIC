package runtime

import (
	"testing"

	"bitshinbasic/internal/value"
)

func nums(n ...float64) []value.Value {
	out := make([]value.Value, len(n))
	for i, v := range n {
		out[i] = value.Num(v)
	}
	return out
}

func TestParseSphereClassic(t *testing.T) {
	r, segs, parent := parseSphereArgs(nums(10, 3), func(id int) bool { return id == 3 })
	if r != 1 || segs != 10 || parent != 3 {
		t.Fatalf("classic CreateSphere(10, player) got r=%v segs=%d parent=%d", r, segs, parent)
	}
}

func TestParseSphereSegmentsNotParent(t *testing.T) {
	// CreateSphere(8) after entity 8 exists (Platform 64 coins).
	r, segs, parent := parseSphereArgs(nums(8), func(id int) bool { return id == 8 })
	if r != 1 || segs != 8 || parent != 0 {
		t.Fatalf("CreateSphere(8) must be 8 segments, not parent 8; got r=%v segs=%d parent=%d", r, segs, parent)
	}
}

func TestParseSphereModern(t *testing.T) {
	r, segs, parent := parseSphereArgs(nums(1.5, 16, 0), nil)
	if r != 1.5 || segs != 16 || parent != 0 {
		t.Fatalf("CreateSphere(1.5, 16, 0) got r=%v segs=%d parent=%d", r, segs, parent)
	}
}

func TestParseCylinderClassic(t *testing.T) {
	r, h, segs, caps, parent := parseCylinderArgs(nums(10, 3), func(id int) bool { return id == 3 })
	if r != 1 || h != 2 || segs != 10 || !caps || parent != 3 {
		t.Fatalf("classic CreateCylinder(10, player) got r=%v h=%v segs=%d caps=%v parent=%d", r, h, segs, caps, parent)
	}
}

func TestParseCylinderModern(t *testing.T) {
	r, h, segs, caps, parent := parseCylinderArgs(nums(0.5, 3, 16, 0, 0), nil)
	if r != 0.5 || h != 3 || segs != 16 || caps || parent != 0 {
		t.Fatalf("CreateCylinder(0.5, 3, 16, 0, 0) got r=%v h=%v segs=%d caps=%v parent=%d", r, h, segs, caps, parent)
	}
}

func TestFogFactorLinear(t *testing.T) {
	if f := fogFactor(1, 50, 0, 100, 0.02); f < 0.49 || f > 0.51 {
		t.Fatalf("linear mid fog want ~0.5 got %v", f)
	}
	if f := fogFactor(0, 50, 0, 100, 0.02); f != 1 {
		t.Fatalf("fog off want 1 got %v", f)
	}
}

func TestCommandTableFXGeom(t *testing.T) {
	m := New(".").commandTable()
	for _, name := range []string{
		"createskybox", "setweather", "createemitter2d", "particle2dspeed",
		"createbox", "createcapsule", "enablefog", "camerafogdensity",
	} {
		if m[name] == nil {
			t.Errorf("missing command %s", name)
		}
	}
}
