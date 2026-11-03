package runtime

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestShaderCacheRoundTrip(t *testing.T) {
	c := &shaderDiskCache{dir: filepath.Join(t.TempDir(), "a", "b")}
	if _, _, ok := c.Load("k"); ok {
		t.Fatal("empty cache returned a program")
	}
	c.Store("k", []byte{1, 2, 3}, 0xABCD)
	data, format, ok := c.Load("k")
	if !ok || format != 0xABCD || !bytes.Equal(data, []byte{1, 2, 3}) {
		t.Fatalf("got %v %x %v", data, format, ok)
	}
	left, _ := os.ReadDir(c.dir)
	if len(left) != 1 {
		t.Fatalf("a temp file was left behind: %v", left)
	}
}

func TestShaderCacheIgnoresATruncatedFile(t *testing.T) {
	c := &shaderDiskCache{dir: t.TempDir()}
	if err := os.WriteFile(c.path("k"), []byte{1, 2}, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := c.Load("k"); ok {
		t.Fatal("a truncated file is not a program")
	}
}

func TestShaderCacheFolderFollowsTheDriver(t *testing.T) {
	a := shaderCacheDir("root", "AMD", "780M", "4.6.1")
	if a == shaderCacheDir("root", "AMD", "780M", "4.6.2") {
		t.Fatal("a driver update must start a new folder")
	}
	if shaderCacheDir("", "AMD", "780M", "1") != "" {
		t.Fatal("no root means no cache")
	}
}
