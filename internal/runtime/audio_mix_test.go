package runtime

import "testing"

func TestFadeAndBusGain(t *testing.T) {
	w := New(t.TempDir())
	w.sounds[1] = &sndSlot{vol: 1, pitch: 1}
	w.fadeSound(1, 0, 1)
	w.delta = 0.5
	w.tickFades()
	got := w.sounds[1].vol
	if got < 0.49 || got > 0.51 {
		t.Fatalf("fade vol %v", got)
	}
	w.setAudioBus("Music", 0.25)
	w.soundBus(1, "music")
	if g := w.busGain("Music"); g != 0.25 {
		t.Fatalf("bus %v", g)
	}
	w.master = 0.5
	if w.sounds[1].bus != "music" {
		t.Fatalf("bus name %q", w.sounds[1].bus)
	}
}
