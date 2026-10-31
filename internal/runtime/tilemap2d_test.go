package runtime

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"moonbasic/internal/value"
)

func TestTileLayersFlipsAndCollision(t *testing.T) {
	w, cmds := newWorld2D(t)
	m := call(t, cmds, "createtilemap", 16, 16, 4, 3).Num
	if l := call(t, cmds, "addtilelayer", m).Num; l != 1 {
		t.Fatalf("second layer index %v", l)
	}
	call(t, cmds, "settile", m, 1, 1, 5)    // layer 0
	call(t, cmds, "settile", m, 2, 1, 7, 1) // layer 1
	call(t, cmds, "settileflip", m, 1, 1, 1, 0, 1)
	if v := call(t, cmds, "gettile", m, 1, 1).Num; v != 5 {
		t.Fatalf("GetTile ignores flip bits: %v", v)
	}
	if v := call(t, cmds, "gettile", m, 2, 1, 1).Num; v != 7 {
		t.Fatalf("GetTile layer 1: %v", v)
	}
	tm := w.tiles[int(m)]
	if c := tm.layers[0].cells[1*4+1]; c&tileFlipH == 0 || c&tileFlipD == 0 || c&tileFlipV != 0 {
		t.Fatalf("flip bits %x", c)
	}
	// Collision follows the chosen layer.
	if v := call(t, cmds, "tileat", m, 40, 20).Num; v != 0 {
		t.Fatalf("layer 0 at 2,1 = %v", v)
	}
	call(t, cmds, "settilecollisionlayer", m, 1)
	if v := call(t, cmds, "tileat", m, 40, 20).Num; v != 7 {
		t.Fatalf("collision layer 1 at 2,1 = %v", v)
	}
	if _, err := cmds["settile"]([]value.Value{value.Num(m), value.Num(0), value.Num(0), value.Num(1), value.Num(9)}); err == nil {
		t.Error("SetTile accepted layer 9")
	}
}

func TestAutoTileMasks(t *testing.T) {
	_, cmds := newWorld2D(t)
	m := call(t, cmds, "createtilemap", 16, 16, 5, 5).Num
	call(t, cmds, "setautotile", m, 100)
	// A plus sign: the centre touches all four, the arms touch only it.
	for _, p := range [][2]float64{{2, 2}, {2, 1}, {3, 2}, {2, 3}, {1, 2}} {
		call(t, cmds, "paintautotile", m, p[0], p[1])
	}
	want := map[[2]float64]float64{{2, 2}: 100 + 15, {2, 1}: 100 + 4, {3, 2}: 100 + 8, {2, 3}: 100 + 1, {1, 2}: 100 + 2}
	for p, id := range want {
		if v := call(t, cmds, "gettile", m, p[0], p[1]).Num; v != id {
			t.Errorf("auto tile at %v = %v, want %v", p, v, id)
		}
	}
	call(t, cmds, "paintautotile", m, 2, 2, 0) // erase the centre: arms lose a neighbour
	if v := call(t, cmds, "gettile", m, 2, 1).Num; v != 100 {
		t.Errorf("north arm after erase = %v", v)
	}
}

func TestTileAnimFrames(t *testing.T) {
	w, cmds := newWorld2D(t)
	m := call(t, cmds, "createtilemap", 16, 16, 2, 2).Num
	call(t, cmds, "settileanim", m, 10, 3, 4) // 10, 11, 12 at 4 fps
	tm := w.tiles[int(m)]
	for _, c := range []struct {
		t    float64
		want int
	}{{0, 10}, {0.26, 11}, {0.6, 12}, {0.76, 10}} {
		if got := tm.frameAt(10, c.t); got != c.want {
			t.Errorf("frame at %v = %d, want %d", c.t, got, c.want)
		}
	}
}

func writePNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = 200
	}
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func b64(gids []uint32, comp string) string {
	var raw bytes.Buffer
	binary.Write(&raw, binary.LittleEndian, gids)
	var out bytes.Buffer
	switch comp {
	case "zlib":
		z := zlib.NewWriter(&out)
		z.Write(raw.Bytes())
		z.Close()
	case "gzip":
		z := gzip.NewWriter(&out)
		z.Write(raw.Bytes())
		z.Close()
	default:
		out = raw
	}
	return base64.StdEncoding.EncodeToString(out.Bytes())
}

func TestLoadTiledJSON(t *testing.T) {
	dir := t.TempDir()
	writePNG(t, filepath.Join(dir, "tiles.png"), 64, 32)
	data := b64([]uint32{1, 2, 0, tileFlipH | 3, 0, 4}, "zlib")
	js := `{"width":3,"height":2,"tilewidth":16,"tileheight":16,
	 "tilesets":[{"firstgid":1,"image":"tiles.png","tilewidth":16,"tileheight":16,"columns":4,"tilecount":8,
	   "tiles":[{"id":1,"properties":[{"name":"Solid","type":"bool","value":true},{"name":"cost","type":"int","value":3}]},
	            {"id":0,"animation":[{"tileid":0,"duration":100},{"tileid":1,"duration":100}]}]}],
	 "layers":[{"type":"group","layers":[
	   {"type":"tilelayer","name":"Ground","width":3,"height":2,"encoding":"base64","compression":"zlib","data":"` + data + `"}]},
	   {"type":"tilelayer","name":"Deco","width":3,"height":2,"opacity":0.5,"visible":false,"data":[0,0,5,0,0,0]},
	   {"type":"objectgroup","name":"Spawns","objects":[{"name":"player","type":"spawn","x":24,"y":8,"width":0,"height":0,
	     "properties":[{"name":"facing","type":"string","value":"left"}]}]}]}`
	p := filepath.Join(dir, "level.tmj")
	os.WriteFile(p, []byte(js), 0o644)

	w, cmds := newWorld2D(t)
	v, err := cmds["loadtiledmap"]([]value.Value{value.Str(p)})
	if err != nil {
		t.Fatal(err)
	}
	tm := w.tiles[v.Int()]
	if tm.cols != 3 || tm.rows != 2 || len(tm.layers) != 2 {
		t.Fatalf("map %dx%d layers %d", tm.cols, tm.rows, len(tm.layers))
	}
	if c := tm.layers[0].cells[3]; tileID(c) != 3 || c&tileFlipH == 0 {
		t.Fatalf("flipped cell %x", c)
	}
	if l := tm.layers[1]; l.visible || l.opacity != 0.5 || tileID(l.cells[2]) != 5 {
		t.Fatalf("deco layer %+v", l)
	}
	if got, _ := cmds["tilelayerindex"]([]value.Value{v, value.Str("deco")}); got.Num != 1 {
		t.Fatalf("TileLayerIndex deco = %v", got.Num)
	}
	if tm.data[2]["solid"].Num != 1 || tm.data[2]["cost"].Num != 3 {
		t.Fatalf("tile properties %v", tm.data[2])
	}
	if tm.frameAt(1, 0.15) != 2 {
		t.Fatal("tile animation not loaded")
	}
	if len(tm.objects) != 1 || tm.objects[0].name != "player" || tm.objects[0].x != 24 || tm.objects[0].props["facing"].String() != "left" {
		t.Fatalf("objects %+v", tm.objects)
	}
	if img, sx, sy, _, _ := w.tileSource(tm, 6); img == nil || sx != 16 || sy != 16 {
		t.Fatalf("tile 6 source %v %d,%d", img != nil, sx, sy)
	}
}

func TestLoadTiledTMXInfiniteExternalSet(t *testing.T) {
	dir := t.TempDir()
	writePNG(t, filepath.Join(dir, "set.png"), 32, 32)
	tsx := `<?xml version="1.0"?><tileset name="s" tilewidth="16" tileheight="16" columns="2" tilecount="4">
	 <image source="set.png" width="32" height="32"/></tileset>`
	os.WriteFile(filepath.Join(dir, "set.tsx"), []byte(tsx), 0o644)
	chunk := b64([]uint32{1, 2, 3, 4}, "gzip")
	tmx := `<?xml version="1.0"?>
<map version="1.10" orientation="orthogonal" width="2" height="2" tilewidth="16" tileheight="16" infinite="1">
 <tileset firstgid="1" source="set.tsx"/>
 <layer id="1" name="Main" width="2" height="2">
  <data encoding="base64" compression="gzip">
   <chunk x="-2" y="0" width="2" height="2">` + chunk + `</chunk>
  </data>
 </layer>
 <layer id="2" name="Csv" width="2" height="2"><data encoding="csv">
  <chunk x="0" y="0" width="2" height="2">1,0,
0,2</chunk></data></layer>
 <objectgroup id="3" name="Things"><object id="1" name="door" class="exit" x="8" y="16" width="16" height="32"/></objectgroup>
</map>`
	p := filepath.Join(dir, "world.tmx")
	os.WriteFile(p, []byte(tmx), 0o644)
	w, cmds := newWorld2D(t)
	v, err := cmds["loadtiledmap"]([]value.Value{value.Str(p)})
	if err != nil {
		t.Fatal(err)
	}
	tm := w.tiles[v.Int()]
	// Chunks span x -2..2: four columns, shifted so the left edge is 0.
	if tm.cols != 4 || tm.rows != 2 {
		t.Fatalf("infinite bounds %dx%d", tm.cols, tm.rows)
	}
	main := tm.layers[0].cells
	if tileID(main[0]) != 1 || tileID(main[1]) != 2 || tileID(main[4]) != 3 || tileID(main[5]) != 4 {
		t.Fatalf("main layer %v", main)
	}
	csv := tm.layers[1].cells
	if tileID(csv[2]) != 1 || tileID(csv[7]) != 2 {
		t.Fatalf("csv layer %v", csv)
	}
	if len(tm.objects) != 1 || tm.objects[0].kind != "exit" || tm.objects[0].x != 8+32 {
		t.Fatalf("object %+v", tm.objects)
	}
	if _, err := cmds["loadtiledmap"]([]value.Value{value.Str(filepath.Join(dir, "missing.tmx"))}); err == nil || !strings.Contains(err.Error(), "LoadTiledMap") {
		t.Fatalf("missing file error %v", err)
	}
}
