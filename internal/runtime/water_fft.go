package runtime

import (
	"math"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// fftTile is one tiled Phillips spectrum. It is a CPU height added to Gerstner.
// It is not a GPU FFT ocean.
type fftTile struct {
	n    int
	size float32
	time float32
	h    []float32
}

func (wb *waterBody) ensureFFT() {
	if wb.fft != nil && wb.fft.n == waterFFTN {
		return
	}
	wb.fft = &fftTile{n: waterFFTN, size: 32}
}

// fftStrength is the SetFFTStrength scale: 1 unset, 0 when set to 0.
func (wb *waterBody) fftStrength() float32 {
	switch {
	case wb.fftAmp == 0:
		return 1
	case wb.fftAmp < 0:
		return 0
	}
	return wb.fftAmp
}

func (ft *fftTile) sample(x, z, t float32) float32 {
	if ft == nil || ft.n < 4 {
		return 0
	}
	if ft.h == nil || ft.time != t {
		ft.bake(t)
	}
	n := ft.n
	size := ft.size
	if size <= 0 {
		size = 32
	}
	u := x / size
	v := z / size
	u -= float32(math.Floor(float64(u)))
	v -= float32(math.Floor(float64(v)))
	fx := u * float32(n)
	fz := v * float32(n)
	x0 := int(fx) % n
	z0 := int(fz) % n
	x1 := (x0 + 1) % n
	z1 := (z0 + 1) % n
	tx := fx - float32(int(fx))
	tz := fz - float32(int(fz))
	h00 := ft.h[z0*n+x0]
	h10 := ft.h[z0*n+x1]
	h01 := ft.h[z1*n+x0]
	h11 := ft.h[z1*n+x1]
	hx0 := h00 + (h10-h00)*tx
	hx1 := h01 + (h11-h01)*tx
	return hx0 + (hx1-hx0)*tz
}

func (ft *fftTile) bake(t float32) {
	n := ft.n
	re := make([]float32, n*n)
	im := make([]float32, n*n)
	ft.spectrum(re, im, t)
	ifft2(re, im, n)
	ft.h = re
	ft.time = t
}

func (ft *fftTile) spectrum(re, im []float32, t float32) {
	n := ft.n
	g := float32(9.81)
	wind := float32(8)
	for z := 0; z < n; z++ {
		for x := 0; x < n; x++ {
			kx := float32(x - n/2)
			kz := float32(z - n/2)
			k := float32(math.Hypot(float64(kx), float64(kz)))
			if k < 0.5 {
				continue
			}
			phillips := float32(math.Exp(float64(-1/(k*k)))) / (k * k * k * k)
			phillips *= kx*kx + kz*kz
			amp := float32(math.Sqrt(float64(phillips))) * 0.35
			omega := float32(math.Sqrt(float64(g * k / wind)))
			phase := omega * t
			re[z*n+x] = amp * float32(math.Cos(float64(phase)))
			im[z*n+x] = amp * float32(math.Sin(float64(phase)))
		}
	}
}

func ifft2(re, im []float32, n int) {
	row := make([]float32, n)
	rowI := make([]float32, n)
	for z := 0; z < n; z++ {
		copy(row, re[z*n:(z+1)*n])
		copy(rowI, im[z*n:(z+1)*n])
		ifft1(row, rowI)
		copy(re[z*n:(z+1)*n], row)
		copy(im[z*n:(z+1)*n], rowI)
	}
	col := make([]float32, n)
	colI := make([]float32, n)
	inv := float32(waterFFTNorm)
	for x := 0; x < n; x++ {
		for z := 0; z < n; z++ {
			col[z] = re[z*n+x]
			colI[z] = im[z*n+x]
		}
		ifft1(col, colI)
		for z := 0; z < n; z++ {
			re[z*n+x] = col[z] * inv
		}
	}
}

func ifft1(re, im []float32) {
	n := len(re)
	for i, j := 0, 0; i < n; i++ {
		if i < j {
			re[i], re[j] = re[j], re[i]
			im[i], im[j] = im[j], im[i]
		}
		m := n >> 1
		for m >= 1 && j >= m {
			j -= m
			m >>= 1
		}
		j += m
	}
	for size := 2; size <= n; size <<= 1 {
		half := size >> 1
		ang := 2 * math.Pi / float64(size)
		wRe := float32(math.Cos(ang))
		wIm := float32(math.Sin(ang))
		for i := 0; i < n; i += size {
			cRe, cIm := float32(1), float32(0)
			for k := 0; k < half; k++ {
				o := i + k + half
				tRe := cRe*re[o] - cIm*im[o]
				tIm := cRe*im[o] + cIm*re[o]
				re[o] = re[i+k] - tRe
				im[o] = im[i+k] - tIm
				re[i+k] += tRe
				im[i+k] += tIm
				nRe := cRe*wRe - cIm*wIm
				cIm = cRe*wIm + cIm*wRe
				cRe = nRe
			}
		}
	}
}

// prepareWaterFFT draws every FFT water's tile for this frame, before the
// scene renders.
func (w *World) prepareWaterFFT() {
	for _, wb := range w.waters {
		if wb == nil || !wb.fftOn || wb.mat == nil {
			continue
		}
		w.uploadWaterFFT(wb, wb.mat.clock())
	}
}

// uploadWaterFFT sends the baked Phillips tile to a repeating RGBA texture.
// The red channel stores height as (h/4)+0.5 so the vertex shader can sample it.
func (w *World) uploadWaterFFT(wb *waterBody, t float32) uint32 {
	if wb == nil || !wb.fftOn {
		return 0
	}
	wb.ensureFFT()
	if wb.fft == nil {
		return 0
	}
	if id := w.drawGPUFFT(wb, t); id != 0 {
		return id
	}
	wb.fft.sample(0, 0, t)
	n := wb.fft.n
	if n < 4 || len(wb.fft.h) != n*n {
		return 0
	}
	if len(wb.fftPix) != n*n*4 {
		wb.fftPix = make([]byte, n*n*4)
	}
	if wb.fftGL == 0 || wb.fftGPUTime != wb.fft.time {
		for i, h := range wb.fft.h {
			e := float32(h)*wb.fftStrength()/4 + 0.5
			if e < 0 {
				e = 0
			}
			if e > 1 {
				e = 1
			}
			wb.fftPix[i*4] = byte(e * 255)
			wb.fftPix[i*4+3] = 255
		}
		if wb.fftGL == 0 {
			var tex uint32
			gl.GenTextures(1, &tex)
			wb.fftGL = tex
		}
		gl.BindTexture(gl.TEXTURE_2D, wb.fftGL)
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(n), int32(n), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(wb.fftPix))
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
		gl.BindTexture(gl.TEXTURE_2D, 0)
		wb.fftGPUTime = wb.fft.time
	}
	return wb.fftGL
}
