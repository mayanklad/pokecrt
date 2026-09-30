// Package render converts normalized sprites into terminal half blocks.
package render

import (
	"bytes"
	"fmt"
	"image"
)

const reset = "\x1b[0m"

// Render preserves source dimensions: one column per pixel and one row per
// vertical pixel pair. Transparency uses the terminal's default background.
// With color disabled, only spaces, block glyphs, and newlines are emitted.
// Partial alpha is rejected; dataset preprocessing must resolve it explicitly.
func Render(pixels image.Image, colorEnabled bool) ([]byte, error) {
	if pixels == nil {
		return nil, fmt.Errorf("cannot render a nil image")
	}
	bounds := pixels.Bounds()
	if bounds.Empty() {
		return nil, fmt.Errorf("cannot render an empty image")
	}
	var output bytes.Buffer
	for y := bounds.Min.Y; y < bounds.Max.Y; y += 2 {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			top, err := readPixel(pixels, x, y)
			if err != nil {
				return nil, err
			}
			bottom := pixel{}
			if y+1 < bounds.Max.Y {
				bottom, err = readPixel(pixels, x, y+1)
				if err != nil {
					return nil, err
				}
			}
			if colorEnabled {
				// Reset before each cell so an opaque neighbor cannot fill a transparent
				// half with its background color. Also restores defaults for blank cells.
				output.WriteString(reset)
				if top.opaque {
					foreground(&output, top)
				} else if bottom.opaque {
					foreground(&output, bottom)
				}
				if top.opaque && bottom.opaque {
					fmt.Fprintf(&output, "\x1b[48;2;%d;%d;%dm", bottom.r, bottom.g, bottom.b)
				}
			}
			switch {
			case top.opaque && bottom.opaque:
				if colorEnabled {
					output.WriteRune('▀')
				} else {
					output.WriteRune('█')
				}
			case top.opaque:
				output.WriteRune('▀')
			case bottom.opaque:
				output.WriteRune('▄')
			default:
				output.WriteByte(' ')
			}
		}
		if colorEnabled {
			output.WriteString(reset)
		}
		output.WriteByte('\n')
	}
	return output.Bytes(), nil
}

type pixel struct {
	r, g, b uint32
	opaque  bool
}

func readPixel(pixels image.Image, x, y int) (pixel, error) {
	r, g, b, a := pixels.At(x, y).RGBA()
	if a == 0 {
		return pixel{}, nil
	}
	if a != 0xffff {
		return pixel{}, fmt.Errorf("partial alpha at pixel (%d, %d)", x, y)
	}
	return pixel{r >> 8, g >> 8, b >> 8, true}, nil
}

func foreground(output *bytes.Buffer, p pixel) {
	fmt.Fprintf(output, "\x1b[38;2;%d;%d;%dm", p.r, p.g, p.b)
}
