package renderer

import (
	"image"
	"image/color"
	"testing"
)

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		name    string
		hex     string
		want    color.RGBA
		wantErr bool
	}{
		{
			name: "white",
			hex:  "#ffffff",
			want: color.RGBA{R: 255, G: 255, B: 255, A: 255},
		},
		{
			name: "black",
			hex:  "#000000",
			want: color.RGBA{R: 0, G: 0, B: 0, A: 255},
		},
		{
			name: "red",
			hex:  "#ff0000",
			want: color.RGBA{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name: "short form",
			hex:  "#fff",
			want: color.RGBA{R: 255, G: 255, B: 255, A: 255},
		},
		{
			name:    "invalid",
			hex:     "invalid",
			wantErr: true,
		},
		{
			name:    "too short",
			hex:     "#ff",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseHexColor(tt.hex)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}

func TestDrawSolidBackground(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	c := parseHexColorMust("#ff0000")

	drawSolidBackground(img, c)

	// Check a pixel in the middle
	pixel := img.At(50, 50)
	r, g, b, a := pixel.RGBA()

	if r != 0xffff || g != 0 || b != 0 || a != 0xffff {
		t.Errorf("expected red pixel, got r=%d g=%d b=%d a=%d", r, g, b, a)
	}
}

func TestDrawGradientBackground(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	top := parseHexColorMust("#ff0000")
	bottom := parseHexColorMust("#0000ff")

	drawGradientBackground(img, top, bottom)

	// Top should be red
	topPixel := img.At(50, 0)
	r, _, _, _ := topPixel.RGBA()
	if r < 0xf000 {
		t.Errorf("expected red at top, got r=%d", r)
	}

	// Bottom should be blue
	bottomPixel := img.At(50, 99)
	_, _, b, _ := bottomPixel.RGBA()
	if b < 0xf000 {
		t.Errorf("expected blue at bottom, got b=%d", b)
	}

	// Middle should be mixed
	midPixel := img.At(50, 50)
	mr, _, mb, _ := midPixel.RGBA()
	if mr < 0x4000 || mb < 0x4000 {
		t.Errorf("expected mixed color in middle, got mr=%d mb=%d", mr, mb)
	}
}

func parseHexColorMust(hex string) color.RGBA {
	c, err := parseHexColor(hex)
	if err != nil {
		panic(err)
	}
	return c
}
