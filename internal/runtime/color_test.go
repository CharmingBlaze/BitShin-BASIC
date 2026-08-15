package runtime

import "testing"

func TestRGBDualScale(t *testing.T) {
	c := rgb(255, 128, 0)
	if abs32(c.R-1) > 1e-5 || abs32(c.G-128.0/255) > 1e-5 || c.B != 0 {
		t.Fatalf("0-255: got %v", *c)
	}
	c = rgb(0.5, 0, 0)
	if abs32(c.R-0.5) > 1e-5 || c.G != 0 || c.B != 0 {
		t.Fatalf("0-1 mid-red: got %v", *c)
	}
	c = rgb(1, 1, 1)
	if c.R != 1 || c.G != 1 || c.B != 1 {
		t.Fatalf("1,1,1 is white: got %v", *c)
	}
	c = rgb(0.24, 0.28, 0.36)
	if abs32(c.R-0.24) > 1e-5 || abs32(c.G-0.28) > 1e-5 || abs32(c.B-0.36) > 1e-5 {
		t.Fatalf("ambient 0-1: got %v", *c)
	}
	c = rgb(70, 82, 100)
	if abs32(c.R-70.0/255) > 1e-5 {
		t.Fatalf("ambient 0-255: got %v", *c)
	}
	c = rgb(2, 0, 0)
	if abs32(c.R-2.0/255) > 1e-5 {
		t.Fatalf("any channel >1 is 0-255: got %v", *c)
	}

	b := rgbBytes(255, 0, 128)
	if b[0] != 255 || b[1] != 0 || b[2] != 128 {
		t.Fatalf("rgbBytes 0-255: got %v", b)
	}
	b = rgbBytes(0.5, 0, 0)
	if b[0] < 127 || b[0] > 128 || b[1] != 0 || b[2] != 0 {
		t.Fatalf("rgbBytes 0-1 mid-red: got %v", b)
	}
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
