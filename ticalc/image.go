package ticalc

import (
	"errors"
	"image"
	"image/color"
	"strings"
)

// ErrNotImage is returned by Imager.Image for variables that are not pictures.
var ErrNotImage = errors.New("variable is not a picture")

// Imager is implemented by calculators that can render picture variables.
type Imager interface {
	Image(v *Variable) (image.Image, error)
}

// ImageText draws a monochrome image as Unicode block characters, two pixel
// rows per text line (▀ ▄ █ and space). A pixel counts as set when it is
// darker than mid gray.
func ImageText(img image.Image) string {
	b := img.Bounds()
	on := func(x, y int) bool {
		if y >= b.Max.Y {
			return false
		}
		g := color.GrayModel.Convert(img.At(x, y)).(color.Gray)
		return g.Y < 0x80
	}
	var sb strings.Builder
	for y := b.Min.Y; y < b.Max.Y; y += 2 {
		line := make([]rune, 0, b.Dx())
		for x := b.Min.X; x < b.Max.X; x++ {
			switch top, bot := on(x, y), on(x, y+1); {
			case top && bot:
				line = append(line, '█')
			case top:
				line = append(line, '▀')
			case bot:
				line = append(line, '▄')
			default:
				line = append(line, ' ')
			}
		}
		sb.WriteString(strings.TrimRight(string(line), " "))
		sb.WriteByte('\n')
	}
	return sb.String()
}
