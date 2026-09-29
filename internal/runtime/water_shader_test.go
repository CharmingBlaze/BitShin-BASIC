package runtime

import (
	"testing"

	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
)

func TestWaterShaderIsGLSL330Scenic(t *testing.T) {
	for _, src := range []string{mbwaterVertex, mbwaterFragment} {
		if containsVersion45(src) {
			t.Fatal("mbwater must stay GLSL 330; found a 4.x version directive")
		}
	}
	need := []string{
		"ClipSpace", "WaterDuDv", "WaterRefract", "WaterReflect",
		"WaterNormal", "WaterMove", "WaterSunDir", "WaterWind",
		"WaterRefractDepth", "out vec4 FragColor",
		"dFdx", "dFdy", "dx_pos", "dy_pos", "gl_FrontFacing", "deepWaterColor", "shallowWaterColor",
		"512.0", "sunGlint", "foamStart", "foamNoise", "6.0", "depthMix", "finalRGB",
		"Time", "WaveSpeed", "0.90", "detail * 0.15", "tex * 1.5",
		"WaterSSROn", "WaterWake", "WakeSpan",
		"texture(WaterRefract", "PointLightPosition", "SpotLightPosition", "WaterReflectivity",
	}
	for _, s := range need {
		if !containsStr(mbwaterFragment, s) && !containsStr(mbwaterVertex, s) {
			t.Fatalf("mbwater missing scenic/Gerstner term %s", s)
		}
	}
}

func TestWaterDuDvTileable(t *testing.T) {
	img := genWaterDuDv(32)
	if img.Bounds().Dx() != 32 || img.Bounds().Dy() != 32 {
		t.Fatal(img.Bounds())
	}
	c0 := img.RGBAAt(0, 4)
	c1 := img.RGBAAt(31, 4)
	dr := int(c0.R) - int(c1.R)
	if dr < 0 {
		dr = -dr
	}
	if dr > 90 {
		t.Fatalf("dudv not wrap-ish: %v vs %v", c0, c1)
	}
}

func TestApplyLitShadersKeepsWater(t *testing.T) {
	w := New(".")
	mat := material.NewStandard(&math32.Color{0.03, 0.16, 0.24})
	bindWaterMaterial(mat)
	if w.ents == nil {
		w.ents = map[int]*Entity{}
	}
	w.ents[1] = &Entity{name: "water", mat: mat}
	w.shadow.on = true
	w.applyLitShaders()
	if mat.Shader() != "mbwater" {
		t.Fatalf("applyLitShaders stole mbwater: %s", mat.Shader())
	}
}

func TestWaterMaterialUpdateAdvancesTime(t *testing.T) {
	wb := &waterBody{speed: 0.04, nWaves: 4, waves: defaultWaves()}
	m := NewWaterMaterial(nil, wb)
	wb.mat = m
	m.Update(0.1)
	if m.Time < 0.099 || m.Time > 0.11 {
		t.Fatalf("Time=%v", m.Time)
	}
	if m.clock() != m.Time*m.WaveSpeed {
		t.Fatalf("clock=%v want %v", m.clock(), m.Time*m.WaveSpeed)
	}
	if m.Shader() != "mbwater" || !m.Transparent() {
		t.Fatal("WaterMaterial must stay mbwater + transparent")
	}
}

func TestWaterStyleCommandsRegistered(t *testing.T) {
	m := New(".").commandTable()
	for _, name := range []string{
		"setwaterstyle", "setwaterwind", "getwaterwind", "setwatermode", "setwaterwavespeed", "getwaterwavespeed",
		"setwatercaustics", "getwatercaustics", "setwaterssr", "getwaterssr", "setwaterambientsound",
		"enablewatercaustics", "enablewaterssr", "getweatherintensity",
	} {
		if m[name] == nil {
			t.Errorf("missing %s", name)
		}
	}
}

func TestWaterLODGridFillsHorizon(t *testing.T) {
	g := newWaterLODGrid(2200, 64)
	pos := g.VBO(gls.VertexPosition)
	if pos == nil {
		t.Fatal("no position vbo")
	}
	buf := pos.Buffer()
	if buf == nil || buf.Len() < 3 {
		t.Fatal("empty lod grid")
	}
	var minX, maxX float32 = 1e9, -1e9
	for i := 0; i+2 < buf.Len(); i += 3 {
		x := (*buf)[i]
		if x < minX {
			minX = x
		}
		if x > maxX {
			maxX = x
		}
	}
	if maxX-minX < 4000 {
		t.Fatalf("lod span too small: %v .. %v (128-cap grids cannot fill horizon)", minX, maxX)
	}
	xs, _ := waterLODCoords(2200, 64)
	mid := len(xs) / 2
	if mid < 1 || xs[mid+1]-xs[mid] > 0.75 {
		t.Fatalf("camera cell too coarse: %v", xs[mid+1]-xs[mid])
	}
}

func TestWaterCausticTile(t *testing.T) {
	img := genWaterCaustic(32)
	if img.Bounds().Dx() != 32 {
		t.Fatal(img.Bounds())
	}
	bright := 0
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			if img.RGBAAt(x, y).R > 80 {
				bright++
			}
		}
	}
	if bright < 8 {
		t.Fatalf("caustic tex too dark (%d bright texels)", bright)
	}
}
