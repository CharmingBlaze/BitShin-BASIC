package audio

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	ebaudio "github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/mewkiz/flac"
)

// openDecoded opens a .wav, .ogg, .mp3, or .flac file and returns a reader of
// stereo s16le PCM at SampleRate. The closer releases the file.
func openDecoded(path string) (io.Reader, io.Closer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	r, err := decodeStream(strings.ToLower(filepath.Ext(path)), f)
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	return r, f, nil
}

func decodeStream(ext string, src io.ReadSeeker) (io.Reader, error) {
	switch ext {
	case ".ogg":
		return vorbis.DecodeWithSampleRate(SampleRate, src)
	case ".mp3":
		return mp3.DecodeWithSampleRate(SampleRate, src)
	case ".flac":
		return decodeFLAC(src)
	default:
		s, err := wav.DecodeWithSampleRate(SampleRate, src)
		if err != nil {
			return nil, fmt.Errorf("need a .wav, .ogg, .mp3, or .flac file (%w)", err)
		}
		return s, nil
	}
}

// decodeFLAC streams FLAC frames as stereo s16le, resampled to SampleRate.
// Mono is copied to both sides; past two channels only front left/right play.
func decodeFLAC(src io.Reader) (io.Reader, error) {
	s, err := flac.New(src)
	if err != nil {
		return nil, err
	}
	if s.Info.NChannels == 0 || s.Info.SampleRate == 0 {
		return nil, fmt.Errorf("flac: bad stream info")
	}
	var r io.Reader = &flacReader{s: s, bps: int(s.Info.BitsPerSample)}
	rate := int(s.Info.SampleRate)
	if rate == SampleRate {
		return r, nil
	}
	size := int64(s.Info.NSamples) * FrameBytes
	if size == 0 {
		// Unknown length: the resampler needs one, so decode it all.
		pcm, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		r, size = bytes.NewReader(pcm), int64(len(pcm))
	}
	rs := ebaudio.ResampleReader(r, size, rate, SampleRate)
	// Over a plain reader the resampler pads its last read with silence;
	// cut it at the resampled length so loops stay seamless.
	if l, ok := rs.(interface{ Length() int64 }); ok {
		return io.LimitReader(rs, l.Length()/FrameBytes*FrameBytes), nil
	}
	return rs, nil
}

type flacReader struct {
	s   *flac.Stream
	bps int
	buf []byte
}

func (r *flacReader) Read(p []byte) (int, error) {
	for len(r.buf) == 0 {
		f, err := r.s.ParseNext()
		if err != nil {
			return 0, err
		}
		if len(f.Subframes) == 0 {
			continue
		}
		bps := int(f.BitsPerSample)
		if bps == 0 {
			bps = r.bps
		}
		left := f.Subframes[0].Samples
		right := left
		if len(f.Subframes) > 1 {
			right = f.Subframes[1].Samples
		}
		out := make([]byte, 0, len(left)*FrameBytes)
		for i := range left {
			l, rr := to16(left[i], bps), to16(right[i], bps)
			out = append(out, byte(l), byte(l>>8), byte(rr), byte(rr>>8))
		}
		r.buf = out
	}
	n := copy(p, r.buf)
	r.buf = r.buf[n:]
	return n, nil
}

func to16(s int32, bps int) int16 {
	switch {
	case bps > 16:
		return int16(s >> (bps - 16))
	case bps < 16:
		return int16(s << (16 - bps))
	}
	return int16(s)
}
