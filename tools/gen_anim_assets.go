//go:build ignore

// Generates examples/assets/walker.gltf: an original CC0 box figure with
// idle, walk, run, strafe_left, strafe_right and wave clips for the blend
// tree example. Every clip keys the same nodes so blends stay clean.
//
//	go run tools/gen_anim_assets.go
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
)

type obj = map[string]any

var (
	buf       bytes.Buffer
	views     []obj
	accessors []obj
)

func pad() {
	for buf.Len()%4 != 0 {
		buf.WriteByte(0)
	}
}

func addFloats(vals []float32, typ string, comps int, minmax bool, target int) int {
	pad()
	off := buf.Len()
	for _, v := range vals {
		binary.Write(&buf, binary.LittleEndian, v)
	}
	view := obj{"buffer": 0, "byteOffset": off, "byteLength": len(vals) * 4}
	if target != 0 {
		view["target"] = target
	}
	views = append(views, view)
	acc := obj{"bufferView": len(views) - 1, "componentType": 5126, "count": len(vals) / comps, "type": typ}
	if minmax {
		mn := make([]float32, comps)
		mx := make([]float32, comps)
		for c := 0; c < comps; c++ {
			mn[c], mx[c] = float32(math.Inf(1)), float32(math.Inf(-1))
		}
		for i, v := range vals {
			c := i % comps
			if v < mn[c] {
				mn[c] = v
			}
			if v > mx[c] {
				mx[c] = v
			}
		}
		acc["min"], acc["max"] = mn, mx
	}
	accessors = append(accessors, acc)
	return len(accessors) - 1
}

func addIndices(idx []uint16) int {
	pad()
	off := buf.Len()
	for _, v := range idx {
		binary.Write(&buf, binary.LittleEndian, v)
	}
	views = append(views, obj{"buffer": 0, "byteOffset": off, "byteLength": len(idx) * 2, "target": 34963})
	accessors = append(accessors, obj{"bufferView": len(views) - 1, "componentType": 5123, "count": len(idx), "type": "SCALAR"})
	return len(accessors) - 1
}

func cube() (pos, nrm []float32, idx []uint16) {
	faces := [][3]float32{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}}
	for _, n := range faces {
		var u, v [3]float32
		switch {
		case n[0] != 0:
			u, v = [3]float32{0, 0, -n[0]}, [3]float32{0, 1, 0}
		case n[1] != 0:
			u, v = [3]float32{1, 0, 0}, [3]float32{0, 0, -n[1]}
		default:
			u, v = [3]float32{n[2], 0, 0}, [3]float32{0, 1, 0}
		}
		base := uint16(len(pos) / 3)
		for _, s := range [][2]float32{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
			for k := 0; k < 3; k++ {
				pos = append(pos, 0.5*(n[k]+s[0]*u[k]+s[1]*v[k]))
				nrm = append(nrm, n[k])
			}
		}
		idx = append(idx, base, base+1, base+2, base, base+2, base+3)
	}
	return
}

func qx(deg float64) []float32 {
	a := deg * math.Pi / 360
	return []float32{float32(math.Sin(a)), 0, 0, float32(math.Cos(a))}
}

func qz(deg float64) []float32 {
	a := deg * math.Pi / 360
	return []float32{0, 0, float32(math.Sin(a)), float32(math.Cos(a))}
}

// Node indices.
const (
	nRoot = iota
	nHips
	nTorso
	nTorsoMesh
	nHead
	nArmL
	nArmLMesh
	nArmR
	nArmRMesh
	nLegL
	nLegLMesh
	nLegR
	nLegRMesh
)

type pose struct {
	hipY, hipX                   float64 // translation offsets
	lean, armL, armR, legL, legR float64 // degrees about X (legs, arms)
	armLz, armRz, legLz, legRz   float64 // degrees about Z
}

func clip(name string, length float64, keys int, f func(ph float64) pose) obj {
	var times []float32
	ps := make([]pose, 0, keys+1)
	for i := 0; i <= keys; i++ {
		ph := float64(i) / float64(keys)
		times = append(times, float32(ph*length))
		ps = append(ps, f(ph))
	}
	tAcc := addFloats(times, "SCALAR", 1, true, 0)
	var samplers, channels []any
	add := func(node int, path string, vals []float32, typ string, comps int) {
		out := addFloats(vals, typ, comps, false, 0)
		samplers = append(samplers, obj{"input": tAcc, "output": out, "interpolation": "LINEAR"})
		channels = append(channels, obj{"sampler": len(samplers) - 1, "target": obj{"node": node, "path": path}})
	}
	var hips []float32
	for _, p := range ps {
		hips = append(hips, float32(p.hipX), float32(0.95+p.hipY), 0)
	}
	add(nHips, "translation", hips, "VEC3", 3)
	rot := func(node int, get func(p pose) (x, z float64)) {
		var v []float32
		for _, p := range ps {
			x, z := get(p)
			a, b := qx(x), qz(z)
			// q = qx * qz
			q := []float32{
				a[3]*b[0] + a[0]*b[3] + a[1]*b[2] - a[2]*b[1],
				a[3]*b[1] - a[0]*b[2] + a[1]*b[3] + a[2]*b[0],
				a[3]*b[2] + a[0]*b[1] - a[1]*b[0] + a[2]*b[3],
				a[3]*b[3] - a[0]*b[0] - a[1]*b[1] - a[2]*b[2],
			}
			v = append(v, q...)
		}
		add(node, "rotation", v, "VEC4", 4)
	}
	rot(nTorso, func(p pose) (float64, float64) { return p.lean, 0 })
	rot(nArmL, func(p pose) (float64, float64) { return p.armL, p.armLz })
	rot(nArmR, func(p pose) (float64, float64) { return p.armR, p.armRz })
	rot(nLegL, func(p pose) (float64, float64) { return p.legL, p.legLz })
	rot(nLegR, func(p pose) (float64, float64) { return p.legR, p.legRz })
	return obj{"name": name, "samplers": samplers, "channels": channels}
}

func main() {
	pos, nrm, idx := cube()
	aPos := addFloats(pos, "VEC3", 3, true, 34962)
	aNrm := addFloats(nrm, "VEC3", 3, false, 34962)
	aIdx := addIndices(idx)
	prim := func(mat int) obj {
		return obj{"primitives": []any{obj{"attributes": obj{"POSITION": aPos, "NORMAL": aNrm}, "indices": aIdx, "material": mat}}}
	}
	mats := []any{
		obj{"name": "Shirt", "pbrMetallicRoughness": obj{"baseColorFactor": []float32{0.20, 0.55, 0.95, 1}, "metallicFactor": 0, "roughnessFactor": 0.7}},
		obj{"name": "Skin", "pbrMetallicRoughness": obj{"baseColorFactor": []float32{0.95, 0.75, 0.55, 1}, "metallicFactor": 0, "roughnessFactor": 0.8}},
		obj{"name": "Pants", "pbrMetallicRoughness": obj{"baseColorFactor": []float32{0.18, 0.20, 0.26, 1}, "metallicFactor": 0, "roughnessFactor": 0.8}},
	}
	meshes := []any{prim(0), prim(1), prim(2)}
	nodes := []any{
		obj{"name": "Walker", "children": []int{nHips}},
		obj{"name": "Hips", "translation": []float32{0, 0.95, 0}, "children": []int{nTorso, nLegL, nLegR}},
		obj{"name": "Torso", "translation": []float32{0, 0.05, 0}, "children": []int{nTorsoMesh, nHead, nArmL, nArmR}},
		obj{"name": "TorsoMesh", "mesh": 0, "translation": []float32{0, 0.3, 0}, "scale": []float32{0.5, 0.6, 0.28}},
		obj{"name": "Head", "mesh": 1, "translation": []float32{0, 0.8, 0}, "scale": []float32{0.3, 0.3, 0.3}},
		obj{"name": "ArmL", "translation": []float32{-0.34, 0.56, 0}, "children": []int{nArmLMesh}},
		obj{"name": "ArmLMesh", "mesh": 1, "translation": []float32{0, -0.28, 0}, "scale": []float32{0.14, 0.56, 0.14}},
		obj{"name": "ArmR", "translation": []float32{0.34, 0.56, 0}, "children": []int{nArmRMesh}},
		obj{"name": "ArmRMesh", "mesh": 1, "translation": []float32{0, -0.28, 0}, "scale": []float32{0.14, 0.56, 0.14}},
		obj{"name": "LegL", "translation": []float32{-0.13, 0, 0}, "children": []int{nLegLMesh}},
		obj{"name": "LegLMesh", "mesh": 2, "translation": []float32{0, -0.45, 0}, "scale": []float32{0.18, 0.9, 0.18}},
		obj{"name": "LegR", "translation": []float32{0.13, 0, 0}, "children": []int{nLegRMesh}},
		obj{"name": "LegRMesh", "mesh": 2, "translation": []float32{0, -0.45, 0}, "scale": []float32{0.18, 0.9, 0.18}},
	}
	s := func(ph float64) float64 { return math.Sin(2 * math.Pi * ph) }
	c := func(ph float64) float64 { return math.Cos(4 * math.Pi * ph) }
	anims := []any{
		clip("idle", 2.0, 8, func(ph float64) pose {
			return pose{hipY: -0.01 + 0.01*math.Cos(2*math.Pi*ph), armL: 3 * s(ph), armR: -3 * s(ph), armLz: -4, armRz: 4}
		}),
		clip("walk", 1.0, 16, func(ph float64) pose {
			return pose{hipY: 0.03 * c(ph), lean: 4, legL: 28 * s(ph), legR: -28 * s(ph), armL: -24 * s(ph), armR: 24 * s(ph), armLz: -3, armRz: 3}
		}),
		clip("run", 0.6, 16, func(ph float64) pose {
			return pose{hipY: 0.07*c(ph) - 0.04, lean: 14, legL: 55 * s(ph), legR: -55 * s(ph), armL: -60*s(ph) - 20, armR: 60*s(ph) - 20, armLz: -6, armRz: 6}
		}),
		clip("strafe_left", 1.0, 16, func(ph float64) pose {
			return pose{hipY: 0.03 * c(ph), hipX: 0, legLz: -18 - 14*s(ph), legRz: 4 - 14*s(ph), armLz: -12, armRz: 6}
		}),
		clip("strafe_right", 1.0, 16, func(ph float64) pose {
			return pose{hipY: 0.03 * c(ph), legLz: -4 + 14*s(ph), legRz: 18 + 14*s(ph), armLz: -6, armRz: 12}
		}),
		clip("wave", 1.2, 12, func(ph float64) pose {
			return pose{armR: 0, armRz: 150 + 25*s(2*ph), armLz: -4, hipY: -0.01}
		}),
	}
	doc := obj{
		"asset":       obj{"version": "2.0", "generator": "moonBASIC tools/gen_anim_assets.go (CC0)"},
		"scene":       0,
		"scenes":      []any{obj{"nodes": []int{nRoot}}},
		"nodes":       nodes,
		"meshes":      meshes,
		"materials":   mats,
		"animations":  anims,
		"accessors":   accessors,
		"bufferViews": views,
	}
	pad()
	doc["buffers"] = []any{obj{"byteLength": buf.Len(), "uri": "data:application/octet-stream;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())}}
	out, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	path := filepath.Join("examples", "assets", "walker.gltf")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		panic(err)
	}
}
