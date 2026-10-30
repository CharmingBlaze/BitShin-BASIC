package runtime

import (
	"fmt"

	"moonbasic/internal/value"
)

const (
	btSeq = iota + 1
	btSel
	btPar
	btCond
	btAct
)

const (
	btFail = 0
	btRun  = 1
	btOK   = 2
)

type btNode struct {
	kind   int
	kids   []int
	fn     string
	cursor int
}

type behavior struct {
	nodes []btNode
}

type btLink struct {
	tree int
	root int
}

func (w *World) behaviorCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"createbehavior": n(func(a []value.Value) (value.Value, error) {
			if w.bts == nil {
				w.bts = map[int]*behavior{}
			}
			id := w.takeHandle(&w.freeBT, &w.nextBT)
			w.bts[id] = &behavior{}
			return value.Num(float64(id)), nil
		}),
		"btsequence": n(func(a []value.Value) (value.Value, error) {
			return w.addBTNode(argI(a, 0, 0), btSeq, "")
		}),
		"btselector": n(func(a []value.Value) (value.Value, error) {
			return w.addBTNode(argI(a, 0, 0), btSel, "")
		}),
		"btparallel": n(func(a []value.Value) (value.Value, error) {
			return w.addBTNode(argI(a, 0, 0), btPar, "")
		}),
		"btcondition": n(func(a []value.Value) (value.Value, error) {
			return w.addBTNode(argI(a, 0, 0), btCond, argS(a, 1))
		}),
		"btaction": n(func(a []value.Value) (value.Value, error) {
			return w.addBTNode(argI(a, 0, 0), btAct, argS(a, 1))
		}),
		"btadd": n(func(a []value.Value) (value.Value, error) {
			bt := w.bts[argI(a, 0, 0)]
			if bt == nil {
				return value.Value{}, fmt.Errorf("BTAdd: no behavior")
			}
			parent := argI(a, 1, 0)
			child := argI(a, 2, 0)
			if parent < 0 || child < 0 || parent >= len(bt.nodes) || child >= len(bt.nodes) {
				return value.Value{}, fmt.Errorf("BTAdd: node out of range")
			}
			bt.nodes[parent].kids = append(bt.nodes[parent].kids, child)
			return value.Num(float64(child)), nil
		}),
		"setbehavior": n(func(a []value.Value) (value.Value, error) {
			ent := argI(a, 0, 0)
			if _, err := w.ent(ent); err != nil {
				return value.Value{}, err
			}
			tree := argI(a, 1, 0)
			bt := w.bts[tree]
			if bt == nil || len(bt.nodes) == 0 {
				return value.Value{}, fmt.Errorf("SetBehavior: empty tree")
			}
			root := argI(a, 2, 0)
			if root < 0 || root >= len(bt.nodes) {
				return value.Value{}, fmt.Errorf("SetBehavior: root %d missing", root)
			}
			if w.btEnt == nil {
				w.btEnt = map[int]btLink{}
			}
			w.btEnt[ent] = btLink{tree: tree, root: root}
			return value.Num(float64(ent)), nil
		}),
		"clearbehavior": n(func(a []value.Value) (value.Value, error) {
			delete(w.btEnt, argI(a, 0, 0))
			return z()
		}),
	}
}

func (w *World) addBTNode(tree, kind int, fn string) (value.Value, error) {
	bt := w.bts[tree]
	if bt == nil {
		return value.Value{}, fmt.Errorf("behavior: no tree %d", tree)
	}
	if (kind == btCond || kind == btAct) && fn == "" {
		return value.Value{}, fmt.Errorf("behavior: leaf needs a function name")
	}
	bt.nodes = append(bt.nodes, btNode{kind: kind, fn: fn})
	return value.Num(float64(len(bt.nodes) - 1)), nil
}

func (w *World) tickBehaviors() {
	if len(w.btEnt) == 0 {
		return
	}
	for ent, link := range w.btEnt {
		bt := w.bts[link.tree]
		if bt == nil || link.root < 0 || link.root >= len(bt.nodes) {
			continue
		}
		w.evalBT(bt, link.root, ent)
	}
}

func (w *World) evalBT(bt *behavior, idx, ent int) int {
	if idx < 0 || idx >= len(bt.nodes) {
		return btFail
	}
	n := &bt.nodes[idx]
	switch n.kind {
	case btSeq:
		for n.cursor < len(n.kids) {
			st := w.evalBT(bt, n.kids[n.cursor], ent)
			if st == btRun {
				return btRun
			}
			if st == btFail {
				n.cursor = 0
				return btFail
			}
			n.cursor++
		}
		n.cursor = 0
		return btOK
	case btSel:
		for n.cursor < len(n.kids) {
			st := w.evalBT(bt, n.kids[n.cursor], ent)
			if st == btOK {
				n.cursor = 0
				return btOK
			}
			if st == btRun {
				return btRun
			}
			n.cursor++
		}
		n.cursor = 0
		return btFail
	case btPar:
		ok, run := 0, 0
		for _, kid := range n.kids {
			switch w.evalBT(bt, kid, ent) {
			case btFail:
				return btFail
			case btRun:
				run++
			default:
				ok++
			}
		}
		if run > 0 {
			return btRun
		}
		if ok == len(n.kids) && ok > 0 {
			return btOK
		}
		return btFail
	case btCond, btAct:
		return w.btLeaf(n.kind, n.fn, ent)
	default:
		return btFail
	}
}

func (w *World) btLeaf(kind int, name string, ent int) int {
	if w.btStub != nil {
		return leafCode(kind, float64(w.btStub(name, ent)), nil)
	}
	if w.runner == nil {
		if kind == btAct {
			return btOK
		}
		return btFail
	}
	v, err := w.runner.CallNamed(name, []value.Value{value.Num(float64(ent))})
	return leafCode(kind, v.Number(), err)
}

func leafCode(kind int, n float64, err error) int {
	if err != nil {
		return btFail
	}
	if kind == btCond {
		if n != 0 {
			return btOK
		}
		return btFail
	}
	if n < 0 {
		return btFail
	}
	if n > 0 {
		return btRun
	}
	return btOK
}
