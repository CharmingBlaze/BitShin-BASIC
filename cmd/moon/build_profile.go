package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// buildProfile is one entry of build.profiles in moon.json. Unset fields
// come from Base, a built-in profile (dev when empty).
type buildProfile struct {
	Base     string `json:"base,omitempty"`
	Pak      *bool  `json:"pak,omitempty"`      // assets in <name>.pak instead of loose files
	Strip    *bool  `json:"strip,omitempty"`    // drop symbols and debug info
	Console  *bool  `json:"console,omitempty"`  // keep the console window on Windows
	Debug    *bool  `json:"debug,omitempty"`    // DebugLog output on, as MOON_DEBUG=1
	Headless *bool  `json:"headless,omitempty"` // no window or GL
	Strict   *bool  `json:"strict,omitempty"`   // a missing asset fails the build
	Tags     string `json:"tags,omitempty"`     // Go build tags, e.g. "enet"
}

// profileFlags is a profile with every field decided.
type profileFlags struct {
	pak, strip, console, debug, headless, strict bool
	tags                                         string
}

// builtinProfiles: dev is loose files with a console (edit assets and run
// again), debug adds DebugLog output, release packs assets, strips the
// binary, and on Windows opens no console window.
var builtinProfiles = map[string]profileFlags{
	"dev":     {console: true},
	"debug":   {console: true, debug: true},
	"release": {pak: true, strip: true},
}

// moonProject is the part of moon.json that moon build reads. The IDE owns
// the file; see ide/project.go for every field.
type moonProject struct {
	Name   string `json:"name"`
	Assets string `json:"assets"`
	Scenes string `json:"scenes"`
	Build  struct {
		OutputDir string                  `json:"outputDir"`
		Icon      string                  `json:"icon"`
		Version   string                  `json:"version"`
		Company   string                  `json:"company"`
		Profile   string                  `json:"profile"`
		Profiles  map[string]buildProfile `json:"profiles"`
	} `json:"build"`
}

// findMoonJSON looks for moon.json in the script's folder and up to three
// folders above it. It returns the project and its root, or nil.
func findMoonJSON(dir string) (*moonProject, string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, "", err
	}
	for i := 0; i < 4; i++ {
		path := filepath.Join(dir, "moon.json")
		if raw, err := os.ReadFile(path); err == nil {
			var p moonProject
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, "", fmt.Errorf("%s: %w", path, err)
			}
			return &p, dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil, "", nil
}

// resolveProfile returns a built-in profile with any moon.json overrides of
// the same name applied, or a moon.json profile on top of its base.
func resolveProfile(name string, custom map[string]buildProfile) (profileFlags, error) {
	return resolveProfileDepth(name, custom, 0)
}

func resolveProfileDepth(name string, custom map[string]buildProfile, depth int) (profileFlags, error) {
	if depth > 4 {
		return profileFlags{}, fmt.Errorf("profile %q: base chain too deep", name)
	}
	c, ok := custom[name]
	b, builtin := builtinProfiles[name]
	if !ok {
		if !builtin {
			return profileFlags{}, fmt.Errorf("unknown profile %q (dev, debug, release, or one in moon.json build.profiles)", name)
		}
		return b, nil
	}
	if !builtin {
		base := c.Base
		if base == "" {
			base = "dev"
		}
		if base == name {
			return profileFlags{}, fmt.Errorf("profile %q is its own base", name)
		}
		var err error
		if b, err = resolveProfileDepth(base, custom, depth+1); err != nil {
			return profileFlags{}, err
		}
	}
	set := func(dst *bool, v *bool) {
		if v != nil {
			*dst = *v
		}
	}
	set(&b.pak, c.Pak)
	set(&b.strip, c.Strip)
	set(&b.console, c.Console)
	set(&b.debug, c.Debug)
	set(&b.headless, c.Headless)
	set(&b.strict, c.Strict)
	if c.Tags != "" {
		b.tags = joinTags(b.tags, c.Tags)
	}
	return b, nil
}

// parseBuildArgs reads moon build's flags. Flags win over the profile, the
// profile over moon.json's build fields, and those over the defaults.
func parseBuildArgs(args []string) (buildOpts, error) {
	var o buildOpts
	if len(args) < 1 || strings.HasPrefix(args[0], "-") {
		return o, fmt.Errorf("moon build: missing program.mb")
	}
	o.src = args[0]
	o.targetOS = runtime.GOOS
	o.targetArch = runtime.GOARCH
	str := map[string]*string{}
	for _, k := range []string{"-o", "-os", "-arch", "-tags", "-profile", "-name", "-version", "-company", "-icon"} {
		str[k] = new(string)
	}
	var pak, headless, strict *bool
	yes, no := true, false
	for i := 1; i < len(args); i++ {
		a := args[i]
		if p, ok := str[a]; ok {
			if i+1 >= len(args) {
				return o, fmt.Errorf("moon build: %s needs a value", a)
			}
			i++
			*p = args[i]
			continue
		}
		switch a {
		case "-pak":
			pak = &yes
		case "-loose", "-nopak":
			pak = &no
		case "-headless", "-server":
			headless = &yes
		case "-strict":
			strict = &yes
		default:
			return o, fmt.Errorf("moon build: unknown flag %s", a)
		}
	}
	proj, root, err := findMoonJSON(filepath.Dir(o.src))
	if err != nil {
		return o, fmt.Errorf("moon build: %w", err)
	}
	if proj == nil {
		proj = &moonProject{}
	}
	fromRoot := func(rel string) string {
		if rel == "" || filepath.IsAbs(rel) || root == "" {
			return rel
		}
		return filepath.Join(root, filepath.FromSlash(rel))
	}
	o.profile = firstNonEmpty(*str["-profile"], proj.Build.Profile, "dev")
	if o.prof, err = resolveProfile(o.profile, proj.Build.Profiles); err != nil {
		return o, fmt.Errorf("moon build: %w", err)
	}
	if pak != nil {
		o.prof.pak = *pak
	}
	if headless != nil {
		o.prof.headless = *headless
	}
	if strict != nil {
		o.prof.strict = *strict
	}
	if *str["-os"] != "" {
		o.targetOS = normalizeOS(*str["-os"])
	}
	if *str["-arch"] != "" {
		o.targetArch = strings.ToLower(*str["-arch"])
	}
	o.tags = *str["-tags"]
	o.outDir = firstNonEmpty(*str["-o"], fromRoot(proj.Build.OutputDir), "dist")
	name := proj.Name
	if name == "Untitled" {
		name = ""
	}
	o.appName = firstNonEmpty(*str["-name"], name)
	o.exeName = firstNonEmpty(safeFileName(o.appName), "moon")
	if o.appName == "" {
		o.appName = strings.TrimSuffix(filepath.Base(o.src), filepath.Ext(o.src))
	}
	o.version = firstNonEmpty(*str["-version"], proj.Build.Version, "1.0.0")
	o.company = firstNonEmpty(*str["-company"], proj.Build.Company)
	o.icon = firstNonEmpty(*str["-icon"], fromRoot(proj.Build.Icon))
	o.dirs = uniqueStrings([]string{firstNonEmpty(proj.Assets, "assets"), firstNonEmpty(proj.Scenes, "scenes")})
	return o, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

// safeFileName drops the characters Windows does not allow in a file name.
func safeFileName(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`<>:"/\|?*`, r) {
			return -1
		}
		return r
	}, s)
	return strings.Trim(s, " .")
}
