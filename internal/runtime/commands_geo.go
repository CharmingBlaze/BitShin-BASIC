package runtime

import (
	"fmt"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/geo"
	"bitshinbasic/internal/value"
)

type geoFrame struct {
	origin    geo.Origin
	lastX     float64
	lastZ     float64
	lastLon   float64
	lastLat   float64
	tileX     int
	tileY     int
	tileZ     int
	jsonCount int
	jsonRoot  int
}

func (w *World) ensureGeo() *geoFrame {
	if w.geo.origin.Scale == 0 {
		w.geo.origin.Scale = 1
	}
	return &w.geo
}

func (w *World) geoCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"setgeoorigin": n(func(a []value.Value) (value.Value, error) {
			g := w.ensureGeo()
			g.origin.Lon = argN(a, 0, 0)
			g.origin.Lat = argN(a, 1, 0)
			if w.stream != nil {
				w.stream.ox, w.stream.oz = 0, 0
			}
			return z()
		}),
		"geooriginlon": n(func(a []value.Value) (value.Value, error) {
			return value.Num(w.ensureGeo().origin.Lon), nil
		}),
		"geooriginlat": n(func(a []value.Value) (value.Value, error) {
			return value.Num(w.ensureGeo().origin.Lat), nil
		}),
		"setgeoscale": n(func(a []value.Value) (value.Value, error) {
			g := w.ensureGeo()
			s := argN(a, 0, 1)
			if s == 0 {
				s = 1
			}
			g.origin.Scale = s
			return value.Num(s), nil
		}),
		"geoscale": n(func(a []value.Value) (value.Value, error) {
			return value.Num(w.ensureGeo().origin.Scale), nil
		}),
		"geoproject": n(func(a []value.Value) (value.Value, error) {
			g := w.ensureGeo()
			x, zz := g.origin.Project(argN(a, 0, 0), argN(a, 1, 0))
			g.lastX, g.lastZ = x, zz
			g.lastLon, g.lastLat = argN(a, 0, 0), argN(a, 1, 0)
			return value.Num(x), nil
		}),
		"geoprojectz": n(func(a []value.Value) (value.Value, error) {
			g := w.ensureGeo()
			if len(a) >= 2 {
				x, zz := g.origin.Project(argN(a, 0, 0), argN(a, 1, 0))
				g.lastX, g.lastZ = x, zz
				return value.Num(zz), nil
			}
			return value.Num(g.lastZ), nil
		}),
		"geounproject": n(func(a []value.Value) (value.Value, error) {
			g := w.ensureGeo()
			lon, lat := g.origin.Unproject(argN(a, 0, 0), argN(a, 1, 0))
			g.lastLon, g.lastLat = lon, lat
			g.lastX, g.lastZ = argN(a, 0, 0), argN(a, 1, 0)
			return value.Num(lon), nil
		}),
		"geounprojectlat": n(func(a []value.Value) (value.Value, error) {
			g := w.ensureGeo()
			if len(a) >= 2 {
				lon, lat := g.origin.Unproject(argN(a, 0, 0), argN(a, 1, 0))
				g.lastLon, g.lastLat = lon, lat
				return value.Num(lat), nil
			}
			return value.Num(g.lastLat), nil
		}),
		"geotilex": n(func(a []value.Value) (value.Value, error) {
			g := w.ensureGeo()
			x, y, z := geo.TileXYZ(argN(a, 0, 0), argN(a, 1, 0), argI(a, 2, 15))
			g.tileX, g.tileY, g.tileZ = x, y, z
			return value.Num(float64(x)), nil
		}),
		"geotiley": n(func(a []value.Value) (value.Value, error) {
			g := w.ensureGeo()
			if len(a) >= 2 {
				x, y, z := geo.TileXYZ(argN(a, 0, 0), argN(a, 1, 0), argI(a, 2, 15))
				g.tileX, g.tileY, g.tileZ = x, y, z
				return value.Num(float64(y)), nil
			}
			return value.Num(float64(g.tileY)), nil
		}),
		"geotilez": n(func(a []value.Value) (value.Value, error) {
			g := w.ensureGeo()
			if len(a) >= 1 && a[0].Kind == value.KindNum && argI(a, 0, 0) >= 0 && argI(a, 0, 0) <= 22 && len(a) == 1 {
				return value.Num(float64(g.tileZ)), nil
			}
			if len(a) >= 2 {
				x, y, z := geo.TileXYZ(argN(a, 0, 0), argN(a, 1, 0), argI(a, 2, 15))
				g.tileX, g.tileY, g.tileZ = x, y, z
				return value.Num(float64(z)), nil
			}
			return value.Num(float64(g.tileZ)), nil
		}),
		"geotmsy": n(func(a []value.Value) (value.Value, error) {
			if len(a) >= 2 {
				return value.Num(float64(geo.TMSY(argI(a, 0, 0), argI(a, 1, 0)))), nil
			}
			g := w.ensureGeo()
			return value.Num(float64(geo.TMSY(g.tileY, g.tileZ))), nil
		}),
		"loadgeojson": need(func(a []value.Value) (value.Value, error) {
			return w.loadGeoJSON(argS(a, 0), argI(a, 1, 0))
		}),
		"geojsoncount": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.ensureGeo().jsonCount)), nil
		}),
		"geoheight": n(func(a []value.Value) (value.Value, error) {
			g := w.ensureGeo()
			x, zz := g.origin.Project(argN(a, 0, 0), argN(a, 1, 0))
			return value.Num(float64(w.terrainHeight(float32(x), float32(zz)))), nil
		}),
		"loadgeodem": need(func(a []value.Value) (value.Value, error) {
			return w.loadGeoDEM(a)
		}),
		"createterrainfromgeodem": need(func(a []value.Value) (value.Value, error) {
			return w.terrainFromGeoHeightmap(a)
		}),
	}
}

func (w *World) loadGeoJSON(rel string, parent int) (value.Value, error) {
	path, err := w.openPath(rel)
	if err != nil {
		return value.Num(0), err
	}
	g := w.ensureGeo()
	col, err := geo.LoadGeoJSONFile(path, g.origin)
	if err != nil {
		return value.Num(0), err
	}
	root := w.addEntity(&Entity{node: core.NewNode()}, parent)
	if e := w.ents[root]; e != nil {
		e.name = "geojson"
	}
	n := 0
	for _, f := range col.Features {
		switch f.Kind {
		case "point":
			if len(f.Points) == 0 {
				continue
			}
			p := f.Points[0]
			id := w.meshEnt(geometry.NewCube(1.2), root)
			if e := w.ents[id]; e != nil {
				e.name = f.Name
				if e.name == "" {
					e.name = "geopoint"
				}
				e.node.GetNode().SetPosition(float32(p[0]), float32(p[1])+0.6, float32(p[2]))
				e.mat.SetColor(&math32.Color{0.85, 0.25, 0.2})
			}
			n++
		case "line", "polygon":
			if len(f.Points) < 2 {
				continue
			}
			path := make([]math32.Vector3, 0, len(f.Points))
			for _, p := range f.Points {
				y := float32(p[1])
				if y == 0 {
					y = w.terrainHeight(float32(p[0]), float32(p[2])) + 0.35
				}
				path = append(path, math32.Vector3{X: float32(p[0]), Y: y, Z: float32(p[2])})
			}
			id := w.meshEnt(geometry.NewTube(path, 0.35, 8, f.Kind == "polygon"), root)
			if e := w.ents[id]; e != nil {
				e.name = f.Name
				if e.name == "" {
					e.name = "geopath"
				}
				e.mat.SetColor(&math32.Color{0.95, 0.75, 0.2})
			}
			n++
		}
	}
	g.jsonCount = n
	g.jsonRoot = root
	return value.Num(float64(root)), nil
}

func flipHeightRows(hf *heightField) {
	if hf == nil || hf.gw < 1 || hf.gd < 2 || len(hf.h) < hf.gw*hf.gd {
		return
	}
	tmp := make([]float32, len(hf.h))
	for row := 0; row < hf.gd; row++ {
		src := (hf.gd - 1 - row) * hf.gw
		dst := row * hf.gw
		copy(tmp[dst:dst+hf.gw], hf.h[src:src+hf.gw])
	}
	hf.h = tmp
}

func (w *World) applyGeoBounds(hf *heightField, west, south, east, north float64) {
	g := w.ensureGeo()
	if g.origin.Lon == 0 && g.origin.Lat == 0 {
		g.origin.Lon = (west + east) / 2
		g.origin.Lat = (south + north) / 2
	}
	x0, z0 := g.origin.Project(west, south)
	x1, z1 := g.origin.Project(east, north)
	ww := float32(x1 - x0)
	dd := float32(z1 - z0)
	if ww < 1 {
		ww = 1
	}
	if dd < 1 {
		dd = 1
	}
	hf.ox = float32(x0)
	hf.oz = float32(z0)
	hf.worldW = ww
	hf.worldD = dd
	flipHeightRows(hf)
}

func (w *World) loadGeoDEM(a []value.Value) (value.Value, error) {
	path, err := w.openPath(argS(a, 0))
	if err != nil {
		return value.Num(0), err
	}
	hs := float32(argN(a, 5, 12))
	hf, err := loadHeightImage(path, 64, 64, hs)
	if err != nil {
		return value.Num(0), err
	}
	w.applyGeoBounds(&hf, argN(a, 1, -0.01), argN(a, 2, -0.01), argN(a, 3, 0.01), argN(a, 4, 0.01))
	return value.Num(float64(w.addTerrain(hf, 17, 2))), nil
}

func (w *World) terrainFromGeoHeightmap(a []value.Value) (value.Value, error) {
	hm := w.pickHeightMap(a, 0)
	if hm == nil {
		return value.Num(0), fmt.Errorf("CreateTerrainFromGeoDEM: no heightmap")
	}
	hs := float32(argN(a, 5, 12))
	hf := heightFieldFromMap(hm, 64, 64, hs)
	w.applyGeoBounds(&hf, argN(a, 1, -0.01), argN(a, 2, -0.01), argN(a, 3, 0.01), argN(a, 4, 0.01))
	return value.Num(float64(w.addTerrain(hf, 17, 2))), nil
}
