package main

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"path/filepath"
)

// writeMarketplace renders the store graphics in the default (dark) theme, as
// super-chat's does, all but the icon the open scroll with the wordmark
// centered on the theme's background. Google Play: the 512x512 app icon (a
// full square, which Play rounds itself) and the 1024x500 feature graphic.
// App Store: the 3840x1646 product page header, whose scroll stays inside
// Apple's centred 1646x661 safe area since the sides are cropped on narrow
// screens, and the 3840x2560 (3:2) search results image.
func writeMarketplace(dir string, p Palette, s scroll) error {
	if err := writePNG(filepath.Join(dir, "icon-512.png"), renderIcon(p, s.launcher, 512, false, 0)); err != nil {
		return err
	}
	if err := writePNG(filepath.Join(dir, "feature-graphic-1024x500.png"), renderBanner(p, s.icon, 1024, 500, 0.6*1024, 0.7*500)); err != nil {
		return err
	}
	if err := writePNG(filepath.Join(dir, "appstore-header-3840x1646.png"), renderBanner(p, s.icon, 3840, 1646, 0.9*1646, 0.9*661)); err != nil {
		return err
	}
	return writePNG(filepath.Join(dir, "appstore-search-3840x2560.png"), renderBanner(p, s.icon, 3840, 2560, 0.6*3840, 0.7*2560))
}

// renderBanner draws a picture centered on the theme's background, as large
// as fits a maxW x maxH box; cells are 3:4 like the art's.
func renderBanner(p Palette, g Grid, w, h int, maxW, maxH float64) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	bg := p.Bg
	draw.Draw(img, img.Bounds(), &image.Uniform{color.NRGBA{clamp(bg[0]), clamp(bg[1]), clamp(bg[2]), 255}}, image.Point{}, draw.Src)
	cw := math.Min(maxW/float64(g.w()), maxH/(float64(len(g))*pixAspect))
	ch := cw * pixAspect
	x0 := (float64(w) - cw*float64(g.w())) / 2
	y0 := (float64(h) - ch*float64(len(g))) / 2
	for y, row := range g {
		for x := range len(row) {
			if row[x] == '.' {
				continue
			}
			c := p.color(legend[row[x]])
			r := image.Rect(
				int(math.Round(x0+float64(x)*cw)), int(math.Round(y0+float64(y)*ch)),
				int(math.Round(x0+float64(x+1)*cw)), int(math.Round(y0+float64(y+1)*ch)),
			)
			draw.Draw(img, r, &image.Uniform{color.NRGBA{clamp(c[0]), clamp(c[1]), clamp(c[2]), 255}}, image.Point{}, draw.Src)
		}
	}
	return img
}
