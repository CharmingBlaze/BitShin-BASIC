package audio

import "testing"

func TestGeneratedMotionClips(t *testing.T) {
	for _, c := range []*Clip{GenerateEngine(), GenerateFootstep(), GenerateCloth()} {
		if c == nil || len(c.PCM) < FrameBytes || len(c.PCM)%FrameBytes != 0 {
			t.Fatalf("clip pcm %d", len(c.PCM))
		}
	}
}
