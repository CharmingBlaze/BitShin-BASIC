package runtime

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"

	"moonbasic/internal/value"
)

func (w *World) exportCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), _ func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"exportgltf": n(func(a []value.Value) (value.Value, error) {
			file := argS(a, 0)
			e, err := w.ent(argI(a, 1, 0))
			if err != nil {
				return value.Value{}, err
			}
			if err := w.writeGLTF(file, e); err != nil {
				return value.Value{}, err
			}
			return value.Num(1), nil
		}),
		"exportmesh": n(func(a []value.Value) (value.Value, error) {
			file := argS(a, 0)
			e, err := w.ent(argI(a, 1, 0))
			if err != nil {
				return value.Value{}, err
			}
			if err := w.writeGLTF(file, e); err != nil {
				return value.Value{}, err
			}
			return value.Num(1), nil
		}),
	}
}

type gltfPart struct {
	name string
	pos  []float32
	idx  []uint32
}

func collectGLTFMeshes(n core.INode, out *[]gltfPart) {
	if n == nil {
		return
	}
	if m, ok := n.(*graphic.Mesh); ok && m != nil && m.GetGeometry() != nil {
		g := m.GetGeometry()
		posV := g.VBO(gls.VertexPosition)
		idx := g.Indices()
		if posV != nil && posV.Buffer() != nil && len(idx) >= 3 {
			stride := posV.Stride()
			if stride <= 0 {
				stride = 3
			}
			buf := *posV.Buffer()
			count := len(buf) / stride
			pos := packedPositions(buf, stride, count)
			name := m.GetNode().Name()
			*out = append(*out, gltfPart{
				name: name,
				pos:  append([]float32(nil), pos...),
				idx:  append([]uint32(nil), idx...),
			})
		}
	}
	for _, c := range n.GetNode().Children() {
		collectGLTFMeshes(c, out)
	}
}

func (w *World) writeGLTF(file string, e *Entity) error {
	if e == nil || e.node == nil {
		return fmt.Errorf("ExportGLTF: no mesh")
	}
	var parts []gltfPart
	collectGLTFMeshes(e.node, &parts)
	if len(parts) == 0 {
		return fmt.Errorf("ExportGLTF: entity is not a mesh")
	}
	var bin []byte
	putF := func(v float32) {
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], math.Float32bits(v))
		bin = append(bin, b[:]...)
	}
	putU := func(v uint32) {
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], v)
		bin = append(bin, b[:]...)
	}
	var nodes, meshes, accessors, views []any
	sceneNodes := make([]int, 0, len(parts))
	for i, part := range parts {
		posOff := len(bin)
		for _, v := range part.pos {
			putF(v)
		}
		idxOff := len(bin)
		for _, v := range part.idx {
			putU(v)
		}
		minX, minY, minZ := part.pos[0], part.pos[1], part.pos[2]
		maxX, maxY, maxZ := minX, minY, minZ
		for k := 0; k+2 < len(part.pos); k += 3 {
			if part.pos[k] < minX {
				minX = part.pos[k]
			}
			if part.pos[k+1] < minY {
				minY = part.pos[k+1]
			}
			if part.pos[k+2] < minZ {
				minZ = part.pos[k+2]
			}
			if part.pos[k] > maxX {
				maxX = part.pos[k]
			}
			if part.pos[k+1] > maxY {
				maxY = part.pos[k+1]
			}
			if part.pos[k+2] > maxZ {
				maxZ = part.pos[k+2]
			}
		}
		posView := len(views)
		views = append(views, map[string]any{
			"buffer": 0, "byteOffset": posOff, "byteLength": len(part.pos) * 4, "target": 34962,
		})
		idxView := len(views)
		views = append(views, map[string]any{
			"buffer": 0, "byteOffset": idxOff, "byteLength": len(part.idx) * 4, "target": 34963,
		})
		posAcc := len(accessors)
		accessors = append(accessors, map[string]any{
			"bufferView": posView, "componentType": 5126, "count": len(part.pos) / 3, "type": "VEC3",
			"min": []float32{minX, minY, minZ}, "max": []float32{maxX, maxY, maxZ},
		})
		idxAcc := len(accessors)
		accessors = append(accessors, map[string]any{
			"bufferView": idxView, "componentType": 5125, "count": len(part.idx), "type": "SCALAR",
		})
		meshIndex := len(meshes)
		name := part.name
		if name == "" {
			name = e.name
		}
		meshes = append(meshes, map[string]any{
			"name": name,
			"primitives": []any{
				map[string]any{
					"attributes": map[string]int{"POSITION": posAcc},
					"indices":    idxAcc,
					"mode":       4,
				},
			},
		})
		sceneNodes = append(sceneNodes, i)
		nodes = append(nodes, map[string]any{"mesh": meshIndex, "name": name})
	}
	doc := map[string]any{
		"asset":       map[string]any{"version": "2.0", "generator": "moonBASIC"},
		"scene":       0,
		"scenes":      []any{map[string]any{"nodes": sceneNodes}},
		"nodes":       nodes,
		"meshes":      meshes,
		"accessors":   accessors,
		"bufferViews": views,
		"buffers": []any{
			map[string]any{
				"byteLength": len(bin),
				"uri":        "data:application/octet-stream;base64," + base64.StdEncoding.EncodeToString(bin),
			},
		},
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(w.resolve(file), raw, 0o644)
}
