package runtime

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestJobPoolSubmitWait(t *testing.T) {
	p := newJobPool(2)
	var n atomic.Int32
	id := p.submit(func() {
		time.Sleep(5 * time.Millisecond)
		n.Add(1)
	})
	if id <= 0 {
		t.Fatal("job id")
	}
	p.wait(id)
	if n.Load() != 1 {
		t.Fatalf("work not done: %d", n.Load())
	}
	id2 := p.submit(func() { n.Add(10) })
	id3 := p.submit(func() { n.Add(100) })
	p.waitAll()
	if n.Load() != 111 {
		t.Fatalf("waitAll got %d", n.Load())
	}
	_ = id2
	_ = id3
}

func TestJobCommandsRegistered(t *testing.T) {
	m := New(".").commandTable()
	for _, name := range []string{"jobsubmit", "jobwait", "jobwaitall", "jobcount"} {
		if m[name] == nil {
			t.Errorf("missing %s", name)
		}
	}
}
