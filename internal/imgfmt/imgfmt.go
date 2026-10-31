// Package imgfmt registers the extra image formats every image.Decode call in
// the engine understands: BMP and WebP from golang.org/x/image, and TGA from
// the decoder in this package. PNG, JPEG, and GIF come from the standard
// library. Import it for its side effects.
package imgfmt

import (
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)
