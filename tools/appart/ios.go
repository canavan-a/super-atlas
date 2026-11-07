package main

import "path/filepath"

// writeIOSIcon renders the iOS app icon: one 1024 px square in the default
// (dark) theme, which Xcode scales to every size. iOS rounds the corners
// itself, and the App Store refuses an icon with transparency, so it's a
// full square (fully opaque, so it's written without an alpha channel).
// iOS can switch icons too, but only with a system alert each time, so the
// icon doesn't follow the theme there (src/appIcon.ts is Android only).
func writeIOSIcon(dir string, p Palette, icon Grid) error {
	return writePNG(filepath.Join(dir, "icon-1024.png"), renderIcon(p, icon, 1024, false, 0))
}
