package glslang

import "testing"

const tinyVert = `#version 330 core
void main() { gl_Position = vec4(0.0); }
`

const tinyComp = `#version 430
layout(local_size_x = 64) in;
void main() {}
`

func TestValidateVert(t *testing.T) {
	r := Validate(tinyVert, StageVert)
	if !r.OK {
		t.Fatal(r.Log)
	}
}

func TestValidateCompute(t *testing.T) {
	r := Validate(tinyComp, StageComp)
	if !r.OK {
		t.Fatal(r.Log)
	}
}

func TestValidateRejectsEmpty(t *testing.T) {
	r := Validate("", StageVert)
	if r.OK {
		t.Fatal("expected fail")
	}
}

func TestValidateRejectsNoVersion(t *testing.T) {
	r := Validate("void main() {}", StageFrag)
	if r.OK {
		t.Fatal("expected missing #version")
	}
}

func TestCompileAlwaysValidates(t *testing.T) {
	r := Compile(tinyVert, StageVert)
	if !r.OK {
		t.Fatal(r.Log)
	}
	// SPIR-V is optional; empty is fine without glslangValidator.
	if HasNative() && len(r.SPIRV) == 0 {
		t.Log("validator on PATH but no SPIR-V (still ok)")
	}
}

func TestParseStage(t *testing.T) {
	if ParseStage("vertex") != StageVert || ParseStage("compute") != StageComp {
		t.Fatal(ParseStage("vertex"), ParseStage("compute"))
	}
}
