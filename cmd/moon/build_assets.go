package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// assetExt matches a quoted path with a file type moon build packages.
var assetExt = regexp.MustCompile(`(?i)"([^"]+\.(png|jpg|jpeg|gif|bmp|tga|webp|dds|ktx|ktx2|hdr|wav|ogg|mp3|flac|obj|mtl|gltf|glb|dae|fbx|stl|ply|ttf|otf|csv|json|zip|pak|glsl|vert|frag|mb|bb))"`)

// outputCall matches a command before a quoted path that writes the file
// rather than reads it (SaveScreenshot "shot.png"), so it is not missing.
var outputCall = regexp.MustCompile(`(?i)\b(save|write|export|screenshot|delete|create(file|dir)|makedir|copyfile|renamefile)\w*`)

// assetScan is what a script and the scripts it includes refer to. Paths are
// relative to the main script's folder, with forward slashes, the way the
// runtime resolves them.
type assetScan struct {
	scripts []string // .mb files, the main script first
	assets  []string // every other file that exists
	missing []missingAsset
}

// missingAsset is a quoted path that names a file that is not there.
type missingAsset struct {
	file string // script, relative to the main script's folder
	line int
	path string
}

// scanAssets reads the main script and, recursively, the .mb files it names.
func scanAssets(main string) (*assetScan, error) {
	base := filepath.Dir(main)
	s := &assetScan{}
	seen := map[string]bool{}
	var walk func(rel string) error
	walk = func(rel string) error {
		raw, err := os.ReadFile(filepath.Join(base, filepath.FromSlash(rel)))
		if err != nil {
			return err
		}
		s.scripts = append(s.scripts, rel)
		for i, line := range strings.Split(string(raw), "\n") {
			if isCommentLine(line) {
				continue
			}
			for _, m := range assetExt.FindAllStringSubmatchIndex(line, -1) {
				path := strings.ReplaceAll(line[m[2]:m[3]], "\\", "/")
				path = strings.TrimPrefix(path, "./")
				if seen[path] {
					continue
				}
				seen[path] = true
				if filepath.IsAbs(filepath.FromSlash(path)) || strings.HasPrefix(path, "../") {
					// Outside the game folder: it would not ship, so the
					// runtime would not find it either.
					s.missing = append(s.missing, missingAsset{rel, i + 1, path})
					continue
				}
				st, err := os.Stat(filepath.Join(base, filepath.FromSlash(path)))
				if err != nil || st.IsDir() {
					if !outputCall.MatchString(line[:m[0]]) {
						s.missing = append(s.missing, missingAsset{rel, i + 1, path})
					}
					continue
				}
				ext := strings.ToLower(filepath.Ext(path))
				if ext == ".mb" || ext == ".bb" {
					if err := walk(path); err != nil {
						return err
					}
					continue
				}
				s.assets = append(s.assets, path)
			}
		}
		return nil
	}
	main = filepath.Base(main)
	seen[main] = true
	if err := walk(main); err != nil {
		return nil, err
	}
	return s, nil
}

// isCommentLine is true for a line that is only a ; ' or // comment.
func isCommentLine(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, ";") || strings.HasPrefix(t, "'") || strings.HasPrefix(t, "//")
}

// listDir returns every file under base/dir, relative to base with forward
// slashes, or nothing when the folder does not exist.
func listDir(base, dir string) ([]string, error) {
	root := filepath.Join(base, filepath.FromSlash(dir))
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		return nil, nil
	}
	var out []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(out)
	return out, err
}

// storedExt are formats that are already compressed; deflating them again
// only costs load time.
var storedExt = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".ogg": true, ".mp3": true,
	".flac": true, ".zip": true, ".pak": true, ".ktx2": true, ".glb": true,
}

// writePak zips files (relative to base) into path. The runtime opens it at
// startup when it is named <exe name>.pak or data.pak.
func writePak(path, base string, files []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(out)
	err = func() error {
		for _, rel := range files {
			in, err := os.Open(filepath.Join(base, filepath.FromSlash(rel)))
			if err != nil {
				return err
			}
			method := zip.Deflate
			if storedExt[strings.ToLower(filepath.Ext(rel))] {
				method = zip.Store
			}
			dst, err := zw.CreateHeader(&zip.FileHeader{Name: rel, Method: method})
			if err == nil {
				_, err = io.Copy(dst, in)
			}
			in.Close()
			if err != nil {
				return err
			}
		}
		return zw.Close()
	}()
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}
