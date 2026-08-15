package runtime

import (
	"testing"

	"bitshinbasic/internal/value"
)

func strs(ss ...string) []value.Value {
	out := make([]value.Value, len(ss))
	for i, s := range ss {
		out[i] = value.Str(s)
	}
	return out
}

func TestGLModernCommandsRegistered(t *testing.T) {
	m := New(".").commandTable()
	need := []string{
		"glversion", "glmajor", "glminor", "glrenderer",
		"glhascompute", "glhasssbo", "glhasubo", "glhasinstancing",
		"glhasgeometry", "glhastessellation", "glfeature",
		"createcomputeshader", "dispatchcompute", "computelog",
		"createstoragebuffer", "setstoragebuffer", "getstoragebuffer",
		"bindstoragebuffer", "storagebuffersize",
		"createuniformbuffer", "setuniformbuffer", "getuniformbuffer",
		"binduniformbuffer",
		"createinstancedmesh", "setinstancetransform", "setinstancedata",
		"instancecount", "batchinstances", "instanceentity",
		"enablegpuinstances", "gpuinstances",
		"creategeompoints", "setgeompoint", "geompointcount",
		"enabletessellation", "tessellation",
		"compileshader", "shaderlog", "shaderspirvsize", "glslangnative",
		"noisefbm", "sheval", "jobxform",
	}
	for _, name := range need {
		if m[name] == nil {
			t.Errorf("missing %s", name)
		}
	}
}

func TestParseGLVersion(t *testing.T) {
	maj, min := parseGLVersion("4.6.0 NVIDIA")
	if maj != 4 || min != 6 {
		t.Fatalf("got %d.%d", maj, min)
	}
	maj, min = parseGLVersion("3.3.0")
	if maj != 3 || min != 3 {
		t.Fatalf("got %d.%d", maj, min)
	}
}

func TestCompileShaderViaTable(t *testing.T) {
	w := New(".")
	fn := w.commandTable()["compileshader"]
	if fn == nil {
		t.Fatal("compileshader")
	}
	v, err := fn(strs(`#version 330 core
void main() { gl_Position = vec4(0.0); }
`, "vert"))
	if err != nil {
		t.Fatal(err)
	}
	if v.Num < 1 {
		t.Fatalf("handle %v", v.Num)
	}
	logFn := w.commandTable()["shaderlog"]
	lg, err := logFn(nums(v.Num))
	if err != nil {
		t.Fatal(err)
	}
	if lg.Str == "" {
		t.Fatal("empty log")
	}
}

func TestNoiseFBMCommand(t *testing.T) {
	fn := New(".").commandTable()["noisefbm"]
	v, err := fn(nums(1.25, 3.5, 4))
	if err != nil {
		t.Fatal(err)
	}
	if v.Num < -1.2 || v.Num > 1.2 {
		t.Fatalf("fbm %v", v.Num)
	}
}

func TestDispatchComputeNoContext(t *testing.T) {
	fn := New(".").commandTable()["dispatchcompute"]
	v, err := fn(nums(1, 1, 1, 1))
	if err != nil {
		t.Fatal(err)
	}
	if v.Num != 0 {
		t.Fatalf("expected 0 skip, got %v", v.Num)
	}
}
