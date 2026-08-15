// Generates tiny original CC0 textures and a coin beep for examples/platform64.bb.
package main

import (
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

func main() {
	dir := filepath.Join("examples", "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	writeChecker(filepath.Join(dir, "grass.png"), color.RGBA{46, 130, 70, 255}, color.RGBA{36, 100, 52, 255})
	writeChecker(filepath.Join(dir, "stone.png"), color.RGBA{186, 150, 96, 255}, color.RGBA{140, 108, 70, 255})
	writeCoin(filepath.Join(dir, "coin.png"))
	writeBeep(filepath.Join(dir, "coin.wav"))
}

func writeChecker(path string, a, b color.RGBA) {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			c := a
			if (x/8+y/8)%2 == 1 {
				c = b
			}
			img.Set(x, y, c)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func writeCoin(path string) {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	cx, cy, r := 15.5, 15.5, 13.0
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			d := math.Sqrt(dx*dx + dy*dy)
			if d <= r {
				t := d / r
				img.Set(x, y, color.RGBA{255, uint8(210 - t*40), 40, 255})
			} else {
				img.Set(x, y, color.RGBA{180, 140, 20, 255})
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func writeBeep(path string) {
	const rate = 44100
	n := rate / 8 // 125 ms
	data := make([]byte, 44+n*2)
	copy(data[0:4], "RIFF")
	binary.LittleEndian.PutUint32(data[4:8], uint32(36+n*2))
	copy(data[8:12], "WAVE")
	copy(data[12:16], "fmt ")
	binary.LittleEndian.PutUint32(data[16:20], 16)
	binary.LittleEndian.PutUint16(data[20:22], 1)
	binary.LittleEndian.PutUint16(data[22:24], 1)
	binary.LittleEndian.PutUint32(data[24:28], rate)
	binary.LittleEndian.PutUint32(data[28:32], rate*2)
	binary.LittleEndian.PutUint16(data[32:34], 2)
	binary.LittleEndian.PutUint16(data[34:36], 16)
	copy(data[36:40], "data")
	binary.LittleEndian.PutUint32(data[40:44], uint32(n*2))
	for i := 0; i < n; i++ {
		t := float64(i) / rate
		env := 1 - t*8
		if env < 0 {
			env = 0
		}
		s := math.Sin(2*math.Pi*988*t) * env * 0.45
		v := int16(s * 32767)
		binary.LittleEndian.PutUint16(data[44+i*2:], uint16(v))
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		panic(err)
	}
}
