package runtime

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gobwas/pool"
	"github.com/gobwas/pool/pbytes"
	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/value"
)

type userPool struct {
	name string
	size int
	free []int
}

type bankSlot struct {
	data []byte
	pool int
}

var (
	byteBags   = pbytes.New(64, 1<<20)
	sizeBags   = pool.New(16, 1<<20)
	entBag     = sync.Pool{New: func() any { return &Entity{} }}
	sbBag      = sync.Pool{New: func() any { return &strings.Builder{} }}
	vecBag     = sync.Pool{New: func() any { return &math32.Vector3{} }}
)

func (w *World) takeHandle(free *[]int, next *int) int {
	if len(*free) > 0 {
		id := (*free)[len(*free)-1]
		*free = (*free)[:len(*free)-1]
		return id
	}
	if *next < 1 {
		*next = 1
	}
	id := *next
	*next++
	return id
}

func (w *World) recycleHandle(free *[]int, id int) {
	if id > 0 {
		*free = append(*free, id)
	}
}

func scratchBuilder() *strings.Builder {
	b := sbBag.Get().(*strings.Builder)
	b.Reset()
	return b
}

func releaseBuilder(b *strings.Builder) {
	if b != nil {
		b.Reset()
		sbBag.Put(b)
	}
}

func scratchVec() *math32.Vector3 {
	return vecBag.Get().(*math32.Vector3)
}

func releaseVec(v *math32.Vector3) {
	if v != nil {
		*v = math32.Vector3{}
		vecBag.Put(v)
	}
}

func (w *World) poolCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		"createpool": n(func(a []value.Value) (value.Value, error) {
			name := argS(a, 0)
			sz := argI(a, 1, 64)
			if sz < 1 {
				sz = 1
			}
			id := w.takeHandle(&w.freePools, &w.nextPool)
			if w.pools == nil {
				w.pools = map[int]*userPool{}
			}
			w.pools[id] = &userPool{name: name, size: sz}
			return value.Num(float64(id)), nil
		}),
		"poolget": n(func(a []value.Value) (value.Value, error) {
			p, err := w.poolOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if len(p.free) > 0 {
				id := p.free[len(p.free)-1]
				p.free = p.free[:len(p.free)-1]
				return value.Num(float64(id)), nil
			}
			raw := byteBags.GetLen(p.size)
			id := w.takeHandle(&w.freeBanks, &w.nextBank)
			if w.banks == nil {
				w.banks = map[int]*bankSlot{}
			}
			w.banks[id] = &bankSlot{data: raw[:p.size], pool: argI(a, 0, 0)}
			return value.Num(float64(id)), nil
		}),
		"poolput": n(func(a []value.Value) (value.Value, error) {
			p, err := w.poolOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			item := argI(a, 1, 0)
			if b := w.banks[item]; b != nil {
				p.free = append(p.free, item)
			}
			return z()
		}),
		"poolclear": n(func(a []value.Value) (value.Value, error) {
			p, err := w.poolOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			for _, id := range p.free {
				if b := w.banks[id]; b != nil {
					byteBags.Put(b.data)
					delete(w.banks, id)
					w.recycleHandle(&w.freeBanks, id)
				}
			}
			p.free = p.free[:0]
			return z()
		}),
		"createbank": n(func(a []value.Value) (value.Value, error) {
			sz := argI(a, 0, 64)
			if sz < 1 {
				sz = 1
			}
			raw := byteBags.GetLen(sz)
			id := w.takeHandle(&w.freeBanks, &w.nextBank)
			if w.banks == nil {
				w.banks = map[int]*bankSlot{}
			}
			w.banks[id] = &bankSlot{data: raw[:sz]}
			return value.Num(float64(id)), nil
		}),
		"creatememblock": n(func(a []value.Value) (value.Value, error) {
			return w.Call("createbank", a)
		}),
		"freebank": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			if b := w.banks[id]; b != nil {
				byteBags.Put(b.data)
				delete(w.banks, id)
				w.recycleHandle(&w.freeBanks, id)
			}
			return z()
		}),
		"banksize": n(func(a []value.Value) (value.Value, error) {
			if b := w.banks[argI(a, 0, 0)]; b != nil {
				return value.Num(float64(len(b.data))), nil
			}
			return value.Num(0), nil
		}),
	}
}

func (w *World) poolOf(id int) (*userPool, error) {
	p := w.pools[id]
	if p == nil {
		return nil, fmt.Errorf("invalid pool %d", id)
	}
	return p, nil
}

func (w *World) freeEntityID(id int) {
	if em := w.emitters[id]; em != nil {
		w.releaseEmitterParts(em)
		delete(w.emitters, id)
	}
	e, ok := w.ents[id]
	if !ok {
		return
	}
	if e.node != nil {
		n := e.node.GetNode()
		if p := n.Parent(); p != nil {
			p.GetNode().Remove(e.node)
		}
		n.SetVisible(false)
	}
	delete(w.ents, id)
	w.recycleHandle(&w.freeIDs, id)
	*e = Entity{}
	entBag.Put(e)
	_ = sizeBags
}
