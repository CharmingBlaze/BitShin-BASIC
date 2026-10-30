package runtime

import (
	"testing"

	"github.com/g3n/engine/animation"
)

func TestLocomotionNamesAndSpeed(t *testing.T) {
	idle, walk := locomotionClips([]string{"Idle_Breath", "Walk_Fwd", "Wave"})
	if idle != 0 || walk != 1 {
		t.Fatalf("clips idle=%d walk=%d", idle, walk)
	}
	idle, walk = locomotionClips([]string{"seq0"})
	if idle != 0 || walk != 0 {
		t.Fatalf("single clip idle=%d walk=%d", idle, walk)
	}

	w := New(".")
	id := w.createCubeMesh(nil)
	w.ents[id].anim = &animState{
		clips:  []*animation.Animation{nil, nil, nil},
		names:  []string{"Idle_Breath", "Walk_Fwd", "Wave"},
		first:  []float32{0, 0, 0},
		last:   []float32{1, 1, 1},
		length: []float32{1, 1, 1},
		dir:    1,
		speed:  1,
		mode:   1,
		blend:  1,
		prev:   -1,
	}
	w.driveLocomotion(id, 3.2, 0)
	st := w.ents[id].anim
	if st.seq != 1 || st.mode != 1 || st.speed < 1.5 || st.speed > 2.2 {
		t.Fatalf("walk seq=%d mode=%d speed=%v", st.seq, st.mode, st.speed)
	}
	w.driveLocomotion(id, 0.05, 0)
	if st.seq != 0 || st.mode != 1 {
		t.Fatalf("idle seq=%d mode=%d", st.seq, st.mode)
	}
	st.seq = 2
	st.mode = 3
	w.driveLocomotion(id, 4, 0)
	if st.seq != 2 || st.mode != 3 {
		t.Fatalf("one-shot was replaced seq=%d mode=%d", st.seq, st.mode)
	}
}
