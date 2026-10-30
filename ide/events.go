package main

import (
	"encoding/json"
	"fmt"
	"sync"
)

type eventHub struct {
	mu     sync.Mutex
	subs   map[chan []byte]struct{}
	recent [][]byte
}

func newEventHub() *eventHub {
	return &eventHub{subs: map[chan []byte]struct{}{}}
}

func (h *eventHub) publish(frame []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.recent = append(h.recent, frame)
	if len(h.recent) > 64 {
		h.recent = h.recent[len(h.recent)-64:]
	}
	for ch := range h.subs {
		select {
		case ch <- frame:
		default:
			// A full stdout burst must not drop the newest frame (run:exit / run:stopped).
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- frame:
			default:
			}
		}
	}
}

func (h *eventHub) subscribe() (<-chan []byte, func()) {
	ch := make(chan []byte, 32)
	h.mu.Lock()
	for _, frame := range h.recent {
		select {
		case ch <- frame:
		default:
		}
	}
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		if _, ok := h.subs[ch]; ok {
			delete(h.subs, ch)
			close(ch)
		}
		h.mu.Unlock()
	}
}

func (a *App) emit(event string, payload any) {
	if a == nil || a.hub == nil {
		return
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	a.hub.publish([]byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event, b)))
}
