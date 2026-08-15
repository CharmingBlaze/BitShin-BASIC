package runtime

import (
	"testing"

	"bitshinbasic/internal/value"
)

func TestPBRCommandsRegistered(t *testing.T) {
	m := New(".").commandTable()
	need := []string{
		"createpbrmaterial", "setmaterialpbr", "enablepbr", "getmaterialpbr",
		"setmaterial", "setpbrmaterial",
		"setalbedo", "setbasecolor", "setalbedomap", "setbasecolormap",
		"getalbedor", "getbasecolorr", "getalbedog", "getbasecolorg", "getalbedob", "getbasecolorb",
		"setmetallic", "getmetallic", "setmetallicfactor", "getmetallicfactor",
		"setroughness", "getroughness", "setroughnessfactor", "getroughnessfactor",
		"setao", "getao", "setocclusion", "getocclusion", "setocclusionfactor", "getocclusionfactor",
		"setemissive", "getemissiver", "getemissiveg", "getemissiveb",
		"setnormalmap", "setmetallicroughnessmap", "setmetalroughmap",
		"setemissivemap", "setaomap", "setocclusionmap",
		"setenvmap", "setibl", "getibl", "enableibl", "setiblintensity", "getiblintensity",
	}
	for _, name := range need {
		if m[name] == nil {
			t.Errorf("missing PBR command %s", name)
		}
	}
}

func TestCreatePBRMaterialNoGL(t *testing.T) {
	w := New(".")
	mat, err := w.Call("createpbrmaterial", nil)
	if err != nil {
		t.Fatal(err)
	}
	id := mat.Int()
	if id < 1 {
		t.Fatalf("CreatePBRMaterial handle %d", id)
	}
	on, err := w.Call("getmaterialpbr", []value.Value{value.Num(float64(id))})
	if err != nil || on.Int() != 1 {
		t.Fatalf("GetMaterialPBR: %v %v", on, err)
	}
	if _, err := w.Call("setmetallic", []value.Value{value.Num(float64(id)), value.Num(0.8)}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("setroughness", []value.Value{value.Num(float64(id)), value.Num(0.25)}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("setao", []value.Value{value.Num(float64(id)), value.Num(0.4)}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("setalbedo", []value.Value{value.Num(float64(id)), value.Num(10), value.Num(20), value.Num(30)}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("setemissive", []value.Value{value.Num(float64(id)), value.Num(8), value.Num(16), value.Num(24)}); err != nil {
		t.Fatal(err)
	}
	metal, err := w.Call("getmetallic", []value.Value{value.Num(float64(id))})
	if err != nil || metal.Number() < 0.79 || metal.Number() > 0.81 {
		t.Fatalf("GetMetallic %v %v", metal, err)
	}
	rough, err := w.Call("getroughness", []value.Value{value.Num(float64(id))})
	if err != nil || rough.Number() < 0.24 || rough.Number() > 0.26 {
		t.Fatalf("GetRoughness %v %v", rough, err)
	}
	ao, err := w.Call("getao", []value.Value{value.Num(float64(id))})
	if err != nil || ao.Number() < 0.39 || ao.Number() > 0.41 {
		t.Fatalf("GetAO %v %v", ao, err)
	}
	r, err := w.Call("getalbedor", []value.Value{value.Num(float64(id))})
	if err != nil || r.Int() != 10 {
		t.Fatalf("GetAlbedoR %v %v", r, err)
	}
	er, err := w.Call("getemissiver", []value.Value{value.Num(float64(id))})
	if err != nil || er.Int() != 8 {
		t.Fatalf("GetEmissiveR %v %v", er, err)
	}
	if _, err := w.Call("setibl", []value.Value{value.Num(0)}); err != nil {
		t.Fatal(err)
	}
	ibl, err := w.Call("getibl", nil)
	if err != nil || ibl.Int() != 0 {
		t.Fatalf("GetIBL %v %v", ibl, err)
	}
}

func TestPBRShaderIsGLSL330(t *testing.T) {
	for _, src := range []string{mbphysicalVertex, mbphysicalFragment} {
		if containsVersion45(src) {
			t.Fatal("mbphysical must stay GLSL 330; found a 4.5 version directive")
		}
	}
}

func TestCausticsUseTextureLod(t *testing.T) {
	need := "textureLod(WaterCaustic"
	for name, src := range map[string]string{
		"mbphysical": mbphysicalFragment,
		"mbterrain":  mbterrainFragment,
		"bsshadow":   mbshadowFragment,
	} {
		if !containsStr(src, need) {
			t.Fatalf("%s caustics must sample WaterCaustic with textureLod", name)
		}
		if containsStr(src, "texture(WaterCaustic") {
			t.Fatalf("%s still uses texture() for WaterCaustic", name)
		}
	}
}

func TestAtmoMarchHasPixelJitter(t *testing.T) {
	if !containsStr(mbatmoFragment, "gl_FragCoord.xy") {
		t.Fatal("mbatmo march must jitter t0 with gl_FragCoord")
	}
	if !containsStr(mbatmoFragment, "t0 +=") {
		t.Fatal("mbatmo must offset t0 before the 12-step march")
	}
}

func containsVersion45(s string) bool {
	return len(s) > 0 && (containsStr(s, "#version 450") || containsStr(s, "#version 440") || containsStr(s, "#version 430") || containsStr(s, "#version 420") || containsStr(s, "#version 410") || containsStr(s, "#version 400"))
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(findSub(s, sub)) > 0)
}

func findSub(s, sub string) string {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return sub
		}
	}
	return ""
}
