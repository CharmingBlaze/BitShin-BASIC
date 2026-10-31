package main

import (
	"archive/zip"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestScanAssetsFindsIncludesAndMissing(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"game.mb": `Include "lib/util.mb"
tex = LoadTexture("gfx/wall.png")
snd = LoadSound("sfx/gone.wav")
SaveScreenshot "shot.png"
; LoadMesh "old/commented.glb"
`,
		"lib/util.mb":     `m = LoadMesh("models/ship.glb")` + "\n" + `f = LoadFont("missing.ttf")`,
		"gfx/wall.png":    "png",
		"models/ship.glb": "glb",
	})
	s, err := scanAssets(filepath.Join(dir, "game.mb"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(s.scripts, ",") != "game.mb,lib/util.mb" {
		t.Fatalf("scripts %v", s.scripts)
	}
	sort.Strings(s.assets)
	if strings.Join(s.assets, ",") != "gfx/wall.png,models/ship.glb" {
		t.Fatalf("assets %v", s.assets)
	}
	var miss []string
	for _, m := range s.missing {
		miss = append(miss, m.file+":"+m.path)
	}
	if strings.Join(miss, ",") != "lib/util.mb:missing.ttf,game.mb:sfx/gone.wav" {
		t.Fatalf("missing %v (SaveScreenshot output and comments must not count)", miss)
	}
	if s.missing[1].line != 3 {
		t.Fatalf("line %d", s.missing[1].line)
	}
}

func TestWritePakHoldsAssets(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"a.png": "A", "assets/b.wav": "BB"})
	pak := filepath.Join(dir, "out", "game.pak")
	if err := writePak(pak, dir, []string{"a.png", "assets/b.wav"}); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(pak)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	got := map[string]uint64{}
	for _, f := range zr.File {
		got[f.Name] = f.UncompressedSize64
	}
	if len(got) != 2 || got["a.png"] != 1 || got["assets/b.wav"] != 2 {
		t.Fatalf("pak entries %v", got)
	}
}

func TestResolveProfiles(t *testing.T) {
	rel, err := resolveProfile("release", nil)
	if err != nil || !rel.pak || !rel.strip || rel.console {
		t.Fatalf("release %+v %v", rel, err)
	}
	dev, _ := resolveProfile("dev", nil)
	if dev.pak || dev.strip || !dev.console {
		t.Fatalf("dev %+v", dev)
	}
	if _, err := resolveProfile("nope", nil); err == nil {
		t.Fatal("unknown profile should fail")
	}
	yes, no := true, false
	custom := map[string]buildProfile{
		"release": {Console: &yes},
		"server":  {Base: "release", Headless: &yes, Pak: &no, Tags: "enet"},
		"loop":    {Base: "loop"},
	}
	rel, _ = resolveProfile("release", custom)
	if !rel.pak || !rel.console {
		t.Fatalf("moon.json release override %+v", rel)
	}
	srv, err := resolveProfile("server", custom)
	if err != nil || !srv.headless || srv.pak || !srv.strip || srv.tags != "enet" || !srv.console {
		t.Fatalf("server %+v %v", srv, err)
	}
	if _, err := resolveProfile("loop", custom); err == nil {
		t.Fatal("self-based profile should fail")
	}
}

func TestParseBuildArgsReadsMoonJSON(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"main.mb": "Print 1",
		"moon.json": `{"name": "Star Game", "build": {"outputDir": "out", "icon": "icon.png", "version": "2.3.4",
			"company": "Moon Co", "profile": "release", "profiles": {"release": {"strict": true}}}}`,
	})
	o, err := parseBuildArgs([]string{filepath.Join(dir, "main.mb"), "-loose", "-version", "9.9.9"})
	if err != nil {
		t.Fatal(err)
	}
	if o.profile != "release" || o.prof.pak || !o.prof.strict || !o.prof.strip {
		t.Fatalf("profile %s %+v", o.profile, o.prof)
	}
	if o.exeName != "Star Game" || o.appName != "Star Game" || o.version != "9.9.9" || o.company != "Moon Co" {
		t.Fatalf("metadata %+v", o)
	}
	if o.outDir != filepath.Join(dir, "out") || o.icon != filepath.Join(dir, "icon.png") {
		t.Fatalf("paths %s %s", o.outDir, o.icon)
	}
	if _, err := parseBuildArgs([]string{"main.mb", "-bogus"}); err == nil {
		t.Fatal("unknown flag should fail")
	}
}

func TestGoBuildArgsPerProfile(t *testing.T) {
	o := buildOpts{src: "game.mb", targetOS: "windows", exeName: "game", profile: "release", prof: builtinProfiles["release"]}
	got := strings.Join(goBuildArgs(o, "dist/game.exe"), " ")
	for _, want := range []string{"-trimpath", "-s -w", "-H windowsgui", "-X 'main.bootScript=game.mb'", "-X 'main.bootProfile=release'"} {
		if !strings.Contains(got, want) {
			t.Fatalf("release args lack %q: %s", want, got)
		}
	}
	o.prof.headless = true
	o.tags = "enet"
	got = strings.Join(goBuildArgs(o, "dist/game.exe"), " ")
	if strings.Contains(got, "windowsgui") || !strings.Contains(got, "main.bootHeadless=1") || !strings.Contains(got, "-tags enet") {
		t.Fatalf("headless args: %s", got)
	}
	o = buildOpts{src: "game.mb", targetOS: "windows", exeName: "moon", profile: "dev", prof: builtinProfiles["dev"]}
	got = strings.Join(goBuildArgs(o, "dist/moon.exe"), " ")
	if strings.Contains(got, "windowsgui") || strings.Contains(got, "-s -w") {
		t.Fatalf("dev args: %s", got)
	}
}

func TestWindowsResourcesObject(t *testing.T) {
	dir := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 40, 30))
	for i := range img.Pix {
		img.Pix[i] = 200
	}
	img.Set(3, 3, color.NRGBA{255, 0, 0, 255})
	f, err := os.Create(filepath.Join(dir, "icon.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()
	o := buildOpts{targetArch: "amd64", exeName: "game", appName: "Game", version: "1.2.3", company: "Co", icon: filepath.Join(dir, "icon.png")}
	obj, err := windowsResources(o)
	if err != nil {
		t.Fatal(err)
	}
	if len(obj) < 20 || binary.LittleEndian.Uint16(obj) != 0x8664 {
		t.Fatalf("not an amd64 COFF object (%d bytes)", len(obj))
	}
	// VERSIONINFO strings are UTF-16.
	if !strings.Contains(string(obj), "1\x00.\x002\x00.\x003\x00") {
		t.Fatal("version string not embedded")
	}
	o.icon = filepath.Join(dir, "none.png")
	if _, err := windowsResources(o); err == nil {
		t.Fatal("missing icon should fail")
	}
}

func TestSafeFileName(t *testing.T) {
	if got := safeFileName(` My:Game?. `); got != "MyGame" {
		t.Fatalf("got %q", got)
	}
}
