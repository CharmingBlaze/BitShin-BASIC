package runtime

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"moonbasic/internal/value"
)

// Tile cells hold a tile id (1-based; 0 is empty) plus Tiled's flip bits.
const (
	tileFlipH    = 1 << 31
	tileFlipV    = 1 << 30
	tileFlipD    = 1 << 29 // diagonal: swap x and y before the other flips
	tileFlagMask = tileFlipH | tileFlipV | tileFlipD
)

func tileID(cell int) int { return cell &^ tileFlagMask }

type tileLayer struct {
	name     string
	cells    []int
	visible  bool
	opacity  float64
	offX     float64
	offY     float64
	parallax float64 // 1 moves with the world; 0.5 is a far background
	autoBase int     // first of 16 autotile ids, 0 off
}

// tileSet is one atlas image. Tiles count from firstGID in reading order.
type tileSet struct {
	firstGID       int
	img            *ebiten.Image
	tw, th         int
	cols, count    int
	margin, spacng int
}

type tileAnim struct {
	frames []int     // tile ids
	times  []float64 // seconds each
	total  float64
}

type tileObject struct {
	name, kind string
	x, y, w, h float64
	props      map[string]value.Value
}

func (tm *tileMap) layer(i int) *tileLayer {
	if i < 0 || i >= len(tm.layers) {
		return nil
	}
	return tm.layers[i]
}

func (tm *tileMap) addLayer(name string) *tileLayer {
	l := &tileLayer{name: name, cells: make([]int, tm.cols*tm.rows), visible: true, opacity: 1, parallax: 1}
	tm.layers = append(tm.layers, l)
	if len(tm.layers) == 1 {
		tm.cells = l.cells
	}
	return l
}

func (tm *tileMap) layerByArg(a []value.Value, i int) (*tileLayer, error) {
	idx := argI(a, i, 0)
	l := tm.layer(idx)
	if l == nil {
		return nil, fmt.Errorf("layer %d of %d", idx, len(tm.layers))
	}
	return l, nil
}

// frameAt is the tile id an animated tile shows at time t.
func (tm *tileMap) frameAt(id int, t float64) int {
	an, ok := tm.anims[id]
	if !ok || an.total <= 0 {
		return id
	}
	t = math.Mod(t, an.total)
	for i, d := range an.times {
		if t < d {
			return an.frames[i]
		}
		t -= d
	}
	return an.frames[len(an.frames)-1]
}

// tileSource finds the atlas and rectangle for a tile id.
func (w *World) tileSource(tm *tileMap, id int) (*ebiten.Image, int, int, int, int) {
	if len(tm.sets) > 0 {
		var set *tileSet
		for _, s := range tm.sets {
			if id >= s.firstGID && (set == nil || s.firstGID > set.firstGID) {
				set = s
			}
		}
		if set == nil || set.img == nil || set.cols < 1 {
			return nil, 0, 0, 0, 0
		}
		local := id - set.firstGID
		x := set.margin + (local%set.cols)*(set.tw+set.spacng)
		y := set.margin + (local/set.cols)*(set.th+set.spacng)
		return set.img, x, y, set.tw, set.th
	}
	im := w.images[tm.atlas]
	if im == nil || im.img == nil {
		return nil, 0, 0, 0, 0
	}
	acols := tm.atlasCols
	if acols < 1 {
		acols = im.w / tm.tw
		if acols < 1 {
			acols = 1
		}
	}
	tile := id - 1
	return im.img, (tile % acols) * tm.tw, (tile / acols) * tm.th, tm.tw, tm.th
}

// drawTileLayer draws the cells camera c can see. Tiles taller than the
// grid sit on the cell's bottom edge, as in Tiled.
func (w *World) drawTileLayer(dst *ebiten.Image, c *camera2D, tm *tileMap, l *tileLayer, ox, oy float64) {
	if l == nil || !l.visible || l.opacity <= 0 {
		return
	}
	// Parallax: a layer at 0.5 slides half as fast as the camera. As in
	// Tiled, it is measured from the view's top-left corner.
	hw, hh := c.halfView(w)
	ox += l.offX + (c.cx-hw)*(1-l.parallax)
	oy += l.offY + (c.cy-hh)*(1-l.parallax)
	cm := c.geoM(w)
	x0, y0, x1, y1 := c.worldRect(w)
	cx0 := max(0, floorCell(x0-ox, tm.tw)-1)
	cy0 := max(0, floorCell(y0-oy, tm.th)-1)
	cx1 := min(tm.cols-1, floorCell(x1-ox, tm.tw)+1)
	cy1 := min(tm.rows-1, floorCell(y1-oy, tm.th)+1)
	for y := cy0; y <= cy1; y++ {
		for x := cx0; x <= cx1; x++ {
			cell := l.cells[y*tm.cols+x]
			id := tileID(cell)
			if id <= 0 {
				continue
			}
			id = tm.frameAt(id, w.tileClock)
			px, py := ox+float64(x*tm.tw), oy+float64(y*tm.th)
			img, sx, sy, sw, sh := w.tileSource(tm, id)
			opt := &ebiten.DrawImageOptions{}
			if img == nil {
				// No atlas: a coloured block per id, as before.
				opt.GeoM.Scale(float64(tm.tw-1), float64(tm.th-1))
				opt.GeoM.Translate(px, py)
				opt.GeoM.Concat(cm)
				opt.ColorScale.Scale(float32(40+(id*40)%200)/255, float32(80+(id*20)%160)/255, 120.0/255, 1)
				opt.ColorScale.ScaleAlpha(float32(l.opacity))
				dst.DrawImage(w.whitePixel(), opt)
				continue
			}
			src := img.SubImage(rectXYWH(sx, sy, sw, sh)).(*ebiten.Image)
			opt.GeoM.Translate(-float64(sw)/2, -float64(sh)/2)
			if cell&tileFlipD != 0 {
				var d ebiten.GeoM
				d.SetElement(0, 0, 0)
				d.SetElement(0, 1, 1)
				d.SetElement(1, 0, 1)
				d.SetElement(1, 1, 0)
				opt.GeoM.Concat(d)
			}
			fx, fy := 1.0, 1.0
			if cell&tileFlipH != 0 {
				fx = -1
			}
			if cell&tileFlipV != 0 {
				fy = -1
			}
			opt.GeoM.Scale(fx, fy)
			opt.GeoM.Translate(px+float64(sw)/2, py+float64(tm.th)-float64(sh)/2)
			opt.GeoM.Concat(cm)
			opt.ColorScale.ScaleAlpha(float32(l.opacity))
			dst.DrawImage(src, opt)
		}
	}
}

// Autotiling: 16 tiles from autoBase, chosen by which of the four
// neighbours are in the same set (north 1, east 2, south 4, west 8).
func (tm *tileMap) inAuto(l *tileLayer, x, y int) bool {
	if x < 0 || y < 0 || x >= tm.cols || y >= tm.rows {
		return false
	}
	id := tileID(l.cells[y*tm.cols+x])
	return id >= l.autoBase && id < l.autoBase+16
}

func (tm *tileMap) refreshAuto(l *tileLayer, x, y int) {
	if !tm.inAuto(l, x, y) {
		return
	}
	mask := 0
	for i, d := range [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} {
		if tm.inAuto(l, x+d[0], y+d[1]) {
			mask |= 1 << i
		}
	}
	l.cells[y*tm.cols+x] = l.autoBase + mask
}

func (tm *tileMap) paintAuto(l *tileLayer, x, y int, on bool) {
	if x < 0 || y < 0 || x >= tm.cols || y >= tm.rows || l.autoBase <= 0 {
		return
	}
	if on {
		l.cells[y*tm.cols+x] = l.autoBase
	} else {
		l.cells[y*tm.cols+x] = 0
	}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			tm.refreshAuto(l, x+dx, y+dy)
		}
	}
}

// --- Tiled import ---

type tiledProp struct {
	Name  string          `json:"name" xml:"name,attr"`
	Type  string          `json:"type" xml:"type,attr"`
	Value json.RawMessage `json:"value"`
	XVal  string          `xml:"value,attr"`
}

type tiledFrame struct {
	TileID   int `json:"tileid" xml:"tileid,attr"`
	Duration int `json:"duration" xml:"duration,attr"`
}

type tiledTile struct {
	ID         int          `json:"id" xml:"id,attr"`
	Properties []tiledProp  `json:"properties" xml:"properties>property"`
	Animation  []tiledFrame `json:"animation" xml:"animation>frame"`
}

type tiledImage struct {
	Source string `xml:"source,attr"`
}

type tiledSet struct {
	FirstGID   int         `json:"firstgid" xml:"firstgid,attr"`
	Source     string      `json:"source" xml:"source,attr"`
	TileWidth  int         `json:"tilewidth" xml:"tilewidth,attr"`
	TileHeight int         `json:"tileheight" xml:"tileheight,attr"`
	Spacing    int         `json:"spacing" xml:"spacing,attr"`
	Margin     int         `json:"margin" xml:"margin,attr"`
	Columns    int         `json:"columns" xml:"columns,attr"`
	TileCount  int         `json:"tilecount" xml:"tilecount,attr"`
	Image      string      `json:"image"`
	XImage     tiledImage  `xml:"image"`
	Tiles      []tiledTile `json:"tiles" xml:"tile"`
}

type tiledData struct {
	Encoding    string       `xml:"encoding,attr"`
	Compression string       `xml:"compression,attr"`
	Text        string       `xml:",chardata"`
	Chunks      []tiledChunk `xml:"chunk"`
}

type tiledChunk struct {
	X      int             `json:"x" xml:"x,attr"`
	Y      int             `json:"y" xml:"y,attr"`
	Width  int             `json:"width" xml:"width,attr"`
	Height int             `json:"height" xml:"height,attr"`
	Data   json.RawMessage `json:"data"`
	XText  string          `xml:",chardata"`
}

type tiledObject struct {
	Name       string      `json:"name" xml:"name,attr"`
	Type       string      `json:"type" xml:"type,attr"`
	Class      string      `json:"class" xml:"class,attr"`
	X          float64     `json:"x" xml:"x,attr"`
	Y          float64     `json:"y" xml:"y,attr"`
	Width      float64     `json:"width" xml:"width,attr"`
	Height     float64     `json:"height" xml:"height,attr"`
	Properties []tiledProp `json:"properties" xml:"properties>property"`
}

type tiledLayer struct {
	XMLName     xml.Name
	Type        string          `json:"type"`
	Name        string          `json:"name" xml:"name,attr"`
	Visible     *bool           `json:"visible"`
	XVisible    string          `xml:"visible,attr"`
	Opacity     *float64        `json:"opacity" xml:"opacity,attr"`
	OffsetX     float64         `json:"offsetx" xml:"offsetx,attr"`
	OffsetY     float64         `json:"offsety" xml:"offsety,attr"`
	ParallaxX   *float64        `json:"parallaxx" xml:"parallaxx,attr"`
	Encoding    string          `json:"encoding"`
	Compression string          `json:"compression"`
	Data        json.RawMessage `json:"data"`
	XData       tiledData       `xml:"data"`
	Chunks      []tiledChunk    `json:"chunks"`
	Objects     []tiledObject   `json:"objects" xml:"object"`
	Layers      []tiledLayer    `json:"layers" xml:",any"`
}

type tiledMap struct {
	Width      int          `json:"width" xml:"width,attr"`
	Height     int          `json:"height" xml:"height,attr"`
	TileWidth  int          `json:"tilewidth" xml:"tilewidth,attr"`
	TileHeight int          `json:"tileheight" xml:"tileheight,attr"`
	Infinite   bool         `json:"infinite"`
	XInfinite  int          `xml:"infinite,attr"`
	TileSets   []tiledSet   `json:"tilesets" xml:"tileset"`
	Layers     []tiledLayer `json:"layers" xml:",any"`
}

func (w *World) loadEbitenImage(path string) (*ebiten.Image, error) {
	img, _, err := ebitenutil.NewImageFromFile(path)
	return img, err
}

func isXMLPath(p string) bool {
	ext := strings.ToLower(filepath.Ext(p))
	return ext == ".tmx" || ext == ".tsx" || ext == ".xml"
}

func (w *World) loadTiled(path string) (*tileMap, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tmx tiledMap
	if isXMLPath(path) {
		err = xml.Unmarshal(raw, &tmx)
		tmx.Infinite = tmx.XInfinite != 0
	} else {
		err = json.Unmarshal(raw, &tmx)
	}
	if err != nil {
		return nil, err
	}
	if tmx.TileWidth < 1 || tmx.TileHeight < 1 {
		return nil, fmt.Errorf("no tile size")
	}
	dir := filepath.Dir(path)
	tm := &tileMap{tw: tmx.TileWidth, th: tmx.TileHeight, anims: map[int]tileAnim{}, data: map[int]map[string]value.Value{}}

	for _, ts := range tmx.TileSets {
		if ts.Source != "" {
			first := ts.FirstGID
			src := filepath.Join(dir, ts.Source)
			b, err := os.ReadFile(src)
			if err != nil {
				return nil, fmt.Errorf("tileset %s: %w", ts.Source, err)
			}
			if isXMLPath(src) {
				err = xml.Unmarshal(b, &ts)
			} else {
				err = json.Unmarshal(b, &ts)
			}
			if err != nil {
				return nil, fmt.Errorf("tileset %s: %w", ts.Source, err)
			}
			ts.FirstGID = first
			dir := filepath.Dir(src)
			if err := w.addTiledSet(tm, ts, dir); err != nil {
				return nil, err
			}
			continue
		}
		if err := w.addTiledSet(tm, ts, dir); err != nil {
			return nil, err
		}
	}

	// Infinite maps store chunks; find their bounds to size the grid.
	minX, minY := 0, 0
	tm.cols, tm.rows = tmx.Width, tmx.Height
	if tmx.Infinite {
		x0, y0, x1, y1 := math.MaxInt32, math.MaxInt32, math.MinInt32, math.MinInt32
		walkTiled(tmx.Layers, func(l tiledLayer) {
			for _, c := range append(l.Chunks, l.XData.Chunks...) {
				x0, y0 = min(x0, c.X), min(y0, c.Y)
				x1, y1 = max(x1, c.X+c.Width), max(y1, c.Y+c.Height)
			}
		})
		if x1 > x0 {
			minX, minY, tm.cols, tm.rows = x0, y0, x1-x0, y1-y0
		}
	}
	if tm.cols < 1 || tm.rows < 1 {
		return nil, fmt.Errorf("empty map")
	}

	var ferr error
	walkTiled(tmx.Layers, func(l tiledLayer) {
		if ferr != nil {
			return
		}
		kind := l.Type
		if kind == "" {
			kind = map[string]string{"layer": "tilelayer", "objectgroup": "objectgroup"}[l.XMLName.Local]
		}
		switch kind {
		case "tilelayer":
			tl := tm.addLayer(l.Name)
			tl.offX, tl.offY = l.OffsetX, l.OffsetY
			if l.Opacity != nil {
				tl.opacity = *l.Opacity
			}
			if l.Visible != nil {
				tl.visible = *l.Visible
			}
			if l.XVisible == "0" {
				tl.visible = false
			}
			if l.ParallaxX != nil {
				tl.parallax = *l.ParallaxX
			}
			put := func(x, y, gid int) {
				x, y = x-minX, y-minY
				if x >= 0 && y >= 0 && x < tm.cols && y < tm.rows {
					tl.cells[y*tm.cols+x] = gid
				}
			}
			if chunks := append(l.Chunks, l.XData.Chunks...); len(chunks) > 0 {
				enc, comp := l.Encoding, l.Compression
				if l.XData.Encoding != "" {
					enc, comp = l.XData.Encoding, l.XData.Compression
				}
				for _, c := range chunks {
					gids, err := decodeTiled(c.Data, c.XText, enc, comp, c.Width*c.Height)
					if err != nil {
						ferr = fmt.Errorf("layer %q: %w", l.Name, err)
						return
					}
					for i, g := range gids {
						put(c.X+i%c.Width, c.Y+i/c.Width, g)
					}
				}
				return
			}
			enc, comp := l.Encoding, l.Compression
			if l.XData.Encoding != "" || l.XData.Text != "" {
				enc, comp = l.XData.Encoding, l.XData.Compression
			}
			gids, err := decodeTiled(l.Data, l.XData.Text, enc, comp, tmx.Width*tmx.Height)
			if err != nil {
				ferr = fmt.Errorf("layer %q: %w", l.Name, err)
				return
			}
			for i, g := range gids {
				put(i%tmx.Width, i/tmx.Width, g)
			}
		case "objectgroup":
			for _, o := range l.Objects {
				kind := o.Type
				if kind == "" {
					kind = o.Class
				}
				tm.objects = append(tm.objects, tileObject{
					name: o.Name, kind: kind,
					x: o.X - float64(minX*tm.tw), y: o.Y - float64(minY*tm.th), w: o.Width, h: o.Height,
					props: tiledProps(o.Properties),
				})
			}
		}
	})
	if ferr != nil {
		return nil, ferr
	}
	if len(tm.layers) == 0 {
		tm.addLayer("")
	}
	return tm, nil
}

// walkTiled visits layers depth first, inside groups too.
func walkTiled(ls []tiledLayer, fn func(tiledLayer)) {
	for _, l := range ls {
		if l.Type == "group" || l.XMLName.Local == "group" {
			walkTiled(l.Layers, fn)
			continue
		}
		fn(l)
	}
}

func (w *World) addTiledSet(tm *tileMap, ts tiledSet, dir string) error {
	imgPath := ts.Image
	if imgPath == "" {
		imgPath = ts.XImage.Source
	}
	if imgPath == "" {
		return fmt.Errorf("tileset at gid %d has no single image (image collections are not supported)", ts.FirstGID)
	}
	full := filepath.Join(dir, imgPath)
	img, err := w.loadEbitenImage(full)
	if err != nil {
		return fmt.Errorf("tileset image %s: %w", imgPath, err)
	}
	set := &tileSet{firstGID: ts.FirstGID, img: img, tw: ts.TileWidth, th: ts.TileHeight, cols: ts.Columns, count: ts.TileCount, margin: ts.Margin, spacng: ts.Spacing}
	if set.tw < 1 {
		set.tw = tm.tw
	}
	if set.th < 1 {
		set.th = tm.th
	}
	if set.cols < 1 {
		set.cols = max(1, (img.Bounds().Dx()-2*set.margin+set.spacng)/(set.tw+set.spacng))
	}
	tm.sets = append(tm.sets, set)
	for _, t := range ts.Tiles {
		gid := ts.FirstGID + t.ID
		if props := tiledProps(t.Properties); len(props) > 0 {
			tm.data[gid] = props
		}
		if len(t.Animation) > 0 {
			var an tileAnim
			for _, f := range t.Animation {
				d := math.Max(float64(f.Duration)/1000, 0.001)
				an.frames = append(an.frames, ts.FirstGID+f.TileID)
				an.times = append(an.times, d)
				an.total += d
			}
			tm.anims[gid] = an
		}
	}
	return nil
}

func tiledProps(ps []tiledProp) map[string]value.Value {
	if len(ps) == 0 {
		return nil
	}
	out := map[string]value.Value{}
	for _, p := range ps {
		s := p.XVal
		if len(p.Value) > 0 {
			var v any
			if json.Unmarshal(p.Value, &v) == nil {
				s = fmt.Sprint(v)
			}
		}
		switch p.Type {
		case "int", "float":
			f, _ := strconv.ParseFloat(s, 64)
			out[strings.ToLower(p.Name)] = value.Num(f)
		case "bool":
			out[strings.ToLower(p.Name)] = value.Num(float64(b01(s == "true")))
		default:
			out[strings.ToLower(p.Name)] = value.Str(s)
		}
	}
	return out
}

// decodeTiled reads a layer's gids from a JSON array, CSV text, or base64
// little-endian uint32s, optionally zlib- or gzip-compressed.
func decodeTiled(jsonData json.RawMessage, text, enc, comp string, n int) ([]int, error) {
	if len(jsonData) > 0 && jsonData[0] == '[' {
		var arr []uint32
		if err := json.Unmarshal(jsonData, &arr); err != nil {
			return nil, err
		}
		return u32s(arr), nil
	}
	if len(jsonData) > 0 {
		if err := json.Unmarshal(jsonData, &text); err != nil {
			return nil, err
		}
		if enc == "" {
			enc = "base64"
		}
	}
	text = strings.TrimSpace(text)
	switch enc {
	case "csv", "":
		var out []int
		for _, f := range strings.FieldsFunc(text, func(r rune) bool { return r == ',' || r == '\n' || r == '\r' || r == ' ' || r == '\t' }) {
			v, err := strconv.ParseUint(f, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("csv: %w", err)
			}
			out = append(out, int(uint32(v)))
		}
		return out, nil
	case "base64":
		b, err := base64.StdEncoding.DecodeString(text)
		if err != nil {
			return nil, err
		}
		var r io.Reader = bytes.NewReader(b)
		switch comp {
		case "zlib":
			if r, err = zlib.NewReader(r); err != nil {
				return nil, err
			}
		case "gzip":
			if r, err = gzip.NewReader(r); err != nil {
				return nil, err
			}
		case "":
		default:
			return nil, fmt.Errorf("%s compression is not supported (use zlib, gzip, or none)", comp)
		}
		if b, err = io.ReadAll(r); err != nil {
			return nil, err
		}
		arr := make([]uint32, len(b)/4)
		if err := binary.Read(bytes.NewReader(b), binary.LittleEndian, arr); err != nil {
			return nil, err
		}
		return u32s(arr), nil
	}
	return nil, fmt.Errorf("unknown encoding %q", enc)
}

func u32s(a []uint32) []int {
	out := make([]int, len(a))
	for i, v := range a {
		out[i] = int(v)
	}
	return out
}

func (w *World) tileMap2DCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	get := func(a []value.Value, where string) (*tileMap, error) {
		tm := w.tiles[argI(a, 0, 0)]
		if tm == nil {
			return nil, fmt.Errorf("%s: %d is not a tilemap", where, argI(a, 0, 0))
		}
		return tm, nil
	}
	set := func(where string, fn func(tm *tileMap, a []value.Value) error) cmd {
		return n(func(a []value.Value) (value.Value, error) {
			tm, err := get(a, where)
			if err != nil {
				return value.Value{}, err
			}
			if err := fn(tm, a); err != nil {
				return value.Value{}, fmt.Errorf("%s: %w", where, err)
			}
			return z()
		})
	}
	read := func(where string, fn func(tm *tileMap, a []value.Value) (value.Value, error)) cmd {
		return n(func(a []value.Value) (value.Value, error) {
			tm, err := get(a, where)
			if err != nil {
				return value.Value{}, err
			}
			v, err := fn(tm, a)
			if err != nil {
				return value.Value{}, fmt.Errorf("%s: %w", where, err)
			}
			return v, nil
		})
	}
	layerArg := func(tm *tileMap, a []value.Value, i int) (*tileLayer, error) { return tm.layerByArg(a, i) }
	object := func(tm *tileMap, a []value.Value) (*tileObject, error) {
		i := argI(a, 1, 0)
		if i < 0 || i >= len(tm.objects) {
			return nil, fmt.Errorf("object %d of %d", i, len(tm.objects))
		}
		return &tm.objects[i], nil
	}
	return map[string]cmd{
		// LoadTiledMap(path$): a Tiled .tmx or .tmj/.json map with its
		// tilesets, layers, flips, animations, properties, and objects.
		"loadtiledmap": n(func(a []value.Value) (value.Value, error) {
			path, err := w.openPath(argS(a, 0))
			if err != nil {
				return value.Value{}, fmt.Errorf("LoadTiledMap: %w", err)
			}
			tm, err := w.loadTiled(path)
			if err != nil {
				return value.Value{}, fmt.Errorf("LoadTiledMap: %s: %w", filepath.Base(path), err)
			}
			id := w.nextTile
			w.nextTile++
			w.tiles[id] = tm
			return value.Num(float64(id)), nil
		}),
		// AddTileLayer(map [, name$]) returns the new layer's index.
		"addtilelayer": read("AddTileLayer", func(tm *tileMap, a []value.Value) (value.Value, error) {
			tm.addLayer(argS(a, 1))
			return value.Num(float64(len(tm.layers) - 1)), nil
		}),
		"tilelayercount": read("TileLayerCount", func(tm *tileMap, _ []value.Value) (value.Value, error) {
			return value.Num(float64(len(tm.layers))), nil
		}),
		// TileLayerIndex(map, name$): -1 when there is no such layer.
		"tilelayerindex": read("TileLayerIndex", func(tm *tileMap, a []value.Value) (value.Value, error) {
			for i, l := range tm.layers {
				if strings.EqualFold(l.name, argS(a, 1)) {
					return value.Num(float64(i)), nil
				}
			}
			return value.Num(-1), nil
		}),
		"settilelayervisible": set("SetTileLayerVisible", func(tm *tileMap, a []value.Value) error {
			l, err := layerArg(tm, a, 1)
			if err == nil {
				l.visible = argI(a, 2, 1) != 0
			}
			return err
		}),
		"settilelayeralpha": set("SetTileLayerAlpha", func(tm *tileMap, a []value.Value) error {
			l, err := layerArg(tm, a, 1)
			if err == nil {
				l.opacity = clamp01(argN(a, 2, 1))
			}
			return err
		}),
		// SetTileLayerParallax map, layer, factor: 1 moves with the world,
		// 0 stays on screen, 0.5 is a distant background.
		"settilelayerparallax": set("SetTileLayerParallax", func(tm *tileMap, a []value.Value) error {
			l, err := layerArg(tm, a, 1)
			if err == nil {
				l.parallax = argN(a, 2, 1)
			}
			return err
		}),
		"settilelayeroffset": set("SetTileLayerOffset", func(tm *tileMap, a []value.Value) error {
			l, err := layerArg(tm, a, 1)
			if err == nil {
				l.offX, l.offY = argN(a, 2, 0), argN(a, 3, 0)
			}
			return err
		}),
		// SetTileCollisionLayer map, layer: the layer SpriteHitTile and
		// TileAt read.
		"settilecollisionlayer": set("SetTileCollisionLayer", func(tm *tileMap, a []value.Value) error {
			if _, err := layerArg(tm, a, 1); err != nil {
				return err
			}
			tm.collide = argI(a, 1, 0)
			return nil
		}),
		// SetTileFlip map, x, y, flipX, flipY [, diagonal, layer]
		"settileflip": set("SetTileFlip", func(tm *tileMap, a []value.Value) error {
			l, err := layerArg(tm, a, 6)
			if err != nil {
				return err
			}
			x, y := argI(a, 1, 0), argI(a, 2, 0)
			if x < 0 || y < 0 || x >= tm.cols || y >= tm.rows {
				return fmt.Errorf("%d,%d out of map", x, y)
			}
			c := tileID(l.cells[y*tm.cols+x])
			if argI(a, 3, 0) != 0 {
				c |= tileFlipH
			}
			if argI(a, 4, 0) != 0 {
				c |= tileFlipV
			}
			if argI(a, 5, 0) != 0 {
				c |= tileFlipD
			}
			l.cells[y*tm.cols+x] = c
			return nil
		}),
		// SetTileAnim map, tile, frames, fps: tile cycles through tile,
		// tile+1, ... tile+frames-1.
		"settileanim": set("SetTileAnim", func(tm *tileMap, a []value.Value) error {
			id, frames, fps := argI(a, 1, 0), argI(a, 2, 1), argN(a, 3, 8)
			if tm.anims == nil {
				tm.anims = map[int]tileAnim{}
			}
			if frames <= 1 || fps <= 0 {
				delete(tm.anims, id)
				return nil
			}
			an := tileAnim{}
			for i := 0; i < frames; i++ {
				an.frames = append(an.frames, id+i)
				an.times = append(an.times, 1/fps)
			}
			an.total = float64(frames) / fps
			tm.anims[id] = an
			return nil
		}),
		// SetTileData map, tile, key$, value / TileData(map, tile, key$):
		// properties per tile id. Tiled maps load theirs.
		"settiledata": set("SetTileData", func(tm *tileMap, a []value.Value) error {
			if tm.data == nil {
				tm.data = map[int]map[string]value.Value{}
			}
			id := argI(a, 1, 0)
			if tm.data[id] == nil {
				tm.data[id] = map[string]value.Value{}
			}
			if len(a) > 3 {
				tm.data[id][strings.ToLower(argS(a, 2))] = a[3]
			}
			return nil
		}),
		"tiledata": read("TileData", func(tm *tileMap, a []value.Value) (value.Value, error) {
			if v, ok := tm.data[argI(a, 1, 0)][strings.ToLower(argS(a, 2))]; ok {
				return v, nil
			}
			return value.Num(0), nil
		}),
		// SetAutoTile map, firstTile [, layer]: the 16 tiles from firstTile
		// form one autotile set. PaintAutoTile picks the right one.
		"setautotile": set("SetAutoTile", func(tm *tileMap, a []value.Value) error {
			l, err := layerArg(tm, a, 2)
			if err == nil {
				l.autoBase = argI(a, 1, 0)
			}
			return err
		}),
		// PaintAutoTile map, x, y [, on, layer]
		"paintautotile": set("PaintAutoTile", func(tm *tileMap, a []value.Value) error {
			l, err := layerArg(tm, a, 4)
			if err != nil {
				return err
			}
			if l.autoBase <= 0 {
				return fmt.Errorf("call SetAutoTile first")
			}
			tm.paintAuto(l, argI(a, 1, 0), argI(a, 2, 0), argI(a, 3, 1) != 0)
			return nil
		}),
		"tilemapwidth": read("TileMapWidth", func(tm *tileMap, _ []value.Value) (value.Value, error) {
			return value.Num(float64(tm.cols)), nil
		}),
		"tilemapheight": read("TileMapHeight", func(tm *tileMap, _ []value.Value) (value.Value, error) {
			return value.Num(float64(tm.rows)), nil
		}),
		"tilewidth": read("TileWidth", func(tm *tileMap, _ []value.Value) (value.Value, error) {
			return value.Num(float64(tm.tw)), nil
		}),
		"tileheight": read("TileHeight", func(tm *tileMap, _ []value.Value) (value.Value, error) {
			return value.Num(float64(tm.th)), nil
		}),
		// Object layers from Tiled: spawn points, triggers, areas.
		"tileobjectcount": read("TileObjectCount", func(tm *tileMap, _ []value.Value) (value.Value, error) {
			return value.Num(float64(len(tm.objects))), nil
		}),
		"tileobjectname": read("TileObjectName$", func(tm *tileMap, a []value.Value) (value.Value, error) {
			o, err := object(tm, a)
			if err != nil {
				return value.Value{}, err
			}
			return value.Str(o.name), nil
		}),
		"tileobjecttype": read("TileObjectType$", func(tm *tileMap, a []value.Value) (value.Value, error) {
			o, err := object(tm, a)
			if err != nil {
				return value.Value{}, err
			}
			return value.Str(o.kind), nil
		}),
		"tileobjectx": read("TileObjectX", func(tm *tileMap, a []value.Value) (value.Value, error) {
			o, err := object(tm, a)
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(o.x), nil
		}),
		"tileobjecty": read("TileObjectY", func(tm *tileMap, a []value.Value) (value.Value, error) {
			o, err := object(tm, a)
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(o.y), nil
		}),
		"tileobjectwidth": read("TileObjectWidth", func(tm *tileMap, a []value.Value) (value.Value, error) {
			o, err := object(tm, a)
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(o.w), nil
		}),
		"tileobjectheight": read("TileObjectHeight", func(tm *tileMap, a []value.Value) (value.Value, error) {
			o, err := object(tm, a)
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(o.h), nil
		}),
		"tileobjectproperty": read("TileObjectProperty", func(tm *tileMap, a []value.Value) (value.Value, error) {
			o, err := object(tm, a)
			if err != nil {
				return value.Value{}, err
			}
			if v, ok := o.props[strings.ToLower(argS(a, 2))]; ok {
				return v, nil
			}
			return value.Num(0), nil
		}),
	}
}
