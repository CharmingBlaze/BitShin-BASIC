package audio

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"
)

// writeWAV writes 16-bit stereo PCM frames as a .wav at rate.
func writeWAV(t *testing.T, path string, rate int, frames [][2]int16) {
	t.Helper()
	var b bytes.Buffer
	data := len(frames) * 4
	b.WriteString("RIFF")
	binary.Write(&b, binary.LittleEndian, uint32(36+data))
	b.WriteString("WAVEfmt ")
	for _, v := range []any{uint32(16), uint16(1), uint16(2), uint32(rate), uint32(rate * 4), uint16(4), uint16(16)} {
		binary.Write(&b, binary.LittleEndian, v)
	}
	b.WriteString("data")
	binary.Write(&b, binary.LittleEndian, uint32(data))
	for _, f := range frames {
		binary.Write(&b, binary.LittleEndian, f)
	}
	if err := os.WriteFile(path, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeFLAC encodes int32 samples per channel with the flac encoder.
func writeFLAC(t *testing.T, path string, rate int, chans [][]int32) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	n := len(chans[0])
	info := &meta.StreamInfo{
		BlockSizeMin: 16, BlockSizeMax: uint16(n), SampleRate: uint32(rate),
		NChannels: uint8(len(chans)), BitsPerSample: 16, NSamples: uint64(n),
	}
	enc, err := flac.NewEncoder(f, info)
	if err != nil {
		t.Fatal(err)
	}
	layout := frame.ChannelsMono
	if len(chans) == 2 {
		layout = frame.ChannelsLR
	}
	fr := &frame.Frame{Header: frame.Header{
		HasFixedBlockSize: true, BlockSize: uint16(n), SampleRate: uint32(rate),
		Channels: layout, BitsPerSample: 16,
	}}
	for _, c := range chans {
		fr.Subframes = append(fr.Subframes, &frame.Subframe{
			SubHeader: frame.SubHeader{Pred: frame.PredVerbatim},
			Samples:   c, NSamples: n,
		})
	}
	if err := enc.WriteFrame(fr); err != nil {
		t.Fatal(err)
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}
}

func ramp(n int) [][2]int16 {
	out := make([][2]int16, n)
	for i := range out {
		out[i] = [2]int16{int16(i * 10), int16(-i * 10)}
	}
	return out
}

func TestLoadWAVAndStream(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.wav")
	writeWAV(t, path, SampleRate, ramp(1000))
	clip, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(clip.PCM) != 1000*FrameBytes {
		t.Fatalf("pcm %d bytes", len(clip.PCM))
	}

	sc, err := LoadStream(path)
	if err != nil || sc.Stream != path || sc.PCM != nil {
		t.Fatalf("stream clip %+v %v", sc, err)
	}
	s, err := newStreamReader(path, 1, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(s)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	if !bytes.Equal(got, clip.PCM) {
		t.Fatalf("streamed %d bytes, decoded %d; content differs", len(got), len(clip.PCM))
	}
}

func TestStreamLoopsPitchAndPan(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.wav")
	writeWAV(t, path, SampleRate, ramp(300))
	full, _ := Load(path)

	s, err := newStreamReader(path, 1, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 700*FrameBytes)
	if _, err := io.ReadFull(s, buf); err != nil {
		t.Fatal(err)
	}
	s.Close()
	// Frame 650 is frame 50 of the third pass.
	if !bytes.Equal(buf[650*FrameBytes:651*FrameBytes], full.PCM[50*FrameBytes:51*FrameBytes]) {
		t.Fatal("loop did not wrap to the start")
	}

	s, _ = newStreamReader(path, 2, 0, false)
	fast, _ := io.ReadAll(s)
	s.Close()
	if len(fast) != 150*FrameBytes {
		t.Fatalf("pitch 2 played %d frames, want 150", len(fast)/FrameBytes)
	}

	s, _ = newStreamReader(path, 1, 1, false)
	right, _ := io.ReadAll(s)
	s.Close()
	l := int16(binary.LittleEndian.Uint16(right[100*FrameBytes:]))
	r := int16(binary.LittleEndian.Uint16(right[100*FrameBytes+2:]))
	if l != 0 || r != -1000 {
		t.Fatalf("pan right: l=%d r=%d", l, r)
	}

	if _, err := LoadStream(filepath.Join(dir, "missing.ogg")); err == nil {
		t.Fatal("missing file should fail")
	}
}

func TestLoadFLAC(t *testing.T) {
	dir := t.TempDir()
	// Stereo at the mix rate: samples come through exactly.
	l, r := make([]int32, 400), make([]int32, 400)
	for i := range l {
		l[i], r[i] = int32(i*7), int32(-i*3)
	}
	path := filepath.Join(dir, "s.flac")
	writeFLAC(t, path, SampleRate, [][]int32{l, r})
	clip, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(clip.PCM) != 400*FrameBytes {
		t.Fatalf("pcm %d bytes", len(clip.PCM))
	}
	if v := int16(binary.LittleEndian.Uint16(clip.PCM[123*FrameBytes+2:])); v != -369 {
		t.Fatalf("right sample %d", v)
	}

	// Mono at half rate: doubled in length, copied to both sides.
	m := make([]int32, 500)
	for i := range m {
		m[i] = 1000
	}
	path = filepath.Join(dir, "m.flac")
	writeFLAC(t, path, SampleRate/2, [][]int32{m})
	clip, err = Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(clip.PCM) / FrameBytes; n < 990 || n > 1010 {
		t.Fatalf("resampled to %d frames, want ~1000", n)
	}
	mid := clip.PCM[500*FrameBytes:]
	if !bytes.Equal(mid[0:2], mid[2:4]) {
		t.Fatal("mono should play on both sides")
	}

	s, err := newStreamReader(path, 1, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	streamed, _ := io.ReadAll(s)
	s.Close()
	if len(streamed) != len(clip.PCM) {
		t.Fatalf("streamed %d bytes, loaded %d", len(streamed), len(clip.PCM))
	}
}

// silentMP3 is n MPEG-1 Layer III frames (128 kbit/s, 44.1 kHz) whose side
// info and main data are all zero, which decodes to silence.
func silentMP3(n int) []byte {
	const size = 144 * 128000 / 44100
	var b []byte
	for i := 0; i < n; i++ {
		f := make([]byte, size)
		f[0], f[1], f[2], f[3] = 0xFF, 0xFB, 0x90, 0x00
		b = append(b, f...)
	}
	return b
}

func TestLoadMP3(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.mp3")
	if err := os.WriteFile(path, silentMP3(8), 0o644); err != nil {
		t.Fatal(err)
	}
	clip, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(clip.PCM) / FrameBytes; n < 1152 {
		t.Fatalf("decoded %d frames", n)
	}
	if _, err := LoadStream(path); err != nil {
		t.Fatal(err)
	}
}
