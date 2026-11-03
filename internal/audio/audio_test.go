package audio

import (
	"encoding/binary"
	"testing"
)

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

func TestRateReaderPanAndSeek(t *testing.T) {
	pcm := make([]byte, 20*FrameBytes)
	binary.LittleEndian.PutUint16(pcm[10*FrameBytes:], 1000)
	binary.LittleEndian.PutUint16(pcm[10*FrameBytes+2:], 1000)
	r := &rateReader{pcm: pcm, rate: 1}
	r.setPan(1)
	r.setPos(10)
	buf := make([]byte, FrameBytes)
	if _, err := r.Read(buf); err != nil {
		t.Fatal(err)
	}
	l := int16(binary.LittleEndian.Uint16(buf[0:2]))
	right := int16(binary.LittleEndian.Uint16(buf[2:4]))
	if l != 0 || right != 1000 {
		t.Fatalf("pan right l=%d r=%d", l, right)
	}
	if r.position() < 10.9 || r.position() > 11.1 {
		t.Fatalf("pos %v", r.position())
	}
}
