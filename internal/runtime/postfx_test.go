package runtime

import "testing"

func TestPostAndShaderCommandsRegistered(t *testing.T) {
	m := New(".").commandTable()
	for _, name := range []string{
		"enablepostfx", "setbloom", "setexposure", "setfxaa", "setcolorgrade", "postfx",
		"createshader", "loadshader", "setshader", "setshaderuniform", "shaderok",
		"mouselook", "stopanim", "setanimblend", "setsky", "savescene",
	} {
		if m[name] == nil {
			t.Errorf("missing command %s", name)
		}
	}
}
