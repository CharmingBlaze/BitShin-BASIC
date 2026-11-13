package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/tc-hib/winres"
)

// sysoName is the resource object moon build drops into cmd/moon for one
// go build; the _windows_<arch> suffix keeps it out of other targets.
func sysoName(arch string) string {
	return "zz_moonbuild_windows_" + arch + ".syso"
}

// windowsResources builds the icon and version info Explorer shows for the
// game binary, as a COFF object the Go linker embeds.
func windowsResources(o buildOpts) ([]byte, error) {
	var arch winres.Arch
	switch o.targetArch {
	case "amd64":
		arch = winres.ArchAMD64
	case "arm64":
		arch = winres.ArchARM64
	case "386":
		arch = winres.ArchI386
	default:
		return nil, fmt.Errorf("no Windows resources for -arch %s", o.targetArch)
	}
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
	var buf bytes.Buffer
	if err := rs.WriteObject(&buf, arch); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeWindowsResources writes the resource object into cmd/moon and returns
// its path; the caller removes it after go build.
func writeWindowsResources(o buildOpts, root string) (string, error) {
	obj, err := windowsResources(o)
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, "cmd", "moon", sysoName(o.targetArch))
	if err := os.WriteFile(path, obj, 0o644); err != nil {
		return "", fmt.Errorf("resources: %w (run moon build from the moonBASIC source root)", err)
	}
	return path, nil
}

// loadIcon reads an .ico as is, or makes one from a PNG / JPEG / GIF
// resized to 256, 64, 48, 32, and 16 pixels.
func loadIcon(path string) (*winres.Icon, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if strings.EqualFold(filepath.Ext(path), ".ico") {
		return winres.LoadICO(f)
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return winres.NewIconFromResizedImage(img, nil)
}
