package runtime

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type pakFS struct {
	zr  *zip.ReadCloser
	tmp string
}

func openPakFile(path string) (*pakFS, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("OpenPak: %w", err)
	}
	tmp, err := os.MkdirTemp("", "mb-pak-*")
	if err != nil {
		zr.Close()
		return nil, err
	}
	return &pakFS{zr: zr, tmp: tmp}, nil
}

func (p *pakFS) close() {
	if p == nil {
		return
	}
	_ = p.zr.Close()
	_ = os.RemoveAll(p.tmp)
}

func (p *pakFS) extract(rel string) (string, error) {
	want := strings.ReplaceAll(rel, "\\", "/")
	want = strings.TrimPrefix(want, "./")
	var f *zip.File
	for _, e := range p.zr.File {
		name := strings.ReplaceAll(e.Name, "\\", "/")
		if name == want || strings.HasSuffix(name, "/"+want) {
			f = e
			break
		}
	}
	if f == nil {
		return "", fmt.Errorf("OpenPak: %s not in archive", rel)
	}
	out := filepath.Join(p.tmp, filepath.FromSlash(want))
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return "", err
	}
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()
	dst, err := os.Create(out)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(dst, rc)
	_ = dst.Close()
	return out, err
}
