package runtime

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/g3n/engine/gls"
)

// Compiled shaders on disk, so only a game's first launch on a machine pays to
// compile them. A driver may refuse a stored binary at any time, a driver update
// being the usual reason. gls then compiles from source and stores the new one, so
// a stale cache costs one launch and never a broken picture. The folder is keyed
// on the GPU and driver strings, so an update starts a fresh one.
//
// MOON_NO_SHADER_CACHE=1 turns this off. MOON_SHADER_LOG=1 prints what each program
// cost; the same totals are in StatsShaderMs.

type shaderDiskCache struct{ dir string }

// shaderCacheDir is where this GPU and driver's programs live, or "" if there is
// nowhere to write.
func shaderCacheDir(root, vendor, renderer, version string) string {
	if root == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(vendor + "\x00" + renderer + "\x00" + version))
	return filepath.Join(root, "moonbasic", "shaders", hex.EncodeToString(sum[:8]))
}

func (c *shaderDiskCache) path(key string) string { return filepath.Join(c.dir, key+".bin") }

func (c *shaderDiskCache) Load(key string) ([]byte, uint32, bool) {
	raw, err := os.ReadFile(c.path(key))
	if err != nil || len(raw) <= 4 {
		return nil, 0, false
	}
	return raw[4:], binary.LittleEndian.Uint32(raw), true
}

// Store writes beside the target and renames, so two games starting together can
// never read half a file.
func (c *shaderDiskCache) Store(key string, data []byte, format uint32) {
	if len(data) == 0 || os.MkdirAll(c.dir, 0o755) != nil {
		return
	}
	raw := make([]byte, 4+len(data))
	binary.LittleEndian.PutUint32(raw, format)
	copy(raw[4:], data)
	tmp, err := os.CreateTemp(c.dir, "new-*")
	if err != nil {
		return
	}
	_, werr := tmp.Write(raw)
	cerr := tmp.Close()
	if werr != nil || cerr != nil || os.Rename(tmp.Name(), c.path(key)) != nil {
		os.Remove(tmp.Name())
	}
}

// shaderStats totals what building programs cost this run.
var shaderStats struct {
	ms              float64
	built, fromDisk int
}

// startShaderCache runs once the GL context is current.
func startShaderCache(gs *gls.GLS) {
	logIt := os.Getenv("MOON_SHADER_LOG") != ""
	gls.ProgramBuilt = func(key string, took time.Duration, cached bool) {
		ms := float64(took.Microseconds()) / 1000
		shaderStats.ms += ms
		if cached {
			shaderStats.fromDisk++
		} else {
			shaderStats.built++
		}
		if logIt {
			fmt.Printf("shader %s %8.2f ms cached=%v\n", key[:8], ms, cached)
		}
	}
	if os.Getenv("MOON_NO_SHADER_CACHE") != "" || gs == nil || !gs.BinarySupported() {
		return
	}
	root, err := os.UserCacheDir()
	if err != nil {
		return
	}
	dir := shaderCacheDir(root, gs.GetString(gls.VENDOR), gs.GetString(gls.RENDERER), gs.GetString(gls.VERSION))
	if dir != "" {
		gls.ProgramCache = &shaderDiskCache{dir: dir}
	}
}
