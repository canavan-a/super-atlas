package main

import (
	"image"
	"image/color"
	"math"
	"path/filepath"
)

// writeExpoAssets renders the images app.json points at, in the default
// (dark) theme: the base app icon, the adaptive icon's layers and the
// favicon. On Android the per-theme icons from writeAndroid take over once
// the withAppIcon plugin has run; these are the fallback everywhere else.
func writeExpoAssets(dir string, p Palette, icon Grid) error {
	frac := iconWidth(icon)
	solid := func(Tone) RGB { return RGB{255, 255, 255} }
	files := map[string]image.Image{
		"icon.png":                    renderIcon(p, icon, 1024, false, frac, 0),
		"favicon.png":                 renderIcon(p, icon, 48, false, frac, 0.2),
		"android-icon-background.png": renderLayer(p, nil, 432, frac, nil, true),
		"android-icon-foreground.png": renderLayer(p, icon, 432, frac, p.color, false),
		"android-icon-monochrome.png": renderLayer(p, icon, 432, frac, solid, false),
	}
	for name, img := range files {
		if err := writePNG(filepath.Join(dir, name), img); err != nil {
			return err
		}
	}
	return nil
}

// renderLayer draws one adaptive-icon layer: the theme background (bg), or
// the icon's pixels on transparency, colored by paint. Supersampled 3x3.
func renderLayer(p Palette, icon Grid, size int, widthFrac float64, paint func(Tone) RGB, bg bool) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	if bg {
		c := color.NRGBA{clamp(p.Bg[0]), clamp(p.Bg[1]), clamp(p.Bg[2]), 255}
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				img.SetNRGBA(x, y, c)
			}
		}
		return img
	}
	pw, ph, x0, y0 := iconGeom(icon, float64(size), widthFrac*float64(size))
	const ss = 3
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			var acc RGB
			var cover float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					fx := float64(px) + (float64(sx)+0.5)/ss
					fy := float64(py) + (float64(sy)+0.5)/ss
					ch := icon.at(int(math.Floor((fx-x0)/pw)), int(math.Floor((fy-y0)/ph)))
					if ch == '.' {
						continue
					}
					c := paint(legend[ch])
					acc[0], acc[1], acc[2] = acc[0]+c[0], acc[1]+c[1], acc[2]+c[2]
					cover++
				}
			}
			if cover == 0 {
				continue
			}
			img.SetNRGBA(px, py, color.NRGBA{
				R: clamp(acc[0] / cover), G: clamp(acc[1] / cover), B: clamp(acc[2] / cover),
				A: uint8(255 * cover / (ss * ss)),
			})
		}
	}
	return img
}
