package audio

import (
	"fmt"
	"io"
	"sync"
)

// LoadStream checks that path decodes and returns a clip that decodes it from
// disk while it plays, a block at a time, instead of holding the whole PCM.
// Each play opens its own decoder; looping reopens the file at the end.
func LoadStream(path string) (*Clip, error) {
	r, c, err := openDecoded(path)
	if err != nil {
		return nil, fmt.Errorf("LoadMusic: %w", err)
	}
	var probe [FrameBytes]byte
	_, err = io.ReadFull(r, probe[:])
	c.Close()
	if err != nil {
		return nil, fmt.Errorf("LoadMusic: %s has no audio (%w)", path, err)
	}
	return &Clip{Stream: path}, nil
}

// streamReader feeds an Oto player from a live decoder, applying pitch (by
// stepping through frames at rate) and pan as it goes.
type streamReader struct {
	mu     sync.Mutex
	path   string
	dec    io.Reader
	closer io.Closer
	buf    []byte // decoded frames not yet played
	pos    float64
	rate   float64
	loop   bool
	left   float64
	right  float64
	panned bool
	done   bool
}

func newStreamReader(path string, rate, pan float64, loop bool) (*streamReader, error) {
	dec, closer, err := openDecoded(path)
	if err != nil {
		return nil, err
	}
	s := &streamReader{path: path, dec: dec, closer: closer, rate: rate, loop: loop}
	s.setPan(pan)
	return s, nil
}

func (s *streamReader) setPan(pan float64) {
	if pan < -1 {
		pan = -1
	}
	if pan > 1 {
		pan = 1
	}
	s.left = min(1, 1-pan)
	s.right = min(1, 1+pan)
	s.panned = pan != 0
}

func (s *streamReader) setRate(v float64) {
	if v <= 0 {
		v = 1
	}
	s.mu.Lock()
	s.rate = v
	s.mu.Unlock()
}

// fill decodes more frames into buf. It returns false at the end of a
// non-looping stream (or a looping one that yields nothing on reopen).
func (s *streamReader) fill() bool {
	if s.done || s.dec == nil {
		return false
	}
	drop := int(s.pos) * FrameBytes
	if drop > len(s.buf) {
		drop = len(s.buf) - len(s.buf)%FrameBytes
	}
	s.buf = append(s.buf[:0], s.buf[drop:]...)
	s.pos -= float64(drop / FrameBytes)
	var tmp [16 << 10]byte
	reopened := false
	for {
		n, err := s.dec.Read(tmp[:])
		s.buf = append(s.buf, tmp[:n]...)
		if n > 0 {
			reopened = false
		}
		if len(s.buf)/FrameBytes > int(s.pos) {
			return true
		}
		if err == nil {
			continue
		}
		if err != io.EOF || !s.loop || reopened {
			s.done = true
			return false
		}
		s.closer.Close()
		s.buf = s.buf[:len(s.buf)-len(s.buf)%FrameBytes]
		dec, closer, oerr := openDecoded(s.path)
		if oerr != nil {
			s.dec, s.closer, s.done = nil, nil, true
			return false
		}
		s.dec, s.closer = dec, closer
		reopened = true
	}
}

func (s *streamReader) Read(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rate := s.rate
	if rate <= 0 {
		rate = 1
	}
	n := 0
	for n+FrameBytes <= len(p) {
		i := int(s.pos) * FrameBytes
		if i+FrameBytes > len(s.buf) {
			if !s.fill() {
				break
			}
			i = int(s.pos) * FrameBytes
		}
		copy(p[n:n+FrameBytes], s.buf[i:i+FrameBytes])
		if s.panned {
			l := float64(int16(p[n]) | int16(p[n+1])<<8)
			r := float64(int16(p[n+2]) | int16(p[n+3])<<8)
			ls, rs := int16(clamp16(l*s.left)), int16(clamp16(r*s.right))
			p[n], p[n+1], p[n+2], p[n+3] = byte(ls), byte(ls>>8), byte(rs), byte(rs>>8)
		}
		s.pos += rate
		n += FrameBytes
	}
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

func (s *streamReader) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closer != nil {
		s.closer.Close()
		s.closer = nil
	}
	s.dec = nil
	s.done = true
}
