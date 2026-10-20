package main

import "path/filepath"

// writeWebIcons renders the web build's favicon and home-screen icons in the
// default (dark) theme.
func writeWebIcons(dir string, p Palette, icon Grid) error {
	frac := iconWidth(icon)
	for name, size := range map[string]int{"favicon.png": 48, "icon-192.png": 192, "icon-512.png": 512} {
		if err := writePNG(filepath.Join(dir, name), renderIcon(p, icon, size, false, frac, 0.2)); err != nil {
			return err
		}
	}
	return nil
}
