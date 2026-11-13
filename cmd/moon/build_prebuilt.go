package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/tc-hib/winres"
	winver "github.com/tc-hib/winres/version"
)

// Prebuilt packaging: with no Go toolchain or no moonBASIC source tree (the
// SDK as shipped), moon build copies its own binary instead of compiling one.
// The boot script and profile go into a trailer at the end of the file, which
// main reads at startup in place of the -X settings a source build stamps.
//
//	<payload: key=value lines> <payload length: uint64 LE> "MOONBOOT"

const bootMagic = "MOONBOOT"

// sourceRoot is the moonBASIC source root that go build can compile from:
// a folder with go.mod and cmd/moon, found from the working directory or the
// running binary. Empty in the SDK.
func sourceRoot() string {
	for _, start := range []string{".", exeDir()} {
		root := findGoModRoot(start)
		if root == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, "cmd", "moon", "main.go")); err == nil {
			return root
		}
	}
	return ""
}

// canCompileFromSource reports whether moon build can run go build, and from
// which folder.
func canCompileFromSource() (string, bool) {
	if os.Getenv("MOON_BUILD_PREBUILT") != "" {
		return "", false
	}
	root := sourceRoot()
	if root == "" {
		return "", false
	}
	if _, err := exec.LookPath("go"); err != nil {
		return "", false
	}
	return root, true
}

// writePrebuiltBinary writes the game binary from the running moon binary.
func writePrebuiltBinary(o buildOpts, binPath string) error {
	if o.targetOS != runtime.GOOS || o.targetArch != runtime.GOARCH {
		return fmt.Errorf("moon build: this moonBASIC builds games for %s/%s only; build on the target OS, or install Go and build from the moonBASIC source (see docs/RELEASE.md)", runtime.GOOS, runtime.GOARCH)
	}
	if tags := joinTags(o.prof.tags, o.tags); tags != "" {
		fmt.Fprintf(os.Stderr, "moon build: warning: build tags %q need Go and the moonBASIC source; packaging this moon binary as it is\n", tags)
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(self)
	if err != nil {
		return err
	}
	raw = stripBootTrailer(raw)
	if o.targetOS == "windows" {
		if raw, err = patchWindowsExe(raw, o); err != nil {
			return fmt.Errorf("moon build: %w", err)
		}
	}
	var payload strings.Builder
	fmt.Fprintf(&payload, "script=%s\nprofile=%s\n", filepath.Base(o.src), o.profile)
	if o.prof.headless {
		payload.WriteString("headless=1\n")
	}
	raw = appendBootTrailer(raw, payload.String())
	if err := os.MkdirAll(filepath.Dir(binPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(binPath, raw, 0o755)
}

func appendBootTrailer(raw []byte, payload string) []byte {
	out := make([]byte, 0, len(raw)+len(payload)+16)
	out = append(out, raw...)
	out = append(out, payload...)
	out = binary.LittleEndian.AppendUint64(out, uint64(len(payload)))
	return append(out, bootMagic...)
}

// stripBootTrailer drops a trailer, so a packaged game can itself package.
func stripBootTrailer(raw []byte) []byte {
	if n, ok := bootTrailerLen(raw[max(0, len(raw)-16):]); ok && int(n)+16 <= len(raw) {
		return raw[:len(raw)-16-int(n)]
	}
	return raw
}

func bootTrailerLen(tail []byte) (uint64, bool) {
	if len(tail) != 16 || string(tail[8:]) != bootMagic {
		return 0, false
	}
	return binary.LittleEndian.Uint64(tail[:8]), true
}

// readBootTrailer fills bootScript, bootProfile, and bootHeadless from the
// running binary's trailer, if it has one.
func readBootTrailer() {
	self, err := os.Executable()
	if err != nil {
		return
	}
	f, err := os.Open(self)
	if err != nil {
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.Size() < 16 {
		return
	}
	tail := make([]byte, 16)
	if _, err := f.ReadAt(tail, st.Size()-16); err != nil {
		return
	}
	n, ok := bootTrailerLen(tail)
	if !ok || n == 0 || n > 4096 || int64(n)+16 > st.Size() {
		return
	}
	payload := make([]byte, n)
	if _, err := f.ReadAt(payload, st.Size()-16-int64(n)); err != nil && err != io.EOF {
		return
	}
	for _, line := range strings.Split(string(payload), "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch k {
		case "script":
			bootScript = v
		case "profile":
			bootProfile = v
		case "headless":
			bootHeadless = v
		}
	}
}

// patchWindowsExe sets the icon and version info, and the GUI subsystem
// (no console window) when the profile drops the console.
func patchWindowsExe(raw []byte, o buildOpts) ([]byte, error) {
	rs := winres.ResourceSet{}
	if o.icon != "" {
		icon, err := loadIcon(o.icon)
		if err != nil {
			return nil, fmt.Errorf("icon %s: %w", o.icon, err)
		}
		if err := rs.SetIcon(winres.Name("APP"), icon); err != nil {
			return nil, fmt.Errorf("icon %s: %w", o.icon, err)
		}
	}
	rs.SetVersionInfo(versionInfo(o))
	var out bytes.Buffer
	if err := rs.WriteToEXE(&out, bytes.NewReader(raw)); err != nil {
		return nil, fmt.Errorf("resources: %w", err)
	}
	raw = out.Bytes()
	gui := !o.prof.console && !o.prof.headless
	if err := setPESubsystem(raw, gui); err != nil {
		return nil, err
	}
	return raw, nil
}

// versionInfo is the version resource Explorer shows under Details.
func versionInfo(o buildOpts) winver.Info {
	vi := winver.Info{}
	fields := map[string]string{
		winver.ProductName:      o.appName,
		winver.FileDescription:  o.appName,
		winver.CompanyName:      o.company,
		winver.InternalName:     o.exeName,
		winver.OriginalFilename: o.exeName + ".exe",
		winver.Comments:         "Made with moonBASIC",
	}
	for k, v := range fields {
		if v != "" {
			_ = vi.Set(winver.LangDefault, k, v)
		}
	}
	vi.SetFileVersion(o.version)
	vi.SetProductVersion(o.version)
	return vi
}

// setPESubsystem writes the optional header's Subsystem field: 2 is a
// Windows GUI program (no console), 3 a console program. The checksum is
// left alone; Windows checks it only for drivers.
func setPESubsystem(raw []byte, gui bool) error {
	if len(raw) < 0x40 || raw[0] != 'M' || raw[1] != 'Z' {
		return fmt.Errorf("not a Windows executable")
	}
	pe := int(binary.LittleEndian.Uint32(raw[0x3C:]))
	off := pe + 4 + 20 + 68
	if pe <= 0 || off+2 > len(raw) || string(raw[pe:pe+4]) != "PE\x00\x00" {
		return fmt.Errorf("not a PE executable")
	}
	var v uint16 = 3
	if gui {
		v = 2
	}
	binary.LittleEndian.PutUint16(raw[off:], v)
	return nil
}
