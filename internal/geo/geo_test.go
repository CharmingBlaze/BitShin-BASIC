package geo

import (
	"math"
	"testing"
)

func TestMercatorRoundTrip(t *testing.T) {
	pairs := [][2]float64{
		{0, 0},
		{-73.9857, 40.7484},
		{139.6917, 35.6895},
		{12.4964, 41.9028},
	}
	for _, p := range pairs {
		x, y := LonLatToMerc(p[0], p[1])
		lon, lat := MercToLonLat(x, y)
		if math.Abs(lon-p[0]) > 1e-8 || math.Abs(lat-p[1]) > 1e-8 {
			t.Fatalf("roundtrip %v → %v %v → %v %v", p, x, y, lon, lat)
		}
	}
}

func TestMercatorNullIsland(t *testing.T) {
	x, y := LonLatToMerc(0, 0)
	if math.Abs(x) > 1e-9 || math.Abs(y) > 1e-9 {
		t.Fatalf("null island merc %v %v", x, y)
	}
	if math.Abs(OriginShift-20037508.342789244) > 1e-6 {
		t.Fatalf("origin shift %v (go-geo MERC_BBOX)", OriginShift)
	}
}

func TestLocalProjectUnproject(t *testing.T) {
	o := Origin{Lon: -73.9857, Lat: 40.7484, Scale: 1}
	x, z := o.Project(-73.9857, 40.7484)
	if math.Hypot(x, z) > 1e-6 {
		t.Fatalf("origin should be 0,0 got %v %v", x, z)
	}
	// 1e-3 deg north is ~111 m on the sphere; Web Mercator stretches that near 40°N.
	x2, z2 := o.Project(-73.9857, 40.7494)
	if math.Abs(x2) > 5 || z2 < 100 || z2 > 180 {
		t.Fatalf("1e-3 deg north should be +Z ~110–150m mercator, got %v %v", x2, z2)
	}
	lon, lat := o.Unproject(x2, z2)
	if math.Abs(lon+73.9857) > 1e-8 || math.Abs(lat-40.7494) > 1e-8 {
		t.Fatalf("unproject %v %v", lon, lat)
	}
}

func TestTileXYZ(t *testing.T) {
	x, y, z := TileXYZ(0, 0, 0)
	if x != 0 || y != 0 || z != 0 {
		t.Fatalf("z0 %d %d %d", x, y, z)
	}
	x, y, z = TileXYZ(0, 0, 1)
	if z != 1 || x != 1 || y != 1 {
		t.Fatalf("null island z1 expected 1,1 got %d,%d", x, y)
	}
	// Equator/prime-meridian sits on a tile edge; floor() picks the SE tile.
	x, y, z = TileXYZ(0, 0, 2)
	if x != 2 || y != 2 || z != 2 {
		t.Fatalf("z2 %d %d %d", x, y, z)
	}
	if TMSY(0, 2) != 3 || TMSY(1, 2) != 2 {
		t.Fatalf("tms flip")
	}
}

func TestParseGeoJSON(t *testing.T) {
	o := Origin{Lon: 0, Lat: 0, Scale: 1}
	src := `{
	  "type":"FeatureCollection",
	  "features":[
	    {"type":"Feature","properties":{"name":"a"},"geometry":{"type":"Point","coordinates":[1,0]}},
	    {"type":"Feature","geometry":{"type":"LineString","coordinates":[[0,0],[0.001,0],[0.001,0.001]]}}
	  ]
	}`
	c, err := ParseGeoJSON([]byte(src), o)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Features) != 2 {
		t.Fatalf("features %d", len(c.Features))
	}
	if c.Features[0].Kind != "point" || c.Features[0].Name != "a" {
		t.Fatalf("point %+v", c.Features[0])
	}
	if c.Features[1].Kind != "line" || len(c.Features[1].Points) != 3 {
		t.Fatalf("line %+v", c.Features[1])
	}
	if c.Features[0].Points[0][0] <= 0 {
		t.Fatalf("point should be +X (east of origin)")
	}
}
