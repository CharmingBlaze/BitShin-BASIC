// Package geo is a thin, pure-Go subset of flywave/go-geo for BitShin BASIC.
//
// Upstream go-geo (https://github.com/flywave/go-geo) is a tile-proxy helper:
// EPSG/SRS via PROJ, GEOS coverages, TMS/XYZ grids, GCJ-02, geoid, geometry
// reprojection. Its go.mod replace-points at sibling CGO trees (go-proj,
// go-geos, go-geoid) and needs native PROJ + GEOS — not a clean Windows
// dependency for this module.
//
// This package keeps the pieces a game author needs: WGS84 ↔ Web Mercator
// (EPSG:3857, same constants as go-geo), local XZ around an origin, XYZ/TMS
// tiles, and a small GeoJSON loader. No CGO, no FlatBuffers, no PROJ data.
package geo

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
)

// OriginShift is π * 6378137, matching go-geo's MERC_BBOX half-extent.
const OriginShift = math.Pi * 6378137.0

// MaxMercLat is the Web Mercator latitude clamp (same as OSM/Google).
const MaxMercLat = 85.05112878

// Origin is a WGS84 lon/lat that maps to world (0, 0) on XZ.
type Origin struct {
	Lon, Lat float64
	Scale    float64 // world units per mercator metre (default 1)
}

func (o Origin) scale() float64 {
	if o.Scale == 0 {
		return 1
	}
	return o.Scale
}

func clampLat(lat float64) float64 {
	if lat > MaxMercLat {
		return MaxMercLat
	}
	if lat < -MaxMercLat {
		return -MaxMercLat
	}
	return lat
}

// LonLatToMerc converts WGS84 degrees to EPSG:3857 metres (lon→X, lat→Y north).
func LonLatToMerc(lon, lat float64) (x, y float64) {
	lat = clampLat(lat)
	x = lon * OriginShift / 180
	y = math.Log(math.Tan((90+lat)*math.Pi/360)) / (math.Pi / 180)
	y = y * OriginShift / 180
	return x, y
}

// MercToLonLat inverts LonLatToMerc.
func MercToLonLat(x, y float64) (lon, lat float64) {
	lon = (x / OriginShift) * 180
	lat = (y / OriginShift) * 180
	lat = 180 / math.Pi * (2*math.Atan(math.Exp(lat*math.Pi/180)) - math.Pi/2)
	return lon, lat
}

// Project maps lon/lat to world XZ (+X east, +Z north) relative to the origin.
func (o Origin) Project(lon, lat float64) (x, z float64) {
	mx, my := LonLatToMerc(lon, lat)
	ox, oy := LonLatToMerc(o.Lon, o.Lat)
	s := o.scale()
	return (mx - ox) * s, (my - oy) * s
}

// Unproject maps world XZ back to WGS84 degrees.
func (o Origin) Unproject(x, z float64) (lon, lat float64) {
	ox, oy := LonLatToMerc(o.Lon, o.Lat)
	s := o.scale()
	if s == 0 {
		s = 1
	}
	return MercToLonLat(ox+x/s, oy+z/s)
}

// TileXYZ is the OSM/Google (XYZ) tile for a lon/lat at zoom (origin upper-left).
func TileXYZ(lon, lat float64, zoom int) (x, y, z int) {
	if zoom < 0 {
		zoom = 0
	}
	if zoom > 22 {
		zoom = 22
	}
	n := float64(uint64(1) << uint(zoom))
	lat = clampLat(lat)
	x = int(math.Floor((lon + 180.0) / 360.0 * n))
	latRad := lat * math.Pi / 180
	y = int(math.Floor((1.0 - math.Log(math.Tan(latRad)+1/math.Cos(latRad))/math.Pi) / 2.0 * n))
	max := int(n) - 1
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x > max {
		x = max
	}
	if y > max {
		y = max
	}
	return x, y, zoom
}

// TMSY flips an XYZ tile Y into TMS (origin lower-left), as go-geo grids do.
func TMSY(xyzY, zoom int) int {
	if zoom < 0 {
		return xyzY
	}
	return (1 << uint(zoom)) - 1 - xyzY
}

// Feature is a projected GeoJSON point or polyline (game XZ, optional Y).
type Feature struct {
	Kind       string // "point", "line", "polygon"
	Name       string
	Points     [][3]float64 // x, y, z
	Properties map[string]any
}

// Collection is the result of LoadGeoJSON.
type Collection struct {
	Features []Feature
}

type rawGeoJSON struct {
	Type     string          `json:"type"`
	Name     string          `json:"name"`
	Geometry json.RawMessage `json:"geometry"`
	Features []rawGeoJSON    `json:"features"`
	Coords   json.RawMessage `json:"coordinates"`
	Props    map[string]any  `json:"properties"`
}

// LoadGeoJSONFile reads a FeatureCollection, Feature, or Geometry from disk.
func LoadGeoJSONFile(path string, o Origin) (Collection, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Collection{}, err
	}
	return ParseGeoJSON(b, o)
}

// ParseGeoJSON decodes GeoJSON and projects coordinates with o.
func ParseGeoJSON(data []byte, o Origin) (Collection, error) {
	var raw rawGeoJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return Collection{}, fmt.Errorf("geojson: %w", err)
	}
	var out Collection
	if err := collectFeatures(&out, raw, o); err != nil {
		return Collection{}, err
	}
	return out, nil
}

func collectFeatures(out *Collection, raw rawGeoJSON, o Origin) error {
	t := strings.ToLower(strings.TrimSpace(raw.Type))
	switch t {
	case "featurecollection":
		for _, f := range raw.Features {
			if err := collectFeatures(out, f, o); err != nil {
				return err
			}
		}
	case "feature":
		var g rawGeoJSON
		if len(raw.Geometry) > 0 {
			if err := json.Unmarshal(raw.Geometry, &g); err != nil {
				return fmt.Errorf("geojson feature geometry: %w", err)
			}
		}
		name := raw.Name
		if name == "" && raw.Props != nil {
			if s, ok := raw.Props["name"].(string); ok {
				name = s
			}
		}
		g.Name = name
		g.Props = raw.Props
		return collectFeatures(out, g, o)
	case "point", "multipoint", "linestring", "multilinestring", "polygon", "multipolygon":
		return appendGeom(out, raw, o)
	default:
		if len(raw.Features) > 0 {
			for _, f := range raw.Features {
				if err := collectFeatures(out, f, o); err != nil {
					return err
				}
			}
			return nil
		}
		if t == "" {
			return fmt.Errorf("geojson: missing type")
		}
		return fmt.Errorf("geojson: unsupported type %q", raw.Type)
	}
	return nil
}

func appendGeom(out *Collection, raw rawGeoJSON, o Origin) error {
	t := strings.ToLower(raw.Type)
	name := raw.Name
	switch t {
	case "point":
		pt, err := decodePos(raw.Coords)
		if err != nil {
			return err
		}
		out.Features = append(out.Features, Feature{Kind: "point", Name: name, Points: [][3]float64{projectPos(o, pt)}, Properties: raw.Props})
	case "multipoint":
		pts, err := decodeLine(raw.Coords)
		if err != nil {
			return err
		}
		for _, p := range pts {
			out.Features = append(out.Features, Feature{Kind: "point", Name: name, Points: [][3]float64{projectPos(o, p)}, Properties: raw.Props})
		}
	case "linestring":
		pts, err := decodeLine(raw.Coords)
		if err != nil {
			return err
		}
		out.Features = append(out.Features, Feature{Kind: "line", Name: name, Points: projectLine(o, pts), Properties: raw.Props})
	case "multilinestring":
		rings, err := decodeRings(raw.Coords)
		if err != nil {
			return err
		}
		for _, r := range rings {
			out.Features = append(out.Features, Feature{Kind: "line", Name: name, Points: projectLine(o, r), Properties: raw.Props})
		}
	case "polygon":
		rings, err := decodeRings(raw.Coords)
		if err != nil {
			return err
		}
		if len(rings) > 0 {
			out.Features = append(out.Features, Feature{Kind: "polygon", Name: name, Points: projectLine(o, rings[0]), Properties: raw.Props})
		}
	case "multipolygon":
		polys, err := decodePolys(raw.Coords)
		if err != nil {
			return err
		}
		for _, rings := range polys {
			if len(rings) > 0 {
				out.Features = append(out.Features, Feature{Kind: "polygon", Name: name, Points: projectLine(o, rings[0]), Properties: raw.Props})
			}
		}
	}
	return nil
}

func projectPos(o Origin, p []float64) [3]float64 {
	lon, lat := p[0], p[1]
	x, z := o.Project(lon, lat)
	y := 0.0
	if len(p) > 2 {
		y = p[2]
	}
	return [3]float64{x, y, z}
}

func projectLine(o Origin, pts [][]float64) [][3]float64 {
	out := make([][3]float64, 0, len(pts))
	for _, p := range pts {
		if len(p) < 2 {
			continue
		}
		out = append(out, projectPos(o, p))
	}
	return out
}

func decodePos(raw json.RawMessage) ([]float64, error) {
	var p []float64
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("geojson point: %w", err)
	}
	if len(p) < 2 {
		return nil, fmt.Errorf("geojson point: need lon,lat")
	}
	return p, nil
}

func decodeLine(raw json.RawMessage) ([][]float64, error) {
	var pts [][]float64
	if err := json.Unmarshal(raw, &pts); err != nil {
		return nil, fmt.Errorf("geojson line: %w", err)
	}
	return pts, nil
}

func decodeRings(raw json.RawMessage) ([][][]float64, error) {
	var rings [][][]float64
	if err := json.Unmarshal(raw, &rings); err != nil {
		return nil, fmt.Errorf("geojson rings: %w", err)
	}
	return rings, nil
}

func decodePolys(raw json.RawMessage) ([][][][]float64, error) {
	var polys [][][][]float64
	if err := json.Unmarshal(raw, &polys); err != nil {
		return nil, fmt.Errorf("geojson multipolygon: %w", err)
	}
	return polys, nil
}
