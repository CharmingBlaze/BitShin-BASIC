package runtime

import (
	goruntime "runtime"
	"sync"
	"sync/atomic"

	"bitshinbasic/internal/value"
)

// jobPool is a small Go worker pool. Workers must never create GL objects
// or call G3N upload APIs — queue those with enqueueGL and flush on Flip.
type jobPool struct {
	ch     chan *jobItem
	gl     chan func()
	mu     sync.Mutex
	next   int
	items  map[int]*jobItem
	closed atomic.Bool
	wg     sync.WaitGroup
	queued atomic.Int32
	live   atomic.Int32
}

type jobItem struct {
	id   int
	fn   func()
	done chan struct{}
}

func newJobPool(workers int) *jobPool {
	if workers < 1 {
		workers = goruntime.NumCPU()
	}
	if workers < 1 {
		workers = 2
	}
	if workers > 32 {
		workers = 32
	}
	p := &jobPool{
		ch:    make(chan *jobItem, 256),
		gl:    make(chan func(), 256),
		items: map[int]*jobItem{},
		next:  1,
	}
	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go p.loop()
	}
	return p
}

func (p *jobPool) loop() {
	defer p.wg.Done()
	for j := range p.ch {
		p.queued.Add(-1)
		p.live.Add(1)
		if j.fn != nil {
			j.fn()
		}
		p.live.Add(-1)
		close(j.done)
	}
}

func (p *jobPool) submit(fn func()) int {
	if p == nil || p.closed.Load() {
		if fn != nil {
			fn()
		}
		return 0
	}
	p.mu.Lock()
	id := p.next
	p.next++
	j := &jobItem{id: id, fn: fn, done: make(chan struct{})}
	p.items[id] = j
	p.mu.Unlock()
	p.queued.Add(1)
	p.ch <- j
	return id
}

func (p *jobPool) wait(id int) {
	if p == nil || id <= 0 {
		return
	}
	p.mu.Lock()
	j := p.items[id]
	p.mu.Unlock()
	if j == nil {
		return
	}
	<-j.done
}

func (p *jobPool) waitAll() {
	if p == nil {
		return
	}
	p.mu.Lock()
	ids := make([]int, 0, len(p.items))
	for id := range p.items {
		ids = append(ids, id)
	}
	p.mu.Unlock()
	for _, id := range ids {
		p.wait(id)
	}
}

func (p *jobPool) enqueueGL(fn func()) {
	if p == nil || fn == nil {
		return
	}
	if p.closed.Load() {
		fn()
		return
	}
	select {
	case p.gl <- fn:
	default:
		// queue full: apply on caller (must be GL thread)
		fn()
	}
}

func (p *jobPool) flushGL() {
	if p == nil {
		return
	}
	for {
		select {
		case fn := <-p.gl:
			if fn != nil {
				fn()
			}
		default:
			return
		}
	}
}

func (p *jobPool) pending() int {
	if p == nil {
		return 0
	}
	return int(p.queued.Load() + p.live.Load())
}

func (p *jobPool) close() {
	if p == nil || p.closed.Swap(true) {
		return
	}
	close(p.ch)
	p.wg.Wait()
}

func (w *World) ensureJobs() *jobPool {
	if w.jobs == nil {
		n := w.jobWorkers
		if n < 1 {
			n = goruntime.NumCPU()
		}
		w.jobs = newJobPool(n)
	}
	return w.jobs
}

func (w *World) jobCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		"jobsubmit": n(func(a []value.Value) (value.Value, error) {
			work := argI(a, 0, 0)
			id := w.ensureJobs().submit(func() {
				acc := uint64(0)
				for i := 0; i < work; i++ {
					acc = acc*1315423911 + uint64(i)
				}
				_ = acc
			})
			return value.Num(float64(id)), nil
		}),
		"jobwait": n(func(a []value.Value) (value.Value, error) {
			w.ensureJobs().wait(argI(a, 0, 0))
			return z()
		}),
		"jobwaitall": n(func(a []value.Value) (value.Value, error) {
			w.ensureJobs().waitAll()
			return z()
		}),
		"jobcount": n(func(a []value.Value) (value.Value, error) {
			if w.jobs == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(w.jobs.pending())), nil
		}),
		"jobqueue": n(func(a []value.Value) (value.Value, error) {
			if w.jobs == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(w.jobs.queued.Load())), nil
		}),
		"setjobworkers": n(func(a []value.Value) (value.Value, error) {
			w.jobWorkers = argI(a, 0, goruntime.NumCPU())
			return value.Num(float64(w.jobWorkers)), nil
		}),
		"getjobworkers": n(func(a []value.Value) (value.Value, error) {
			n := w.jobWorkers
			if n <= 0 {
				n = goruntime.NumCPU()
			}
			return value.Num(float64(n)), nil
		}),
	}
}
