package render

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/mayanklad/pokecrt/internal/sprite"
)

func TestPixelPairs(t *testing.T) {
	red := color.NRGBA{R: 255, A: 255}
	blue := color.NRGBA{B: 255, A: 255}
	transparent := color.NRGBA{R: 123}
	tests := []struct {
		name           string
		top, bottom    color.NRGBA
		colored, plain string
	}{
		{"transparent", transparent, transparent, " ", " "},
		{"top", red, transparent, "\x1b[38;2;255;0;0m▀", "▀"},
		{"bottom", transparent, blue, "\x1b[38;2;0;0;255m▄", "▄"},
		{"different colors", red, blue, "\x1b[38;2;255;0;0m\x1b[48;2;0;0;255m▀", "█"},
		{"same color", red, red, "\x1b[38;2;255;0;0m\x1b[48;2;255;0;0m▀", "█"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pixels := image.NewNRGBA(image.Rect(0, 0, 1, 2))
			pixels.SetNRGBA(0, 0, test.top)
			pixels.SetNRGBA(0, 1, test.bottom)
			for _, colored := range []bool{true, false} {
				got, err := Render(pixels, colored)
				if err != nil {
					t.Fatal(err)
				}
				want := test.plain + "\n"
				if colored {
					want = reset + test.colored + reset + "\n"
				}
				if string(got) != want {
					t.Fatalf("color=%v: got %q, want %q", colored, got, want)
				}
			}
		})
	}
}

func TestOddHeightAndNonzeroOrigin(t *testing.T) {
	pixels := image.NewNRGBA(image.Rect(5, 7, 7, 10))
	pixels.SetNRGBA(5, 9, color.NRGBA{G: 255, A: 255})
	got, err := Render(pixels, false)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "  \n▀ \n" {
		t.Fatalf("got %q", got)
	}
	colored, err := Render(pixels, true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(colored), "48;2;") {
		t.Fatalf("transparent halves received background color: %q", colored)
	}
}

func TestBackgroundResetBetweenCells(t *testing.T) {
	pixels := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	pixels.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	pixels.SetNRGBA(0, 1, color.NRGBA{B: 255, A: 255})
	pixels.SetNRGBA(1, 0, color.NRGBA{G: 255, A: 255})
	got, err := Render(pixels, true)
	if err != nil {
		t.Fatal(err)
	}
	want := reset + "\x1b[38;2;255;0;0m\x1b[48;2;0;0;255m▀" + reset + "\x1b[38;2;0;255;0m▀" + reset + "\n"
	if string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestInvalidImages(t *testing.T) {
	partial := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	partial.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	partial.SetNRGBA(1, 1, color.NRGBA{R: 255, A: 128})
	for _, pixels := range []image.Image{nil, image.NewNRGBA(image.Rect(0, 0, 0, 0)), partial} {
		for _, colored := range []bool{true, false} {
			got, err := Render(pixels, colored)
			if err == nil || got != nil {
				t.Fatalf("expected error without partial output, got %q, %v", got, err)
			}
		}
	}
}

func TestBundledSprites(t *testing.T) {
	for _, asset := range sprite.Inventory() {
		t.Run(asset.Path, func(t *testing.T) {
			pixels, err := sprite.Decode(asset.Key)
			if err != nil {
				t.Fatal(err)
			}
			plain, err := Render(pixels, false)
			if err != nil {
				t.Fatal(err)
			}
			rows := strings.Split(strings.TrimSuffix(string(plain), "\n"), "\n")
			if len(rows) != (asset.Height+1)/2 {
				t.Fatalf("got %d rows", len(rows))
			}
			for _, row := range rows {
				if len([]rune(row)) != asset.Width {
					t.Fatalf("width changed: %q", row)
				}
			}
			colored, err := Render(pixels, true)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(colored), "\x1b[38;2;") || !strings.HasSuffix(string(colored), reset+"\n") {
				t.Fatal("missing truecolor or final reset")
			}
		})
	}
}
