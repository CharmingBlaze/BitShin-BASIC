package runtime

import (
	"testing"

	"github.com/g3n/engine/window"
)

func TestKeyDownEscapeIgnoredUntilFlip(t *testing.T) {
	w := New(".")
	if w.keyDown(KeyEscape) {
		t.Fatal("Escape must be false before the first Flip")
	}
	w.keys[KeyEscape] = true
	if w.keyDown(KeyEscape) {
		t.Fatal("stuck Escape / KeyDown(1) must not skip the first game loop")
	}
	w.markFlip()
	if w.keyDown(KeyEscape) {
		t.Fatal("phantom Escape must be cleared on the first Flip")
	}
	w.setKey(window.KeyEscape, true)
	if !w.keyDown(KeyEscape) {
		t.Fatal("Escape should work after Flip")
	}
}

func TestSetKeyIgnoresEscapeBeforeFlip(t *testing.T) {
	w := New(".")
	w.setKey(window.KeyEscape, true)
	if w.keys[KeyEscape] || w.hits[KeyEscape] {
		t.Fatal("setKey must not record Escape before Flip")
	}
	w.markFlip()
	w.setKey(window.KeyEscape, true)
	if !w.keys[KeyEscape] || !w.hits[KeyEscape] {
		t.Fatal("setKey must record Escape after Flip")
	}
}

func TestWindowShouldCloseFalseUntilPresented(t *testing.T) {
	w := New(".")
	if w.windowWantsClose() {
		t.Fatal("WindowShouldClose must be false before Flip")
	}
	w.markFlip()
	if w.windowWantsClose() {
		t.Fatal("no GLFW window: close stays false, but must not be suppressed after Flip")
	}
}

func TestEscapeWorksAfterFlipEvenWithGUI(t *testing.T) {
	w := New(".")
	w.guiWantK = true
	w.markFlip()
	w.setKey(window.KeyEscape, true)
	if !w.keyDown(KeyEscape) || !w.keyHit(KeyEscape) {
		t.Fatal("Escape must work after Flip even if imgui wants the keyboard")
	}
}

func TestEscapeLatchIgnoresHeldPhantomThenHonorsPress(t *testing.T) {
	w := New(".")
	w.markFlip()
	w.setKey(window.KeyEscape, true)
	w.escapeLatch = true
	w.keys[KeyEscape] = false
	w.hits[KeyEscape] = false
	w.setKey(window.KeyEscape, true)
	if w.keyDown(KeyEscape) || w.keyHit(KeyEscape) {
		t.Fatal("held phantom Escape must stay ignored until release")
	}
	w.setKey(window.KeyEscape, false)
	w.setKey(window.KeyEscape, true)
	if !w.keyDown(KeyEscape) || !w.keyHit(KeyEscape) {
		t.Fatal("a new Escape press after release must quit")
	}
}

func TestDrainPhantomQuitStopsAfterPresent(t *testing.T) {
	w := New(".")
	w.markFlip()
	w.setKey(window.KeyEscape, true)
	w.drainPhantomQuit()
	if w.keyDown(KeyEscape) || w.keyHit(KeyEscape) {
		t.Fatal("pre-presentOK drain must drop phantom Escape")
	}
	w.presentOK = true
	w.setKey(window.KeyEscape, true)
	w.drainPhantomQuit()
	if !w.keyDown(KeyEscape) || !w.keyHit(KeyEscape) {
		t.Fatal("after first present, Esc must work and must not be drained")
	}
}

func TestKeyDownSpaceIgnoredUntilFlip(t *testing.T) {
	w := New(".")
	w.keys[KeySpace] = true
	if w.keyDown(KeySpace) {
		t.Fatal("KeyDown(SPACE) must be false before the first Flip")
	}
	w.markFlip()
	if w.keyDown(KeySpace) {
		t.Fatal("phantom Space must be cleared on the first Flip")
	}
	w.setKey(window.KeySpace, true)
	if !w.keyDown(KeySpace) {
		t.Fatal("Space should work after Flip when it was not latched")
	}
}

func TestSetKeyIgnoresSpaceBeforeFlip(t *testing.T) {
	w := New(".")
	w.setKey(window.KeySpace, true)
	if w.keys[KeySpace] || w.hits[KeySpace] {
		t.Fatal("setKey must not record Space before Flip")
	}
	w.markFlip()
	w.setKey(window.KeySpace, true)
	if !w.keys[KeySpace] || !w.hits[KeySpace] {
		t.Fatal("setKey must record Space after Flip when it was not latched")
	}
}

func TestLatchHeldSpaceUntilRelease(t *testing.T) {
	w := New(".")
	w.markFlip()
	w.keys[KeySpace] = true
	w.hits[KeySpace] = true
	w.latchHeldKeys()
	if w.keyDown(KeySpace) || w.keyHit(KeySpace) {
		t.Fatal("Space held at first present must stay ignored")
	}
	w.setKey(window.KeySpace, true)
	if w.keyDown(KeySpace) || w.keyHit(KeySpace) {
		t.Fatal("held phantom Space must stay ignored until release")
	}
	w.setKey(window.KeySpace, false)
	w.setKey(window.KeySpace, true)
	if !w.keyDown(KeySpace) || !w.keyHit(KeySpace) {
		t.Fatal("a new Space press after release must grip")
	}
}

func TestLatchHeldEscapeBlocksKeyHit(t *testing.T) {
	w := New(".")
	w.markFlip()
	w.keys[KeyEscape] = true
	w.hits[KeyEscape] = true
	w.latchHeldKeys()
	if w.keyDown(KeyEscape) || w.keyHit(KeyEscape) {
		t.Fatal("Escape held at first present must not End the program")
	}
	w.setKey(window.KeyEscape, false)
	w.setKey(window.KeyEscape, true)
	if !w.keyDown(KeyEscape) || !w.keyHit(KeyEscape) {
		t.Fatal("a new Escape after release must quit")
	}
}

func TestPollEbitenDoesNotKeepEscapeBeforeFlip(t *testing.T) {
	w := New(".")
	w.keys[KeyEscape] = true
	w.hits[KeyEscape] = true
	if w.keyDown(KeyEscape) {
		t.Fatal("2D/3D KeyDown(1) must be false before Flip even if keys map is set")
	}
	w.markFlip()
	w.keys[KeyEscape] = true
	if !w.keyDown(KeyEscape) {
		t.Fatal("KeyDown(1) should work after Flip")
	}
}
