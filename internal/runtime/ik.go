package runtime

import (
	"fmt"
	"math"
	"strings"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

// ikChain is a two-bone solver. Clip playback still owns the pose;
// this runs after the pose and aims the named joints at a world point.
type ikChain struct {
	upper, lower, end string
	x, y, z           float32
	weight            float32
	pole              math32.Vector3
	hasPole           bool
	bones             []string
	chain             bool
}

func (w *World) ikCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"setik": n(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			upper, lower, end := argS(a, 1), argS(a, 2), argS(a, 3)
			if findNamedNode(e.node, upper) == nil || findNamedNode(e.node, lower) == nil || findNamedNode(e.node, end) == nil {
				return value.Value{}, fmt.Errorf("SetIK: joint %q / %q / %q not found", upper, lower, end)
			}
			wt := float32(argN(a, 7, 1))
			if wt < 0 {
				wt = 0
			}
			if wt > 1 {
				wt = 1
			}
			e.ik = &ikChain{
				upper: upper, lower: lower, end: end,
				x: float32(argN(a, 4, 0)), y: float32(argN(a, 5, 0)), z: float32(argN(a, 6, 0)),
				weight: wt,
			}
			w.solveIK(e)
			return value.Num(float64(wt)), nil
		}),
		"setikpole": n(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.ik == nil {
				return value.Value{}, fmt.Errorf("SetIKPole: no IK chain")
			}
			e.ik.pole = math32.Vector3{X: float32(argN(a, 1, 0)), Y: float32(argN(a, 2, 1)), Z: float32(argN(a, 3, 0))}
			e.ik.hasPole = true
			w.solveIK(e)
			return z()
		}),
		"clearik": n(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			e.ik = nil
			return z()
		}),
		"setikchain": n(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			names := splitBones(argS(a, 1))
			if len(names) < 3 {
				return value.Value{}, fmt.Errorf("SetIKChain: need at least 3 joint names")
			}
			for _, name := range names {
				if findNamedNode(e.node, name) == nil {
					return value.Value{}, fmt.Errorf("SetIKChain: joint %q not found", name)
				}
			}
			wt := float32(argN(a, 5, 1))
			if wt < 0 {
				wt = 0
			}
			if wt > 1 {
				wt = 1
			}
			e.ik = &ikChain{
				bones: names, chain: true,
				x: float32(argN(a, 2, 0)), y: float32(argN(a, 3, 0)), z: float32(argN(a, 4, 0)),
				weight: wt,
			}
			w.solveIK(e)
			return value.Num(float64(len(names))), nil
		}),
	}
}

func (w *World) solveIK(e *Entity) {
	if e == nil || e.ik == nil || e.node == nil {
		return
	}
	if e.ik.chain {
		w.solveFABRIK(e)
		return
	}
	ch := e.ik
	if ch.weight <= 0 {
		return
	}
	upN := findNamedNode(e.node, ch.upper)
	loN := findNamedNode(e.node, ch.lower)
	enN := findNamedNode(e.node, ch.end)
	if upN == nil || loN == nil || enN == nil {
		return
	}
	w.refreshWorldMatrices()
	upper := upN.GetNode()
	lower := loN.GetNode()
	end := enN.GetNode()
	root := matPos(upper)
	mid := matPos(lower)
	tip := matPos(end)
	tx, ty, tz := toG3N(ch.x, ch.y, ch.z)
	target := math32.Vector3{X: tx, Y: ty, Z: tz}
	pole := math32.Vector3{}
	if ch.hasPole {
		px, py, pz := toG3N(ch.pole.X, ch.pole.Y, ch.pole.Z)
		pole = math32.Vector3{X: px, Y: py, Z: pz}
	}
	newMid, newTip := solveTwoBone(root, mid, tip, target, pole, ch.hasPole)
	oldUp := upper.Quaternion()
	oldLo := lower.Quaternion()
	aimBone(upper, root, mid, newMid)
	upper.UpdateMatrixWorld()
	midNow := matPos(lower)
	aimBone(lower, midNow, tip, newTip)
	if ch.weight < 0.999 {
		blendLocalQuat(upper, oldUp, ch.weight)
		blendLocalQuat(lower, oldLo, ch.weight)
	}
}

func solveTwoBone(root, mid, tip, target, pole math32.Vector3, hasPole bool) (math32.Vector3, math32.Vector3) {
	aLen := vecLen(vecSub(mid, root))
	bLen := vecLen(vecSub(tip, mid))
	if aLen < 1e-5 {
		aLen = 1e-5
	}
	if bLen < 1e-5 {
		bLen = 1e-5
	}
	diff := vecSub(target, root)
	dist := vecLen(diff)
	maxR := aLen + bLen - 1e-4
	minR := float32(math.Abs(float64(aLen-bLen))) + 1e-4
	if dist > maxR {
		dist = maxR
	}
	if dist < minR {
		dist = minR
	}
	dir := vecNorm(diff)
	if vecLen(dir) < 1e-6 {
		dir = math32.Vector3{Y: 1}
	}
	var bend math32.Vector3
	if hasPole {
		bend = vecCross(dir, vecNorm(vecSub(pole, root)))
	} else {
		bend = vecCross(vecSub(mid, root), vecSub(tip, mid))
	}
	if vecLen(bend) < 1e-5 {
		bend = vecCross(dir, math32.Vector3{Y: 1})
		if vecLen(bend) < 1e-5 {
			bend = vecCross(dir, math32.Vector3{X: 1})
		}
	}
	bend = vecNorm(bend)
	cosA := (dist*dist + aLen*aLen - bLen*bLen) / (2 * dist * aLen)
	if cosA > 1 {
		cosA = 1
	}
	if cosA < -1 {
		cosA = -1
	}
	sinA := float32(math.Sqrt(float64(1 - cosA*cosA)))
	perp := vecNorm(vecCross(bend, dir))
	newMid := vecAdd(root, vecAdd(vecScale(dir, cosA*aLen), vecScale(perp, sinA*aLen)))
	newTip := vecAdd(root, vecScale(dir, dist))
	return newMid, newTip
}

func aimBone(bone *core.Node, from, oldTo, newTo math32.Vector3) {
	a := vecNorm(vecSub(oldTo, from))
	b := vecNorm(vecSub(newTo, from))
	if vecLen(a) < 1e-6 || vecLen(b) < 1e-6 {
		return
	}
	var delta math32.Quaternion
	delta.SetFromUnitVectors(&a, &b)
	wq := matQuat(bone)
	var nw math32.Quaternion
	nw.MultiplyQuaternions(&delta, &wq)
	local := nw
	if p := bone.Parent(); p != nil {
		pq := matQuat(p.GetNode())
		pq.Inverse()
		local.MultiplyQuaternions(&pq, &nw)
	}
	bone.SetQuaternion(local.X, local.Y, local.Z, local.W)
}

func blendLocalQuat(n *core.Node, before math32.Quaternion, weight float32) {
	now := n.Quaternion()
	q := before
	q.Slerp(&now, weight)
	n.SetQuaternion(q.X, q.Y, q.Z, q.W)
}

func matPos(n *core.Node) math32.Vector3 {
	var p math32.Vector3
	m := n.MatrixWorld()
	p.SetFromMatrixPosition(&m)
	return p
}

func matQuat(n *core.Node) math32.Quaternion {
	var p, s math32.Vector3
	var q math32.Quaternion
	m := n.MatrixWorld()
	m.Decompose(&p, &q, &s)
	return q
}

func vecSub(a, b math32.Vector3) math32.Vector3 {
	return math32.Vector3{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z}
}
func vecAdd(a, b math32.Vector3) math32.Vector3 {
	return math32.Vector3{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z}
}
func vecScale(a math32.Vector3, s float32) math32.Vector3 {
	return math32.Vector3{X: a.X * s, Y: a.Y * s, Z: a.Z * s}
}
func vecLen(a math32.Vector3) float32 {
	return float32(math.Sqrt(float64(a.X*a.X + a.Y*a.Y + a.Z*a.Z)))
}
func vecNorm(a math32.Vector3) math32.Vector3 {
	l := vecLen(a)
	if l < 1e-8 {
		return math32.Vector3{}
	}
	return vecScale(a, 1/l)
}
func vecCross(a, b math32.Vector3) math32.Vector3 {
	return math32.Vector3{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

func splitBones(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (w *World) solveFABRIK(e *Entity) {
	ch := e.ik
	if ch == nil || ch.weight <= 0 || len(ch.bones) < 3 {
		return
	}
	w.refreshWorldMatrices()
	nodes := make([]*core.Node, len(ch.bones))
	pts := make([]math32.Vector3, len(ch.bones))
	for i, name := range ch.bones {
		n := findNamedNode(e.node, name)
		if n == nil {
			return
		}
		nodes[i] = n.GetNode()
		pts[i] = matPos(nodes[i])
	}
	lens := make([]float32, len(pts)-1)
	for i := range lens {
		lens[i] = vecLen(vecSub(pts[i+1], pts[i]))
		if lens[i] < 1e-4 {
			lens[i] = 1e-4
		}
	}
	tx, ty, tz := toG3N(ch.x, ch.y, ch.z)
	target := math32.Vector3{X: tx, Y: ty, Z: tz}
	reach := float32(0)
	for _, l := range lens {
		reach += l
	}
	root := pts[0]
	if vecLen(vecSub(target, root)) > reach {
		dir := vecNorm(vecSub(target, root))
		target = vecAdd(root, vecScale(dir, reach*0.999))
	}
	solved := fabrik(pts, lens, root, target)
	for i := 0; i < len(nodes)-1; i++ {
		oldQ := nodes[i].Quaternion()
		from := matPos(nodes[i])
		oldChild := matPos(nodes[i+1])
		aimBone(nodes[i], from, oldChild, solved[i+1])
		if ch.weight < 0.999 {
			blendLocalQuat(nodes[i], oldQ, ch.weight)
		}
		nodes[i].UpdateMatrixWorld()
	}
}

func fabrik(pts []math32.Vector3, lens []float32, root, target math32.Vector3) []math32.Vector3 {
	out := append([]math32.Vector3(nil), pts...)
	n := len(out)
	for iter := 0; iter < 8; iter++ {
		out[n-1] = target
		for i := n - 2; i >= 0; i-- {
			dir := vecNorm(vecSub(out[i], out[i+1]))
			if vecLen(dir) < 1e-6 {
				dir = math32.Vector3{Y: 1}
			}
			out[i] = vecAdd(out[i+1], vecScale(dir, lens[i]))
		}
		out[0] = root
		for i := 1; i < n; i++ {
			dir := vecNorm(vecSub(out[i], out[i-1]))
			if vecLen(dir) < 1e-6 {
				dir = math32.Vector3{Y: 1}
			}
			out[i] = vecAdd(out[i-1], vecScale(dir, lens[i-1]))
		}
	}
	return out
}
