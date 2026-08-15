package runtime

import (
	"bufio"
	"os"
	"time"

	"bitshinbasic/internal/value"
)

func (w *World) timeCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		"createtimer": n(func(a []value.Value) (value.Value, error) {
			hz := argN(a, 0, 60)
			if hz <= 0 {
				hz = 60
			}
			id := w.nextTimer
			w.nextTimer++
			w.timers[id] = &blitzTimer{hz: hz, start: time.Now()}
			return value.Num(float64(id)), nil
		}),
		"freetimer": n(func(a []value.Value) (value.Value, error) {
			delete(w.timers, argI(a, 0, 0))
			return z()
		}),
		"timerticks": n(func(a []value.Value) (value.Value, error) {
			t := w.timers[argI(a, 0, 0)]
			if t == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(int64(time.Since(t.start).Seconds() * t.hz))), nil
		}),
		"waittimer": n(func(a []value.Value) (value.Value, error) {
			t := w.timers[argI(a, 0, 0)]
			if t == nil {
				w.waitHold = false
				return value.Num(0), nil
			}
			total := int64(time.Since(t.start).Seconds() * t.hz)
			pending := total - t.consumed
			if pending <= 0 {
				w.waitHold = w.ready
				return value.Num(0), nil
			}
			t.consumed = total
			w.waitHold = false
			return value.Num(float64(pending)), nil
		}),
		"settimer": n(func(a []value.Value) (value.Value, error) {
			name := argS(a, 0)
			ms := argI(a, 1, 1000)
			if ms < 1 {
				ms = 1
			}
			d := time.Duration(ms) * time.Millisecond
			w.namedT[name] = &namedTimer{interval: d, next: time.Now().Add(d)}
			return z()
		}),
		"timerready": n(func(a []value.Value) (value.Value, error) {
			t := w.namedT[argS(a, 0)]
			if t == nil || time.Now().Before(t.next) {
				return value.Num(0), nil
			}
			t.next = time.Now().Add(t.interval)
			return value.Num(1), nil
		}),
		"delay": n(func(a []value.Value) (value.Value, error) {
			ms := argI(a, 0, 0)
			if ms <= 0 {
				w.waitHold = false
				w.delayUntil = time.Time{}
				return z()
			}
			if !w.ready {
				time.Sleep(time.Duration(ms) * time.Millisecond)
				w.waitHold = false
				return z()
			}
			if w.delayUntil.IsZero() {
				w.delayUntil = time.Now().Add(time.Duration(ms) * time.Millisecond)
			}
			if time.Now().Before(w.delayUntil) {
				w.waitHold = true
				return z()
			}
			w.delayUntil = time.Time{}
			w.waitHold = false
			return z()
		}),
		"waitkey": n(func(a []value.Value) (value.Value, error) {
			if !w.ready {
				_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
				w.waitHold = false
				return value.Num(1), nil
			}
			for code, hit := range w.hits {
				if hit {
					w.hits[code] = false
					w.waitHold = false
					return value.Num(float64(code)), nil
				}
			}
			w.waitHold = true
			return value.Num(0), nil
		}),
		"flushkeys": n(func(a []value.Value) (value.Value, error) {
			w.keys = map[int]bool{}
			w.hits = map[int]bool{}
			w.prev = map[int]bool{}
			return z()
		}),
		"flushmouse": n(func(a []value.Value) (value.Value, error) {
			w.mouse = [8]bool{}
			w.mouseHits = [8]bool{}
			w.mxs, w.mys = 0, 0
			return z()
		}),
		"mousehit": n(func(a []value.Value) (value.Value, error) {
			if w.guiCapturesMouse() {
				return value.Num(0), nil
			}
			b := argI(a, 0, 1)
			if b >= 0 && b < len(w.mouseHits) && w.mouseHits[b] {
				w.mouseHits[b] = false
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
	}
}
