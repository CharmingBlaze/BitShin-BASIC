// Package glslang validates GLSL and optionally emits SPIR-V.
//
// Default (no CGO native lib): syntax/stage checks in pure Go, plus
// glslangValidator on PATH for SPIR-V. OpenGL compile is done by the
// runtime when a 3.3 context exists (CompileShader).
//
// Native Khronos glslang is not shipped (painful on Windows). Build with
// -tags glslang only if you add a local C binding; this package still
// works without that tag. Vulkan is never required.
package glslang

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Stage is a shader pipeline stage name (vert, frag, comp, geom, tesc, tese).
type Stage string

const (
	StageVert Stage = "vert"
	StageFrag Stage = "frag"
	StageComp Stage = "comp"
	StageGeom Stage = "geom"
	StageTesc Stage = "tesc"
	StageTese Stage = "tese"
)

// Result is a validation / optional SPIR-V compile.
type Result struct {
	OK    bool
	Log   string
	Stage Stage
	SPIRV []uint32 // empty unless glslangValidator (or -tags glslang) produced it
}

// ParseStage maps command strings to a stage.
func ParseStage(s string) Stage {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, ".")
	switch s {
	case "vert", "vertex", "vs":
		return StageVert
	case "frag", "fragment", "fs", "pixel":
		return StageFrag
	case "comp", "compute", "cs":
		return StageComp
	case "geom", "geometry", "gs":
		return StageGeom
	case "tesc", "tesscontrol", "ctrl", "tcs":
		return StageTesc
	case "tese", "tesseval", "eval", "tes":
		return StageTese
	default:
		return Stage(s)
	}
}

// Validate checks GLSL source for a stage. Always runs; no GL context needed.
func Validate(src string, stage Stage) Result {
	src = strings.TrimSpace(src)
	r := Result{Stage: stage}
	if src == "" {
		r.Log = "empty shader"
		return r
	}
	if !strings.Contains(src, "#version") {
		r.Log = "missing #version (use 330 for the 3.3 ship path; 430 for compute)"
		return r
	}
	need := stageTokens(stage)
	for _, tok := range need.must {
		if !strings.Contains(src, tok) {
			r.Log = fmt.Sprintf("stage %s: expected %q in source", stage, tok)
			return r
		}
	}
	if need.forbid != "" && strings.Contains(src, need.forbid) {
		r.Log = fmt.Sprintf("stage %s: unexpected %q", stage, need.forbid)
		return r
	}
	if strings.Count(src, "{") != strings.Count(src, "}") {
		r.Log = "unbalanced braces"
		return r
	}
	r.OK = true
	r.Log = "ok (glslang validate)"
	return r
}

type stageNeed struct {
	must   []string
	forbid string
}

func stageTokens(s Stage) stageNeed {
	switch s {
	case StageVert:
		return stageNeed{must: []string{"void main"}}
	case StageFrag:
		return stageNeed{must: []string{"void main"}}
	case StageComp:
		return stageNeed{must: []string{"void main", "local_size"}}
	case StageGeom:
		return stageNeed{must: []string{"void main", "layout"}}
	case StageTesc, StageTese:
		return stageNeed{must: []string{"void main"}}
	default:
		return stageNeed{must: []string{"void main"}}
	}
}

// Compile validates, then tries SPIR-V via glslangValidator if present.
func Compile(src string, stage Stage) Result {
	r := Validate(src, stage)
	if !r.OK {
		return r
	}
	if bin, log, err := emitSPIRV(src, stage); err == nil && len(bin) > 0 {
		r.SPIRV = bin
		r.Log = "ok (glslangValidator SPIR-V)"
		return r
	} else if err != nil && log != "" {
		r.Log = r.Log + "; SPIR-V optional: " + log
	}
	return r
}

// HasNative reports whether a SPIR-V emitter is available (validator on PATH
// or a -tags glslang binding).
func HasNative() bool {
	if _, err := exec.LookPath("glslangValidator"); err == nil {
		return true
	}
	if _, err := exec.LookPath("glslang"); err == nil {
		return true
	}
	return nativeBinding()
}

func nativeBinding() bool { return false }

func emitSPIRV(src string, stage Stage) ([]uint32, string, error) {
	exe, err := exec.LookPath("glslangValidator")
	if err != nil {
		exe, err = exec.LookPath("glslang")
	}
	if err != nil {
		return nil, "glslangValidator not on PATH (SPIR-V optional)", err
	}
	dir, err := os.MkdirTemp("", "mb-glslang-")
	if err != nil {
		return nil, err.Error(), err
	}
	defer os.RemoveAll(dir)
	in := filepath.Join(dir, "shader."+string(stage))
	out := filepath.Join(dir, "out.spv")
	if err := os.WriteFile(in, []byte(src), 0o600); err != nil {
		return nil, err.Error(), err
	}
	cmd := exec.Command(exe, "-V", "-S", string(stage), "-o", out, in)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return nil, strings.TrimSpace(buf.String()), err
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		return nil, err.Error(), err
	}
	if len(raw) < 4 || len(raw)%4 != 0 {
		return nil, "invalid SPIR-V size", fmt.Errorf("spirv size %d", len(raw))
	}
	words := make([]uint32, len(raw)/4)
	for i := 0; i < len(words); i++ {
		words[i] = uint32(raw[i*4]) | uint32(raw[i*4+1])<<8 | uint32(raw[i*4+2])<<16 | uint32(raw[i*4+3])<<24
	}
	if words[0] != 0x07230203 {
		return nil, "missing SPIR-V magic", fmt.Errorf("magic %x", words[0])
	}
	return words, "", nil
}

// CompileFile reads a file and Compile()s it.
func CompileFile(path, stage string) Result {
	b, err := os.ReadFile(path)
	if err != nil {
		return Result{Log: err.Error(), Stage: ParseStage(stage)}
	}
	return Compile(string(b), ParseStage(stage))
}
