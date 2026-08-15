package audio

import "testing"

func TestGenerateThunder(t *testing.T) {
	c := GenerateThunder(0.5)
	if c == nil || len(c.PCM) < SampleRate/4 {
		t.Fatalf("thunder pcm %d", len(c.PCM))
	}
}

func TestDistancePan(t *testing.T) {
	vol, pan := DistancePan(0, 0, 0, 0, 0, 0, 10, 40)
	if vol <= 0 || vol > 1 {
		t.Fatalf("forward vol=%v", vol)
	}
	if pan < -0.05 || pan > 0.05 {
		t.Fatalf("forward pan=%v want ~0", pan)
	}
	_, pan = DistancePan(0, 0, 0, 0, 10, 0, 0, 40)
	if pan <= 0 {
		t.Fatalf("right pan=%v want >0", pan)
	}
}
