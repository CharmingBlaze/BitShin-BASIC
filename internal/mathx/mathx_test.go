package mathx

import (
	"math"
	"testing"

	"github.com/g3n/engine/math32"
	"gonum.org/v1/gonum/mat"
)

func TestVecRoundTrip(t *testing.T) {
	v := math32.Vector3{1.5, -2, 3}
	g := FromVec3(v)
	back := ToVec3(g)
	if math.Abs(float64(back.X-v.X)) > 1e-6 || math.Abs(float64(back.Y-v.Y)) > 1e-6 || math.Abs(float64(back.Z-v.Z)) > 1e-6 {
		t.Fatalf("roundtrip %v -> %v", v, back)
	}
}

func TestMat4Identity(t *testing.T) {
	var id math32.Matrix4
	id.Identity()
	g := FromMat4(id)
	back := ToMat4(g)
	for i := 0; i < 16; i++ {
		if math.Abs(float64(back[i]-id[i])) > 1e-5 {
			t.Fatalf("mat[%d] %v %v", i, back[i], id[i])
		}
	}
}

func TestMul4Identity(t *testing.T) {
	a := mat.NewDense(4, 4, nil)
	for i := 0; i < 4; i++ {
		a.Set(i, i, 1)
	}
	b := mat.NewDense(4, 4, []float64{
		2, 0, 0, 0,
		0, 3, 0, 0,
		0, 0, 4, 0,
		0, 0, 0, 1,
	})
	out := Mul4(a, b)
	if out.At(1, 1) != 3 {
		t.Fatalf("mul %v", out.At(1, 1))
	}
}

func TestBatchMul4(t *testing.T) {
	src := make([]float64, 32)
	for i := 0; i < 2; i++ {
		src[i*16+0] = 1
		src[i*16+5] = 1
		src[i*16+10] = 1
		src[i*16+15] = 1
	}
	right := mat.NewDense(4, 4, nil)
	for i := 0; i < 4; i++ {
		right.Set(i, i, 2)
	}
	out := BatchMul4(src, 2, right)
	if len(out) != 32 || out[0] != 2 || out[16+5] != 2 {
		t.Fatalf("batch %v", out[:8])
	}
}

func TestFBMRange(t *testing.T) {
	n := FBM(1.25, 3.5, 4, 0.5, 2)
	if n < -1.1 || n > 1.1 {
		t.Fatalf("fbm out of range %v", n)
	}
}

func TestSHEvalUp(t *testing.T) {
	sh := ProjectHemisphereSH(1, 1, 1, 0, 0, 0)
	r, _, _ := EvalSH(sh, 0, 1, 0)
	if r <= 0 {
		t.Fatalf("expected sky contribution, got %v", r)
	}
}

func TestSeparate2D(t *testing.T) {
	xs := []float64{0, 0.2}
	zs := []float64{0, 0}
	ax, _ := Separate2D(xs, zs, 0, 1)
	if ax >= 0 {
		t.Fatalf("expected push left, got %v", ax)
	}
}
