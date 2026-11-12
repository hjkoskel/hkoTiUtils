package ti85family

import (
	"fmt"
	"image"
	"image/color"
)

// Screen width of the TI-85/86 in pixels.
const PictureWidth = 128

// PicturePalette is used for decoded pictures: index 0 = clear, 1 = set.
var PicturePalette = color.Palette{color.White, color.Black}

// Picture decodes picture variable data: a length word followed by a
// bitmap of 16 bytes per row, most significant bit leftmost. Pictures are
// normally 1008 bytes (128×63); 1024-byte (128×64) ones also occur. A
// truncated bitmap is decoded as far as it goes and an error is returned
// along with the image.
func Picture(data []byte) (*image.Paletted, error) {
	bits := InnerData(data)
	rows := (len(bits) + PictureWidth/8 - 1) / (PictureWidth / 8)
	img := image.NewPaletted(image.Rect(0, 0, PictureWidth, rows), PicturePalette)
	for i, b := range bits {
		y, x0 := i/(PictureWidth/8), i%(PictureWidth/8)*8
		for k := 0; k < 8; k++ {
			if b&(0x80>>k) != 0 {
				img.Pix[y*img.Stride+x0+k] = 1
			}
		}
	}
	var err error
	if len(bits) != 1008 && len(bits) != 1024 {
		err = fmt.Errorf("picture has %d bytes, expected 1008", len(bits))
	}
	return img, err
}
