package runtime

import (
	"sort"

	"github.com/g3n/engine/texture"

	"bitshinbasic/internal/value"
)

func (w *World) scaleCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"statsfps": n(func(a []value.Value) (value.Value, error) {
			if w.delta <= 0 {
				return value.Num(60), nil
			}
			return value.Num(1 / w.delta), nil
		}),
		"statsdraws": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.visibleMeshCount())), nil
		}),
		"statschunks": n(func(a []value.Value) (value.Value, error) {
			nch := w.streamChunkCount()
			if t := w.terrains[w.curTerrain]; t != nil {
				nch += len(t.ents)
			}
			return value.Num(float64(nch)), nil
		}),
		"statsjobs": n(func(a []value.Value) (value.Value, error) {
			if w.jobs == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(w.jobs.pending())), nil
		}),
		"entitycount": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(len(w.ents))), nil
		}),
		"entitybyindex": n(func(a []value.Value) (value.Value, error) {
			ids := make([]int, 0, len(w.ents))
			for id := range w.ents {
				ids = append(ids, id)
			}
			sort.Ints(ids)
			i := argI(a, 0, 1) - 1
			if i < 0 || i >= len(ids) {
				return value.Num(0), nil
			}
			return value.Num(float64(ids[i])), nil
		}),
		"reloadtexture": need(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			file := argS(a, 1)
			slot := w.texs[id]
			if slot == nil {
				return z()
			}
			if file == "" {
				file = slot.path
			}
			path, err := w.openPath(file)
			if err != nil {
				return z()
			}
			tex, err := texture.NewTexture2DFromImage(path)
			if err != nil {
				return z()
			}
			slot.tex = tex
			slot.path = file
			return value.Num(float64(id)), nil
		}),
		"reloadmesh": need(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			e := w.ents[id]
			if e == nil {
				return z()
			}
			nid, err := w.loadMeshFile(argS(a, 1), e.parent)
			if err != nil {
				return value.Value{}, err
			}
			if ne := w.ents[nid]; ne != nil && e.node != nil {
				p := e.node.GetNode().Position()
				ne.node.GetNode().SetPosition(p.X, p.Y, p.Z)
			}
			return value.Num(float64(nid)), nil
		}),
		"bodysleep": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			if s, ok := w.phys3.(interface{ Sleep(int) }); ok {
				s.Sleep(argI(a, 0, 0))
			}
			return z()
		}),
		"bodywake": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			if s, ok := w.phys3.(interface{ Wake(int) }); ok {
				s.Wake(argI(a, 0, 0))
			}
			return z()
		}),
		"setccd": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			on := argI(a, 1, 1) != 0
			if len(a) == 1 {
				on = argI(a, 0, 1) != 0
				return value.Num(float64(w.phys3.SetCCD(0, on))), nil
			}
			return value.Num(float64(w.phys3.SetCCD(argI(a, 0, 0), on))), nil
		}),
		"setbodyccd": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			return value.Num(float64(w.phys3.SetCCD(argI(a, 0, 0), argI(a, 1, 1) != 0))), nil
		}),
		"getbodyccd": n(func(a []value.Value) (value.Value, error) {
			w.ensurePhys3()
			return value.Num(float64(w.phys3.GetCCD(argI(a, 0, 0)))), nil
		}),
		"physicsthreads": n(func(a []value.Value) (value.Value, error) {
			if len(a) > 0 {
				w.applyPhysThreads(argI(a, 0, 0))
			}
			return value.Num(float64(w.physThreads)), nil
		}),
		"physicsasync": n(func(a []value.Value) (value.Value, error) {
			w.physAsync = argI(a, 0, 1) != 0
			return value.Num(float64(bool01(w.physAsync))), nil
		}),
	}
}

func (w *World) applyPhysThreads(n int) {
	if n < 0 {
		n = 0
	}
	if n > 32 {
		n = 32
	}
	w.physThreads = n
	if n > 0 {
		w.jobWorkers = n
		if w.jobs != nil {
			w.jobs.waitAll()
			w.jobs.close()
			w.jobs = nil
		}
	}
	if w.phys3 != nil {
		w.phys3.SetJobThreads(n)
	}
}
