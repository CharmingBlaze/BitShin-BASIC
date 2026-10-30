package runtime

import (
	"fmt"
	"strings"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/math32"
)

// fbxCluster is one FBX skin cluster: control-point indexes and weights.
type fbxCluster struct {
	name    string
	indexes []int32
	weights []float64
	x, y, z float64
}

// fbxLimb is a LimbNode local translation paired with a cluster by order.
type fbxLimb struct {
	name    string
	x, y, z float64
}

// meshSkin deforms a mesh from joint translation deltas. Up to four
// influences per vertex. Animation clips from the FBX file are not read.
type meshSkin struct {
	joints []skinJoint
	bind   []float32
	index  []int16
	weight []float32
}

type skinJoint struct {
	name                string
	node                *core.Node
	bindX, bindY, bindZ float32
}

func parseFBXSkinASCII(src string) ([]fbxCluster, []fbxLimb) {
	indexes := allASCIIArrays(src, "Indexes")
	weights := allASCIIArrays(src, "Weights")
	n := len(indexes)
	if len(weights) < n {
		n = len(weights)
	}
	limbs := parseFBXLimbs(src)
	var clusters []fbxCluster
	for i := 0; i < n; i++ {
		if len(indexes[i]) == 0 || len(weights[i]) == 0 {
			continue
		}
		c := fbxCluster{name: fmt.Sprintf("joint%d", i)}
		m := len(indexes[i])
		if len(weights[i]) < m {
			m = len(weights[i])
		}
		for k := 0; k < m; k++ {
			c.indexes = append(c.indexes, int32(indexes[i][k]))
			c.weights = append(c.weights, weights[i][k])
		}
		if i < len(limbs) {
			c.name = limbs[i].name
			c.x, c.y, c.z = limbs[i].x, limbs[i].y, limbs[i].z
		}
		clusters = append(clusters, c)
	}
	return clusters, limbs
}

func allASCIIArrays(src, key string) [][]float64 {
	var out [][]float64
	i := 0
	for {
		j := strings.Index(src[i:], key+":")
		if j < 0 {
			return out
		}
		j += i + len(key) + 1
		rest := src[j:]
		a := strings.Index(rest, "a:")
		if a < 0 || a > 80 {
			i = j
			continue
		}
		body := rest[a+2:]
		end := strings.Index(body, "}")
		if end < 0 {
			end = len(body)
		}
		nums := splitNumbers(body[:end])
		if len(nums) > 0 {
			out = append(out, nums)
		}
		i = j + a + 2
	}
}

func parseFBXLimbs(src string) []fbxLimb {
	var out []fbxLimb
	i := 0
	for {
		j := strings.Index(src[i:], "LimbNode")
		if j < 0 {
			return out
		}
		j += i
		head := src[i:j]
		name := fmt.Sprintf("joint%d", len(out))
		if k := strings.LastIndex(head, "Model::"); k >= 0 {
			rest := head[k+len("Model::"):]
			if q := strings.IndexAny(rest, "\","); q > 0 {
				name = rest[:q]
			}
		}
		window := src[j:]
		if len(window) > 800 {
			window = window[:800]
		}
		x, y, z := 0.0, 0.0, 0.0
		if p := strings.Index(window, "Lcl Translation"); p >= 0 {
			line := window[p:]
			if n := strings.IndexByte(line, '\n'); n >= 0 {
				line = line[:n]
			}
			nums := splitNumbers(line)
			if len(nums) >= 3 {
				x, y, z = nums[len(nums)-3], nums[len(nums)-2], nums[len(nums)-1]
			}
		}
		out = append(out, fbxLimb{name: name, x: x, y: y, z: z})
		i = j + len("LimbNode")
	}
}

func (w *World) attachFBXSkin(e *Entity, mesh fbxMesh) {
	if e == nil || e.mesh == nil || len(mesh.clusters) == 0 {
		return
	}
	g := e.mesh.GetGeometry()
	posV := g.VBO(gls.VertexPosition)
	if posV == nil || posV.Buffer() == nil {
		return
	}
	buf := *posV.Buffer()
	stride := posV.Stride()
	if stride <= 0 {
		stride = 3
	}
	nvert := len(buf) / stride
	if nvert <= 0 {
		return
	}
	sk := &meshSkin{
		bind:   make([]float32, nvert*3),
		index:  make([]int16, nvert*4),
		weight: make([]float32, nvert*4),
	}
	for i := range sk.index {
		sk.index[i] = -1
	}
	for i := 0; i < nvert; i++ {
		base := i * stride
		sk.bind[i*3] = buf[base]
		sk.bind[i*3+1] = buf[base+1]
		sk.bind[i*3+2] = buf[base+2]
	}
	scale := 1.0
	if mesh.unit >= 50 {
		scale = mesh.unit
	}
	for ji, c := range mesh.clusters {
		if ji > 32767 {
			break
		}
		node := core.NewNode()
		node.SetName(c.name)
		x, y, z := c.x/scale, c.y/scale, c.z/scale
		if mesh.upAxis == 2 {
			y, z = z, -y
		}
		gx, gy, gz := toG3N(float32(x), float32(y), float32(z))
		node.SetPosition(gx, gy, gz)
		e.node.GetNode().Add(node)
		sk.joints = append(sk.joints, skinJoint{
			name: c.name, node: node,
			bindX: float32(x), bindY: float32(y), bindZ: float32(z),
		})
		for k, cp := range c.indexes {
			vi := int(cp)
			if vi < 0 || vi >= nvert || k >= len(c.weights) {
				continue
			}
			slot := -1
			for s := 0; s < 4; s++ {
				if sk.index[vi*4+s] < 0 {
					slot = s
					break
				}
			}
			if slot < 0 {
				continue
			}
			sk.index[vi*4+slot] = int16(ji)
			sk.weight[vi*4+slot] = float32(c.weights[k])
		}
	}
	if len(sk.joints) == 0 {
		return
	}
	e.skin = sk
	e.kind = "skin"
}

func (w *World) applySkins() {
	for _, e := range w.ents {
		if e == nil || e.skin == nil || e.mesh == nil {
			continue
		}
		sk := e.skin
		g := e.mesh.GetGeometry()
		if g == nil {
			continue
		}
		posV := g.VBO(gls.VertexPosition)
		if posV == nil || posV.Buffer() == nil {
			continue
		}
		stride := posV.Stride()
		if stride < 3 {
			continue
		}
		buf := *posV.Buffer()
		nvert := len(sk.bind) / 3
		if len(buf)/stride < nvert {
			continue
		}
		next := append([]float32(nil), buf...)
		for v := 0; v < nvert; v++ {
			x, y, z := sk.bind[v*3], sk.bind[v*3+1], sk.bind[v*3+2]
			for s := 0; s < 4; s++ {
				ji := sk.index[v*4+s]
				wgt := sk.weight[v*4+s]
				if ji < 0 || wgt == 0 || int(ji) >= len(sk.joints) {
					continue
				}
				jn := sk.joints[ji]
				if jn.node == nil {
					continue
				}
				p := jn.node.Position()
				cx, cy, cz := fromG3N(p.X, p.Y, p.Z)
				x += wgt * (cx - jn.bindX)
				y += wgt * (cy - jn.bindY)
				z += wgt * (cz - jn.bindZ)
			}
			base := v * stride
			next[base] = x
			next[base+1] = y
			next[base+2] = z
		}
		posV.SetBuffer(math32.ArrayF32(next))
	}
}
