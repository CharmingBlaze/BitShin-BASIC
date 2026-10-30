// Package joltstage copies the Jolt 5.4 static libraries from the module
// cache into third_party/jolt so a Linux or macOS link does not depend on
// GOPATH. A custom GOMODCACHE is read from the environment, then from go env.
package joltstage

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const module = "github.com/bbitechnologies/jolt-go@v0.8.4"

// Platforms that link internal/jolt against the jolt-go static libraries.
var Platforms = []string{"linux_amd64", "linux_arm64", "darwin_arm64"}

// Stage copies libJolt.a and libjolt_wrapper.a for each platform when the
// destination file is missing. root is the module directory that contains go.mod.
func Stage(root string) error {
	cache, err := modCache()
	if err != nil {
		return err
	}
	srcRoot := filepath.Join(cache, filepath.FromSlash(module), "jolt", "lib")
	var missing []string
	for _, platform := range Platforms {
		srcDir := filepath.Join(srcRoot, platform)
		dstDir := filepath.Join(root, "third_party", "jolt", platform)
		for _, name := range []string{"libJolt.a", "libjolt_wrapper.a"} {
			dst := filepath.Join(dstDir, name)
			if st, err := os.Stat(dst); err == nil && st.Size() > 0 {
				continue
			}
			src := filepath.Join(srcDir, name)
			if _, err := os.Stat(src); err != nil {
				missing = append(missing, platform+"/"+name)
				continue
			}
			if err := copyFile(src, dst); err != nil {
				return err
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("jolt libraries not in the module cache (%s); run go mod download: %s", srcRoot, strings.Join(missing, ", "))
	}
	return nil
}

func modCache() (string, error) {
	if v := strings.TrimSpace(os.Getenv("GOMODCACHE")); v != "" {
		return v, nil
	}
	out, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		return "", fmt.Errorf("go env GOMODCACHE: %w", err)
	}
	v := strings.TrimSpace(string(out))
	if v == "" {
		return "", fmt.Errorf("GOMODCACHE is empty")
	}
	return v, nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
