package runtime

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
)

// fbxClip is translation curves read from an FBX animation stack.
// One FBX second is 46186158000 ticks.
type fbxClip struct {
	name   string
	curves []fbxCurve
}

type fbxCurve struct {
	bone  string
	axis  string // x y z
	times []float32
	vals  []float32
}

const fbxTicksPerSecond = 46186158000.0

func (c fbxClip) duration() float32 {
	var d float32
	for _, cv := range c.curves {
		if n := len(cv.times); n > 0 && cv.times[n-1] > d {
			d = cv.times[n-1]
		}
	}
	return d
}

func (st *animState) fbxDuration() float32 {
	if st == nil || st.fbxSeq < 0 || st.fbxSeq >= len(st.fbx) {
		return 0
	}
	return st.fbx[st.fbxSeq].duration()
}

func (cv fbxCurve) sample(t float32) float32 {
	if len(cv.vals) == 0 {
		return 0
	}
	if len(cv.times) == 0 || t <= cv.times[0] {
		return cv.vals[0]
	}
	last := len(cv.times) - 1
	if t >= cv.times[last] {
		return cv.vals[last]
	}
	for i := 1; i <= last; i++ {
		if t > cv.times[i] {
			continue
		}
		span := cv.times[i] - cv.times[i-1]
		u := float32(0)
		if span > 0 {
			u = (t - cv.times[i-1]) / span
		}
		return cv.vals[i-1] + (cv.vals[i]-cv.vals[i-1])*u
	}
	return cv.vals[last]
}

func (st *animState) applyFBX(e *Entity) {
	if st == nil || e == nil || e.node == nil || len(st.fbx) == 0 {
		return
	}
	seq := st.fbxSeq
	if seq < 0 || seq >= len(st.fbx) {
		seq = 0
	}
	clip := st.fbx[seq]
	for _, cv := range clip.curves {
		bone := findBone(e.node, cv.bone)
		if bone == nil {
			continue
		}
		p := bone.Position()
		v := cv.sample(st.time)
		switch cv.axis {
		case "y":
			p.Y = v
		case "z":
			p.Z = v
		default:
			p.X = v
		}
		bone.SetPosition(p.X, p.Y, p.Z)
	}
}

func putF32(b *bytes.Buffer, f float32) {
	var tmp [4]byte
	binary.LittleEndian.PutUint32(tmp[:], math.Float32bits(f))
	b.Write(tmp[:])
}

// glTFDocument is a glTF 2.0 JSON file with one translation channel per bone.
func (c fbxClip) glTFDocument() ([]byte, error) {
	order := []string{}
	by := map[string][]fbxCurve{}
	for _, cv := range c.curves {
		name := cv.bone
		if name == "" {
			name = "Armature"
		}
		if _, ok := by[name]; !ok {
			order = append(order, name)
		}
		by[name] = append(by[name], cv)
	}
	if len(order) == 0 {
		order = []string{"Armature"}
	}
	var buf bytes.Buffer
	type view struct {
		offset, length, count int
		vec3                  bool
	}
	var views []view
	addScalar := func(times []float32) int {
		v := view{offset: buf.Len(), count: len(times)}
		for _, t := range times {
			putF32(&buf, t)
		}
		v.length = buf.Len() - v.offset
		views = append(views, v)
		return len(views) - 1
	}
	addVec3 := func(xs [][3]float32) int {
		v := view{offset: buf.Len(), count: len(xs), vec3: true}
		for _, p := range xs {
			putF32(&buf, p[0])
			putF32(&buf, p[1])
			putF32(&buf, p[2])
		}
		v.length = buf.Len() - v.offset
		views = append(views, v)
		return len(views) - 1
	}
	nodes := make([]any, len(order))
	var samplers, channels []any
	for i, bone := range order {
		nodes[i] = map[string]any{"name": bone}
		curves := by[bone]
		if len(curves) == 0 {
			continue
		}
		seen := map[float32]bool{}
		var times []float32
		for _, cv := range curves {
			for _, t := range cv.times {
				if !seen[t] {
					seen[t] = true
					times = append(times, t)
				}
			}
		}
		sort.Slice(times, func(a, b int) bool { return times[a] < times[b] })
		if len(times) == 0 {
			times = []float32{0}
		}
		pts := make([][3]float32, len(times))
		for k, t := range times {
			var p [3]float32
			for _, cv := range curves {
				v := cv.sample(t)
				switch cv.axis {
				case "y":
					p[1] = v
				case "z":
					p[2] = v
				default:
					p[0] = v
				}
			}
			pts[k] = p
		}
		in := addScalar(times)
		out := addVec3(pts)
		samplers = append(samplers, map[string]any{"input": in, "output": out, "interpolation": "LINEAR"})
		channels = append(channels, map[string]any{
			"sampler": len(samplers) - 1,
			"target":  map[string]any{"node": i, "path": "translation"},
		})
	}
	bviews := make([]any, len(views))
	accs := make([]any, len(views))
	for i, v := range views {
		bviews[i] = map[string]any{"buffer": 0, "byteOffset": v.offset, "byteLength": v.length}
		acc := map[string]any{
			"bufferView": i, "componentType": 5126, "count": v.count, "type": "SCALAR",
		}
		if v.vec3 {
			acc["type"] = "VEC3"
		}
		accs[i] = acc
	}
	name := c.name
	if name == "" {
		name = "clip"
	}
	doc := map[string]any{
		"asset":    map[string]any{"version": "2.0", "generator": "moonBASIC"},
		"scene":    0,
		"scenes":   []any{map[string]any{"nodes": []int{0}}},
		"nodes":    nodes,
		"buffers":  []any{},
		"bufferViews": bviews,
		"accessors":   accs,
	}
	if buf.Len() > 0 {
		doc["buffers"] = []any{map[string]any{
			"byteLength": buf.Len(),
			"uri":        "data:application/octet-stream;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()),
		}}
		doc["animations"] = []any{map[string]any{"name": name, "samplers": samplers, "channels": channels}}
	}
	return json.MarshalIndent(doc, "", "  ")
}

// parseFBXClips reads AnimationCurve blocks and OP connections to limb nodes.
func parseFBXClips(src string) []fbxClip {
	models := map[string]string{}
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Model:") {
			continue
		}
		id, name, ok := fbxHeaderID(line)
		if !ok || !strings.Contains(line, "LimbNode") {
			continue
		}
		models[id] = name
	}
	var curves []fbxCurve
	ids := []string{}
	rest := src
	for {
		i := strings.Index(rest, "AnimationCurve:")
		if i < 0 {
			break
		}
		rest = rest[i:]
		lineEnd := strings.Index(rest, "\n")
		head := rest
		if lineEnd > 0 {
			head = rest[:lineEnd]
		}
		id, _, _ := fbxHeaderID(head)
		bodyEnd := strings.Index(rest, "\n}")
		body := rest
		if bodyEnd > 0 {
			body = rest[:bodyEnd]
			rest = rest[bodyEnd+2:]
		} else {
			rest = ""
		}
		times := fbxKeyList(body, "KeyTime")
		vals := fbxKeyList(body, "KeyValueFloat")
		n := len(times)
		if len(vals) < n {
			n = len(vals)
		}
		if n == 0 {
			continue
		}
		cv := fbxCurve{axis: "x", times: make([]float32, n), vals: make([]float32, n)}
		for k := 0; k < n; k++ {
			cv.times[k] = float32(times[k] / fbxTicksPerSecond)
			cv.vals[k] = float32(vals[k])
		}
		curves = append(curves, cv)
		ids = append(ids, id)
	}
	if len(curves) == 0 {
		return nil
	}
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "C:") {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}
		from := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		to := strings.Trim(strings.TrimSpace(parts[2]), "\"")
		prop := strings.Trim(strings.TrimSpace(parts[3]), "\"")
		axis := ""
		switch {
		case strings.Contains(prop, "X"):
			axis = "x"
		case strings.Contains(prop, "Y"):
			axis = "y"
		case strings.Contains(prop, "Z"):
			axis = "z"
		default:
			continue
		}
		bone := models[to]
		for i, id := range ids {
			if id == from {
				curves[i].bone = bone
				curves[i].axis = axis
			}
		}
	}
	limbs := parseFBXLimbs(src)
	li := 0
	for i := range curves {
		if curves[i].bone == "" && li < len(limbs) {
			curves[i].bone = limbs[li].name
			li++
		}
	}
	return []fbxClip{{name: "fbx", curves: curves}}
}

func fbxHeaderID(line string) (id, name string, ok bool) {
	i := strings.Index(line, ":")
	if i < 0 {
		return "", "", false
	}
	rest := strings.TrimSpace(line[i+1:])
	fields := strings.Split(rest, ",")
	if len(fields) == 0 {
		return "", "", false
	}
	id = strings.Trim(strings.TrimSpace(fields[0]), "\"")
	if len(fields) > 1 {
		name = strings.Trim(strings.TrimSpace(fields[1]), "\"")
		if j := strings.LastIndex(name, "::"); j >= 0 {
			name = name[j+2:]
		}
	}
	return id, name, id != ""
}

func fbxKeyList(body, key string) []float64 {
	i := strings.Index(body, key+":")
	if i < 0 {
		return nil
	}
	chunk := body[i:]
	a := strings.Index(chunk, "a:")
	if a < 0 {
		return nil
	}
	chunk = chunk[a+2:]
	end := strings.Index(chunk, "}")
	if end >= 0 {
		chunk = chunk[:end]
	}
	var out []float64
	for _, p := range strings.Split(chunk, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		f, err := strconv.ParseFloat(p, 64)
		if err != nil || math.IsNaN(f) {
			continue
		}
		out = append(out, f)
	}
	return out
}
