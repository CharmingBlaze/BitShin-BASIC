package runtime

import "testing"

// A full GL queue must hold work for the GL thread, not run it on the worker:
// that raced the terrain maps once more than 256 chunks were waiting.
func TestEnqueueGLOverflowWaitsForFlush(t *testing.T) {
	p := newJobPool(2)
	defer p.close()
	const n = 600
	ran := 0
	id := p.submit(func() {
		for i := 0; i < n; i++ {
			p.enqueueGL(func() { ran++ })
		}
	})
	p.wait(id)
	if ran != 0 {
		t.Fatalf("%d jobs ran on the worker", ran)
	}
	p.flushGL()
	if ran != n {
		t.Fatalf("flush ran %d of %d jobs", ran, n)
	}
}
