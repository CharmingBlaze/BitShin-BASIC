package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

func TestBootTrailerRoundTrip(t *testing.T) {
	body := []byte("MZ pretend binary")
	raw := appendBootTrailer(body, "script=main.mb\nprofile=release\n")
	if got := stripBootTrailer(raw); !bytes.Equal(got, body) {
		t.Fatalf("strip: got %q", got)
	}
	// Packaging a packaged game replaces its trailer rather than stacking one
	again := appendBootTrailer(stripBootTrailer(raw), "script=other.mb\n")
	if got := stripBootTrailer(again); !bytes.Equal(got, body) {
		t.Fatalf("restrip: got %q", got)
	}
	if got := stripBootTrailer(body); !bytes.Equal(got, body) {
		t.Fatal("a binary with no trailer changed")
	}
}

func TestSetPESubsystem(t *testing.T) {
	raw := make([]byte, 0x200)
	raw[0], raw[1] = 'M', 'Z'
	binary.LittleEndian.PutUint32(raw[0x3C:], 0x80)
	copy(raw[0x80:], "PE\x00\x00")
	off := 0x80 + 4 + 20 + 68
	if err := setPESubsystem(raw, true); err != nil {
		t.Fatal(err)
	}
	if v := binary.LittleEndian.Uint16(raw[off:]); v != 2 {
		t.Fatalf("gui subsystem = %d", v)
	}
	if err := setPESubsystem(raw, false); err != nil {
		t.Fatal(err)
	}
	if v := binary.LittleEndian.Uint16(raw[off:]); v != 3 {
		t.Fatalf("console subsystem = %d", v)
	}
	if err := setPESubsystem([]byte("not an exe at all, nowhere near long enough"), true); err == nil {
		t.Fatal("no error for a file that is not an exe")
	}
}

func TestPrebuiltWhenForced(t *testing.T) {
	t.Setenv("MOON_BUILD_PREBUILT", "1")
	if _, ok := canCompileFromSource(); ok {
		t.Fatal("MOON_BUILD_PREBUILT=1 still compiles from source")
	}
	os.Unsetenv("MOON_BUILD_PREBUILT")
}
