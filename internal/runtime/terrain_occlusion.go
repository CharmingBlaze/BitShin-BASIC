package runtime

import (
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

// Terrain occlusion: forests behind a hill are not drawn. The height field answers
// this on the CPU with no GPU queries, so there is no read back, no frame of delay,
// and it works the same on a ten-year-old machine. A block of trees is hidden when
// every line from the eye to points across its top goes through the ground.
//
// It covers the main view only. A shadow or a reflection can come from behind a hill,
// and those passes run before this one.

const (
	occludeTreeTop  = 30 // meters above the ground a block's trees may reach
	occludeMargin   = 1  // the ground must clear the line by this much, against popping
	occludeMaxSteps = 24
)

type occlusionState struct {
	off    bool
	hidden int // tree batches hidden last frame
}

// lineUnderGround says whether the ground rises above the line between two points.
func lineUnderGround(height func(x, z float32) float32, ax, ay, az, bx, by, bz, stepM float32) bool {
	dx, dz := bx-ax, bz-az
	d := math32.Sqrt(dx*dx + dz*dz)
	if d < stepM*2 {
		return false
	}
	n := int(d / stepM)
	if n > occludeMaxSteps {
		n = occludeMaxSteps
	}
	for i := 1; i < n; i++ {
		f := float32(i) / float32(n)
		if height(ax+dx*f, az+dz*f) > ay+(by-ay)*f+occludeMargin {
			return true
		}
	}
	return false
}

// blockBehindGround tests a grid of points across the top of a square of ground.
func blockBehindGround(height func(x, z float32) float32, ex, ey, ez, x0, z0, size, top, stepM float32) bool {
	for j := 0; j <= 2; j++ {
		for i := 0; i <= 2; i++ {
			x := x0 + size*float32(i)*0.5
			z := z0 + size*float32(j)*0.5
			if !lineUnderGround(height, ex, ey, ez, x, height(x, z)+top, z, stepM) {
				return false
			}
		}
	}
	return true
}

// hideOccluded hides tree blocks behind the ground for one pass and returns what puts
// them back.
func (w *World) hideOccluded(cam *camera.Camera) func() {
	w.occlusion.hidden = 0
	if w.occlusion.off || cam == nil {
		return func() {}
	}
	p := worldPos(cam.GetNode())
	ex, ey, ez := fromG3N(p.X, p.Y, p.Z)
	var hidden []*graphic.Mesh
	for _, t := range w.terrains {
		if t == nil || len(t.treeMesh) == 0 {
			continue
		}
		if !t.hf.proc && ey > t.hf.hscale {
			continue // above every peak: nothing can be hidden
		}
		height := func(x, z float32) float32 { return w.sampleHeight(&t.hf, x, z) }
		cw := t.chunkWorld()
		size := cw * treeBlock
		step := math32.Max(cw/4, 4)
		for b, ids := range t.treeMesh {
			x0, z0 := float32(b.X)*size, float32(b.Z)*size
			if !blockBehindGround(height, ex, ey, ez, x0, z0, size, occludeTreeTop, step) {
				continue
			}
			for _, id := range ids {
				if e := w.ents[id]; e != nil && e.mesh != nil && e.mesh.Visible() {
					e.mesh.SetVisible(false)
					hidden = append(hidden, e.mesh)
				}
			}
		}
	}
	w.occlusion.hidden = len(hidden)
	return func() {
		for _, m := range hidden {
			m.SetVisible(true)
		}
	}
}

func (w *World) occlusionCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		// SetTerrainOcclusion(on): don't draw trees behind hills. On by default.
		"setterrainocclusion": n(func(a []value.Value) (value.Value, error) {
			w.occlusion.off = argI(a, 0, 1) == 0
			return z()
		}),
		// StatsOccluded(): tree batches the ground hid last frame.
		"statsoccluded": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.occlusion.hidden)), nil
		}),
	}
}
