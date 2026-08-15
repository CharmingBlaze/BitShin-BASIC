# Geo

WGS84 helpers for streaming worlds. **OpenGL 3.3.** No FlatBuffers.

## Why not `github.com/flywave/go-geo`?

[flywave/go-geo](https://github.com/flywave/go-geo) is a tile-proxy library: SRS/PROJ transforms, GEOS coverages, TMS/XYZ grids, GCJ-02, geoid, geometry reprojection. Its `go.mod` `replace`s sibling CGO trees (`go-proj`, `go-geos`, `go-geoid`) and needs native **PROJ** + **GEOS**. That does not build cleanly on Windows with this module.

BitShin BASIC ships a **thin compatible subset** in `internal/geo` (pure Go):

| go-geo | What we kept |
| --- | --- |
| EPSG:3857 / Web Mercator (`MERC_BBOX` = ±20037508.342789244) | `LonLatToMerc` / `MercToLonLat` |
| Local metres around a map origin | `SetGeoOrigin` + `GeoProject` / `GeoUnproject` (+X east, +Z north) |
| XYZ (OSM/Google) + TMS Y flip | `GeoTileX/Y/Z`, `GeoTMSY` |
| Geometry reproject | `LoadGeoJSON` (Point / LineString / Polygon) |
| DEM / quantized-mesh / GEOS clip | **Not ported.** Use a PNG or `GenerateHeightmap` + `LoadGeoDEM` / `CreateTerrainFromGeoDEM` |

Upstream has **no LICENSE file** in-tree. We did not vendor go-geo. `internal/geo` is original code under this repo’s MIT license.

## Commands

See [COMMANDS.md](COMMANDS.md) (Geo). Typical setup:

```basic
SetGeoOrigin(-73.9857, 40.7484)
x# = GeoProject(lon, lat)
z# = GeoProjectZ()
t = LoadGeoDEM("dem.png", west, south, east, north, 12)
path = LoadGeoJSON("route.geojson")
```

`SetGeoOrigin` is the streaming-world pin: world XZ `(0,0)` is that lon/lat. If a world stream exists, its origin is reset to `(0,0)` so chunks stay around the geo pin. Move the player with `GeoProject`, then `SetStreamOrigin` as usual.

`LoadGeoDEM` / `CreateTerrainFromGeoDEM` only set the heightfield’s world bounds (`ox/oz/worldW/worldD`) and flip image rows so the top of the PNG is north. They call existing `addTerrain` — they do **not** replace `CreateTerrain`, Terrain-OpenGL splat, or `heightmap_gen.go`.

`GeoHeight(lon, lat)` is `TerrainHeight` at the projected XZ.

Demo: `examples/geo.bb`.
