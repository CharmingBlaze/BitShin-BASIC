package runtime

import (
	"fmt"
	"os"
	"strings"

	"github.com/g3n/engine/core"

	"moonbasic/internal/value"
)

func (L *animLayer) boneWeight(name string) float32 {
	if L == nil || len(L.mask) == 0 {
		return 1
	}
	if w, ok := L.mask[strings.ToLower(name)]; ok {
		return w
	}
	return 1
}

func (w *World) setBoneMask(a []value.Value) (value.Value, error) {
	e, err := w.ent(argI(a, 0, 0))
	if err != nil {
		return value.Value{}, err
	}
	if e.anim == nil {
		e.anim = &animState{}
	}
	st := e.anim
	slot := argI(a, 1, 0)
	if slot < 0 {
		return value.Value{}, fmt.Errorf("SetBoneMask: slot %d", slot)
	}
	for len(st.layers) <= slot {
		st.layers = append(st.layers, animLayer{seq: -1, speed: 1})
	}
	L := &st.layers[slot]
	if L.mask == nil {
		L.mask = map[string]float32{}
	}
	name := strings.ToLower(argS(a, 2))
	wt := float32(argN(a, 3, 0))
	if wt < 0 {
		wt = 0
	}
	if wt > 1 {
		wt = 1
	}
	if name == "" || name == "*" {
		L.mask = nil
		return value.Num(1), nil
	}
	L.mask[name] = wt
	return value.Num(float64(wt)), nil
}

func (w *World) setRootMotion(a []value.Value) (value.Value, error) {
	e, err := w.ent(argI(a, 0, 0))
	if err != nil {
		return value.Value{}, err
	}
	if e.anim == nil {
		e.anim = &animState{}
	}
	st := e.anim
	st.rootOn = argI(a, 1, 1) != 0
	st.rootBone = argS(a, 2)
	st.rootY = argI(a, 3, 0) != 0
	st.rootHave = false
	if st.rootOn {
		return value.Num(1), nil
	}
	return value.Num(0), nil
}

func (st *animState) applyRootMotion(e *Entity) {
	if st == nil || !st.rootOn || e == nil || e.node == nil {
		return
	}
	bone := findBone(e.node, st.rootBone)
	if bone == nil {
		return
	}
	p := bone.Position()
	if !st.rootHave {
		st.rootHave = true
		st.rootLast = p
		st.rootPlant = p
		return
	}
	dx := p.X - st.rootLast.X
	dy := p.Y - st.rootLast.Y
	dz := p.Z - st.rootLast.Z
	st.rootLast = p
	ent := e.node.GetNode()
	ep := ent.Position()
	if !st.rootY {
		dy = 0
	}
	ent.SetPosition(ep.X+dx, ep.Y+dy, ep.Z+dz)
	bone.SetPosition(st.rootPlant.X, st.rootPlant.Y, st.rootPlant.Z)
}

func findBone(root core.INode, name string) *core.Node {
	if root == nil {
		return nil
	}
	node := root.GetNode()
	if name == "" {
		kids := node.Children()
		if len(kids) == 0 {
			return nil
		}
		return kids[0].GetNode()
	}
	want := strings.ToLower(name)
	var found *core.Node
	var walk func(core.INode)
	walk = func(n core.INode) {
		if n == nil || found != nil {
			return
		}
		nd := n.GetNode()
		if strings.ToLower(nd.Name()) == want {
			found = nd
			return
		}
		for _, c := range nd.Children() {
			walk(c)
		}
	}
	walk(node)
	return found
}

func (w *World) exportAnimClip(a []value.Value) (value.Value, error) {
	e, err := w.ent(argI(a, 0, 0))
	if err != nil {
		return value.Value{}, err
	}
	path := argS(a, 1)
	if path == "" {
		return value.Value{}, fmt.Errorf("ExportAnimClip: path")
	}
	var clip fbxClip
	if e.anim != nil && len(e.anim.fbx) > 0 {
		seq := e.anim.fbxSeq
		if seq < 0 || seq >= len(e.anim.fbx) {
			seq = 0
		}
		clip = e.anim.fbx[seq]
	}
	b, err := clip.glTFDocument()
	if err != nil {
		return value.Value{}, err
	}
	if err := os.WriteFile(w.resolve(path), b, 0o644); err != nil {
		return value.Value{}, err
	}
	return value.Num(1), nil
}
