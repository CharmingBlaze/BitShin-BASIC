package runtime

import (
	"image"
	"image/color"
	"math"

	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/texture"
)

// treeImpostor is a 3D tree template drawn once, side on, into a picture.
// Past the 3D blocks the same trees stand as cards of it, out to the fog.
type treeImpostor struct {
	tex           *texture.Texture2D
	width, height float32 // card size at scale 1, in meters
}

// impostorPx is the picture's long side. A tree past the 3D reach is a few
// dozen pixels tall at most.
const impostorPx = 128

func (w *World) treeImpostorFor(mesh int) *treeImpostor {
	if imp, ok := w.treeImps[mesh]; ok {
		return imp
	}
	parts, minY := w.treeMeshParts(mesh)
	var imp *treeImpostor
	if img, wd, ht := drawImpostor(parts, minY, impostorPx); img != nil {
		imp = &treeImpostor{tex: texture.NewTexture2DFromRGBA(img), width: wd, height: ht}
	}
	if w.treeImps == nil {
		w.treeImps = map[int]*treeImpostor{}
	}
	w.treeImps[mesh] = imp
	return imp
}

// drawImpostor rasterizes the parts seen from the side, 3x supersampled,
// with shape shading only: the card is lit again by the scene. Covered
// colors bleed into the empty texels so mipmaps keep no dark fringe. It
// returns the picture and its size in meters.
func drawImpostor(parts []meshPart, minY float32, px int) (*image.RGBA, float32, float32) {
	maxR, top := float32(0), minY
	for _, p := range parts {
		for _, t := range p.tris {
			for _, v := range t {
				maxR = math32.Max(maxR, math32.Max(math32.Abs(v.X), math32.Abs(v.Z)))
				top = math32.Max(top, v.Y)
			}
		}
	}
	width, height := 2*maxR, top-minY
	if width <= 0 || height <= 0 || px < 8 {
		return nil, 0, 0
	}
	W, H := px, px
	if width < height {
		W = int(math.Round(float64(float32(px) * width / height)))
	} else {
		H = int(math.Round(float64(float32(px) * height / width)))
	}
	if W < 8 {
		W = 8
	}
	if H < 8 {
		H = 8
	}
	const ss = 3
	sw, sh := W*ss, H*ss
	depth := make([]float32, sw*sh)
	for i := range depth {
		depth[i] = math32.Infinity
	}
	col := make([][3]float32, sw*sh)
	hit := make([]bool, sw*sh)
	light := math32.Vector3{X: -0.35, Y: 0.75, Z: -0.55}
	light.Normalize()
	for _, p := range parts {
		for ti, t := range p.tris {
			var n math32.Vector3
			if ti < len(p.nors) {
				for _, c := range p.nors[ti] {
					n.Add(&c)
				}
			}
			if n.LengthSq() < 1e-8 {
				a, b := t[1], t[2]
				a.Sub(&t[0])
				b.Sub(&t[0])
				n.CrossVectors(&a, &b)
			}
			n.Normalize()
			lit := 0.78 + 0.26*math32.Max(0, math32.Abs(n.Dot(&light)))
			var sx, sy, sz [3]float32
			for i, v := range t {
				sx[i] = (v.X + maxR) / width * float32(sw)
				sy[i] = (top - v.Y) / height * float32(sh)
				sz[i] = v.Z // the viewer is at -Z
			}
			x0 := int(math32.Max(0, math32.Floor(min3(sx))))
			x1 := int(math32.Min(float32(sw-1), math32.Ceil(max3(sx))))
			y0 := int(math32.Max(0, math32.Floor(min3(sy))))
			y1 := int(math32.Min(float32(sh-1), math32.Ceil(max3(sy))))
			area := (sx[1]-sx[0])*(sy[2]-sy[0]) - (sx[2]-sx[0])*(sy[1]-sy[0])
			if math32.Abs(area) < 1e-6 {
				continue
			}
			for y := y0; y <= y1; y++ {
				for x := x0; x <= x1; x++ {
					fx, fy := float32(x)+0.5, float32(y)+0.5
					w0 := ((sx[1]-fx)*(sy[2]-fy) - (sx[2]-fx)*(sy[1]-fy)) / area
					w1 := ((sx[2]-fx)*(sy[0]-fy) - (sx[0]-fx)*(sy[2]-fy)) / area
					w2 := 1 - w0 - w1
					if w0 < 0 || w1 < 0 || w2 < 0 {
						continue
					}
					z := w0*sz[0] + w1*sz[1] + w2*sz[2]
					i := y*sw + x
					if z >= depth[i] {
						continue
					}
					depth[i] = z
					// Darker toward the base, like light lost under a crown
					vy := 1 - fy/float32(sh)
					s := lit * (0.86 + 0.14*vy)
					col[i] = [3]float32{p.color.R * s, p.color.G * s, p.color.B * s}
					hit[i] = true
				}
			}
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	covered := make([]bool, W*H)
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			var r, g, b float32
			n := 0
			for j := 0; j < ss; j++ {
				for i := 0; i < ss; i++ {
					k := (y*ss+j)*sw + x*ss + i
					if hit[k] {
						r += col[k][0]
						g += col[k][1]
						b += col[k][2]
						n++
					}
				}
			}
			if n == 0 {
				continue
			}
			f := float32(n)
			covered[y*W+x] = true
			img.SetRGBA(x, y, color.RGBA{to8(r / f), to8(g / f), to8(b / f), uint8(255 * n / (ss * ss))})
		}
	}
	bleedImpostor(img, covered)
	return img, width, height
}

// bleedImpostor grows the covered colors into every empty texel, alpha kept
// 0, so even the smallest mip averages crown colors and not black.
func bleedImpostor(img *image.RGBA, covered []bool) {
	W, H := img.Rect.Dx(), img.Rect.Dy()
	for grew := true; grew; {
		grew = false
		next := append([]bool(nil), covered...)
		for y := 0; y < H; y++ {
			for x := 0; x < W; x++ {
				if covered[y*W+x] {
					continue
				}
				var r, g, b, n int
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						xx, yy := x+dx, y+dy
						if xx < 0 || yy < 0 || xx >= W || yy >= H || !covered[yy*W+xx] {
							continue
						}
						c := img.RGBAAt(xx, yy)
						r += int(c.R)
						g += int(c.G)
						b += int(c.B)
						n++
					}
				}
				if n > 0 {
					img.SetRGBA(x, y, color.RGBA{uint8(r / n), uint8(g / n), uint8(b / n), img.RGBAAt(x, y).A})
					next[y*W+x] = true
					grew = true
				}
			}
		}
		covered = next
	}
}

func to8(v float32) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(v*255 + 0.5)
}

func min3(v [3]float32) float32 { return math32.Min(v[0], math32.Min(v[1], v[2])) }
func max3(v [3]float32) float32 { return math32.Max(v[0], math32.Max(v[1], v[2])) }

// impostorReach is how far, in chunks, impostor cards go: to the fog, or
// twice the 3D reach without fog. The horizon ring caps it.
func (w *World) impostorReach(cw float32, n int) int {
	far := 2 * w.meshTreeReach()
	if w.fogMode != 0 && w.fogFar > 0 {
		far = w.fogFar
	}
	r := int(math.Ceil(float64(far / cw)))
	if r > n {
		r = n
	}
	return r
}

// placeTreeImpostors plants every 3D tree layer's trees as cards on the
// chunks whose block is past the 3D trees, in batches of two blocks a side
// so a far batch seldom needs turning toward the camera.
func (w *World) placeTreeImpostors(t *terrain, cw float32, cam chunkKey, n int) {
	add := w.meshTreeAdd(t, cw)
	outer := w.impostorReach(cw, n)
	if float32(outer) <= add-float32(treeBlock) {
		return
	}
	const group = 2 * treeBlock
	for li, layer := range t.scatters {
		if layer.kind != "tree" || layer.density <= 0 || !w.isMeshTreeLayer(layer) {
			continue
		}
		imp := w.treeImpostorFor(layer.mesh)
		if imp == nil {
			continue
		}
		batches := map[chunkKey][]scatterPt{}
		var order []chunkKey
		for z := cam.Z - outer; z <= cam.Z+outer; z++ {
			for x := cam.X - outer; x <= cam.X+outer; x++ {
				dx, dz := x-cam.X, z-cam.Z
				if dx*dx+dz*dz > outer*outer {
					continue
				}
				b := chunkKey{floorDiv(x, treeBlock), floorDiv(z, treeBlock)}
				if treeBlockDist(b, cam) <= add {
					continue // a 3D block
				}
				pts := w.scatterPoints(t, layer, chunkKey{X: x, Z: z}, li, float32(x)*cw, float32(z)*cw, cw, 1)
				if len(pts) == 0 {
					continue
				}
				g := chunkKey{floorDiv(x, group), floorDiv(z, group)}
				if _, ok := batches[g]; !ok {
					order = append(order, g)
				}
				batches[g] = append(batches[g], pts...)
			}
		}
		for _, g := range order {
			pts := batches[g]
			id := w.addTreeCardMesh(t, pts, false, &math32.Color{R: 1, G: 1, B: 1}, imp.tex, imp.width, imp.height)
			if id == 0 {
				continue
			}
			if e := w.ents[id]; e != nil {
				e.meshNoCast = true // far past the shadow cascades
				w.setEntityCutout(e, 3)
			}
			t.horizonTrees = append(t.horizonTrees, id)
			t.cards = append(t.cards, treeCard{
				ent: id, pts: pts, width: imp.width, height: imp.height, face: true,
			})
		}
	}
}
