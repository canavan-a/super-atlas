// Command appart generates the app's logo data and launcher icons from the
// pixel art in art/, adapted from super-chat's tools/appart. The boot screen
// and the notification icon are a scroll: unrolled, it shows the stacked
// "Super Atlas" wordmark (art/word.txt); unrolled again in the boot
// animation, a world map. The launcher icons are that map on its own, out of
// the scroll, through a fisheye lens (lensSize, lensStrength).
//
//	cd tools && go run ./appart                      # writes into ../app
//	cd tools && go run ./appart -preview /tmp/out    # also renders PNG previews
//
// It reads art/word.txt and the scroll's layers, art/scroll_paper.txt,
// scroll_map.txt and scroll_rollers.txt (one 96 x 72 grid), and writes:
//
//	app/src/logo/logoData.ts      the scroll's layers as tone-mapped vector runs
//	app/src/logo/iconPalette.json per-theme colors
//	app/android/app/src/main/res  adaptive + legacy launcher icons, one per
//	                              theme, and the notification icon
//	app/public                    the web build's favicon and icons (dark
//	                              theme, the app's default)
//
// Theme colors are read from app/src/theme.tsx, never copied.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// The app's themes, in the order Settings lists them.
var themeOrder = []string{"light", "dark", "slate", "sepia", "ember"}

// Tone is where a pixel sits on a theme's dark→light ramp (U, 0..4) and how
// far it leans toward the theme's accent (W, 0..1).
type Tone struct{ U, W float64 }

// The art files' legend. A–I are the wordmark's tones, graded letter by
// letter left to right exactly as super-badger's are. The rest are the
// scroll's: a–e its outline, graded left to right toward the accent; r the
// rollers' wood, h and k their light and dark sides; p the paper, s paper in
// shadow where it curls into a roller; g the map's grid; o the accent (the
// map's land, the handles' knobs).
var legend = map[byte]Tone{
	'A': {4, 1}, 'B': {3.8333, 0.5}, 'C': {4, 0.5}, 'D': {3.6667, 0.5}, 'E': {3.5, 0.5},
	'F': {3.3333, 0.5}, 'G': {3.3333, 0}, 'H': {3.1667, 0}, 'I': {3, 0},
	'a': {3.45, 0.05}, 'b': {3.45, 0.16}, 'c': {3.45, 0.27}, 'd': {3.45, 0.38}, 'e': {3.45, 0.5},
	'r': {2.3, 0}, 'h': {2.7, 0}, 'k': {1.95, 0},
	'p': {1.25, 0}, 's': {1.7, 0}, 'g': {1.85, 0},
	'o': {4, 1},
}

// isLetter reports whether a pixel belongs to the wordmark's lettering.
func isLetter(c byte) bool { return c >= 'A' && c <= 'I' }

// The scroll's geometry on its 96 x 72 grid (art/scroll_*.txt). Open, the
// left roller sits at column openX0; closed, the two meet in the middle, the
// left one at closedX0. The visible paper reaches revealInset columns short
// of the travel, because the rollers and their curl cover its edge.
const (
	scrollCenter = 48
	openX0       = 5
	closedX0     = 44
	revealInset  = 3
)

// Grid is a picture as rows of legend bytes ('.' is empty).
type Grid []string

func (g Grid) w() int {
	n := 0
	for _, r := range g {
		n = max(n, len(r))
	}
	return n
}

func (g Grid) at(x, y int) byte {
	if y < 0 || y >= len(g) || x < 0 || x >= len(g[y]) {
		return '.'
	}
	return g[y][x]
}

type span struct {
	name   string
	x0, x1 int
	// y0, y1 bound the letter's rows once the wordmark is stacked; 0, 0
	// means every row.
	y0, y1 int
}

func readArt(path string) (Grid, []span, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	var g Grid
	var slices []span
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "#") {
			if rest, ok := strings.CutPrefix(line, "# slices:"); ok {
				for _, f := range strings.Fields(rest) {
					name, rng, _ := strings.Cut(f, ":")
					a, b, _ := strings.Cut(rng, "-")
					x0, _ := strconv.Atoi(a)
					x1, _ := strconv.Atoi(b)
					slices = append(slices, span{name: name, x0: x0, x1: x1})
				}
			}
			continue
		}
		for i := 0; i < len(line); i++ {
			if line[i] != '.' {
				if _, ok := legend[line[i]]; !ok {
					return nil, nil, fmt.Errorf("%s: unknown pixel %q", path, line[i])
				}
			}
		}
		g = append(g, line)
	}
	return g, slices, sc.Err()
}

// group is a set of pixels as runs [x, y, len, …] per tone.
type group map[Tone][]int

func groupOf(g Grid, x0, x1 int, keep func(byte) bool) group {
	out := group{}
	for y, row := range g {
		for x := x0; x < min(x1, len(row)); {
			c := row[x]
			if c == '.' || !keep(c) {
				x++
				continue
			}
			t := legend[c]
			n := 1
			for x+n < min(x1, len(row)) && row[x+n] != '.' && keep(row[x+n]) && legend[row[x+n]] == t {
				n++
			}
			out[t] = append(out[t], x, y, n)
			x += n
		}
	}
	return out
}

func sortedTones(g group) []Tone {
	ts := make([]Tone, 0, len(g))
	for t := range g {
		ts = append(ts, t)
	}
	sort.Slice(ts, func(i, j int) bool {
		if ts[i].U != ts[j].U {
			return ts[i].U < ts[j].U
		}
		return ts[i].W < ts[j].W
	})
	return ts
}

func num(f float64) string {
	s := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", f), "0"), ".")
	if s == "" || s == "-" {
		return "0"
	}
	return s
}

func tonesTS(g group, indent string) string {
	var b strings.Builder
	b.WriteString("[\n")
	for _, t := range sortedTones(g) {
		b.WriteString(fmt.Sprintf("%s  {u: %s, w: %s, runs: [", indent, num(t.U), num(t.W)))
		for i, v := range g[t] {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(strconv.Itoa(v))
		}
		b.WriteString("]},\n")
	}
	b.WriteString(indent + "]")
	return b.String()
}

func main() {
	artDir := flag.String("art", "../art", "the art directory")
	app := flag.String("app", "../app", "the app directory")
	preview := flag.String("preview", "", "also render PNG previews into this directory")
	flag.Parse()

	word, slices, err := readArt(filepath.Join(*artDir, "word.txt"))
	must(err)
	layers := map[string]Grid{}
	for _, name := range []string{"paper", "map", "rollers"} {
		g, _, err := readArt(filepath.Join(*artDir, "scroll_"+name+".txt"))
		must(err)
		layers[name] = g
	}
	themeSrc, err := os.ReadFile(filepath.Join(*app, "src/theme.tsx"))
	must(err)
	pals, err := parseThemes(string(themeSrc))
	must(err)
	for _, n := range themeOrder {
		if _, ok := pals[n]; !ok {
			must(fmt.Errorf("theme %q not found in theme.tsx", n))
		}
	}

	s := buildScroll(layers, word, slices)
	must(writeLogoData(filepath.Join(*app, "src/logo/logoData.ts"), s))
	must(writePalette(filepath.Join(*app, "src/logo/iconPalette.json"), pals, s.icon))
	must(writeAndroid(filepath.Join(*app, "android/app/src/main/res"), pals, s.launcher, s.icon))
	must(writeWebIcons(filepath.Join(*app, "public"), pals["dark"], s.launcher))
	if *preview != "" {
		must(writePreviews(*preview, pals, s))
	}
	fmt.Println("generated logo data, palette and icons from", *artDir)
}

// scroll is the boot scroll's parts, all on the scroll's own grid.
type scroll struct {
	w, h                int
	paperText, paperMap Grid // the unrolled paper with the wordmark, and with the map
	rollerL, rollerR    Grid // the rollers, open
	icon                Grid // the open scroll with the wordmark, cropped
	launcher            Grid // the map alone, cropped: the launcher icons
}

func buildScroll(layers map[string]Grid, word Grid, slices []span) scroll {
	paper := layers["paper"]
	w, h := paper.w(), len(paper)
	for name, g := range layers {
		if g.w() != w || len(g) != h {
			must(fmt.Errorf("scroll_%s.txt is %dx%d; scroll_paper.txt is %dx%d", name, g.w(), len(g), w, h))
		}
	}
	// The wordmark stacked, "Super" over "Atlas", and centered on the paper.
	stacked, _ := stackWord(word, slices)
	stacked = crop(stacked)
	x := (w - stacked.w()) / 2
	y := int(math.Round(float64(h-1)/2 - float64(len(stacked))/2))
	s := scroll{
		w: w, h: h,
		paperText: overlay(paper, stacked, x, y),
		paperMap:  overlay(paper, layers["map"], 0, 0),
		rollerL:   sliceGrid(layers["rollers"], 0, scrollCenter),
		rollerR:   overlay(blank(w, h), sliceGrid(layers["rollers"], scrollCenter, w), scrollCenter, 0),
	}
	s.rollerL = overlay(blank(w, h), s.rollerL, 0, 0)
	s.icon = crop(overlay(s.paperText, layers["rollers"], 0, 0))
	s.launcher = crop(layers["map"])
	return s
}

// blank is an empty w x h grid.
func blank(w, h int) Grid {
	g := make(Grid, h)
	for i := range g {
		g[i] = strings.Repeat(".", w)
	}
	return g
}

// overlay draws top's pixels over base, top's (0, 0) at (x, y).
func overlay(base, top Grid, x, y int) Grid {
	out := make(Grid, len(base))
	for j, row := range base {
		b := []byte(row)
		if ty := j - y; ty >= 0 && ty < len(top) {
			for i := range b {
				if c := top.at(i-x, ty); c != '.' {
					b[i] = c
				}
			}
		}
		out[j] = string(b)
	}
	return out
}

// center pads every row to width w, keeping the picture centered.
func center(g Grid, w int) Grid {
	gw := g.w()
	left := (w - gw) / 2
	out := make(Grid, len(g))
	for i, r := range g {
		out[i] = strings.Repeat(".", left) + r + strings.Repeat(".", w-left-len(r))
	}
	return out
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "appart:", err)
		os.Exit(1)
	}
}

func writeLogoData(path string, s scroll) error {
	all := func(byte) bool { return true }
	var b strings.Builder
	b.WriteString(`// GENERATED by "go run ./appart" (from tools/) out of art/word.txt and
// art/scroll_*.txt. Do not edit by hand; edit the art and regenerate.
//
// The logo is described by *tone*, not color: u is the position along the
// art's own dark→light ramp (0..4) and w how far the pixel leans toward the
// accent (0..1). logoTheme.ts turns tones into a theme's colors, so one set of
// shapes serves every theme. runs is flat [x, y, length, ...] in sub-pixels
// (one sub-pixel is 3 units wide and 4 tall; see LogoArt.tsx).

export interface ToneRuns {
  u: number;
  w: number;
  runs: number[];
}

/** The boot scroll, every layer on one grid of width x height sub-pixels. */
export interface LogoData {
  width: number;
  height: number;
  /** the unrolled paper with the stacked wordmark (and the launcher icon) */
  paperText: ToneRuns[];
  /** the unrolled paper with the world map */
  paperMap: ToneRuns[];
  /** the rollers, at the open position */
  rollerLeft: ToneRuns[];
  rollerRight: ToneRuns[];
  /** the column the scroll opens from */
  center: number;
  /** columns each roller travels between open and closed */
  travel: number;
  /** open by t (0..1), the paper shows travel * t - revealInset columns either side of center */
  revealInset: number;
}

`)
	b.WriteString(fmt.Sprintf("export const LOGO: LogoData = {\n  width: %d,\n  height: %d,\n", s.w, s.h))
	b.WriteString(fmt.Sprintf("  paperText: %s,\n", tonesTS(groupOf(s.paperText, 0, s.w, all), "  ")))
	b.WriteString(fmt.Sprintf("  paperMap: %s,\n", tonesTS(groupOf(s.paperMap, 0, s.w, all), "  ")))
	b.WriteString(fmt.Sprintf("  rollerLeft: %s,\n", tonesTS(groupOf(s.rollerL, 0, s.w, all), "  ")))
	b.WriteString(fmt.Sprintf("  rollerRight: %s,\n", tonesTS(groupOf(s.rollerR, 0, s.w, all), "  ")))
	b.WriteString(fmt.Sprintf("  center: %d,\n  travel: %d,\n  revealInset: %d,\n};\n", scrollCenter, closedX0-openX0, revealInset))
	return writeFile(path, b.String())
}

// crop trims empty rows and columns around a picture.
func crop(g Grid) Grid {
	minX, minY, maxX, maxY := math.MaxInt, math.MaxInt, -1, -1
	for y, r := range g {
		for x := range len(r) {
			if r[x] != '.' {
				minX, maxX, minY, maxY = min(minX, x), max(maxX, x), min(minY, y), max(maxY, y)
			}
		}
	}
	out := Grid{}
	for y := minY; y <= maxY; y++ {
		row := g[y] + strings.Repeat(".", max(0, maxX+1-len(g[y])))
		out = append(out, row[minX:maxX+1])
	}
	return out
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// ---- palette (checked by the app's tests) ----

type toneJSON struct {
	U     float64 `json:"u"`
	W     float64 `json:"w"`
	Color string  `json:"color"`
}

func writePalette(path string, pals map[string]Palette, icon Grid) error {
	type themeJSON struct {
		Bg        string     `json:"bg"`
		Accent    string     `json:"accent"`
		Ramp      []string   `json:"ramp"`
		IconTones []toneJSON `json:"iconTones"`
	}
	out := struct {
		Themes map[string]themeJSON `json:"themes"`
	}{Themes: map[string]themeJSON{}}
	tones := sortedTones(groupOf(icon, 0, icon.w(), func(byte) bool { return true }))
	for _, n := range themeOrder {
		p := pals[n]
		t := themeJSON{Bg: toHex(p.Bg), Accent: toHex(p.accent())}
		for _, r := range p.ramp() {
			t.Ramp = append(t.Ramp, toHex(r))
		}
		for _, tone := range tones {
			t.IconTones = append(t.IconTones, toneJSON{tone.U, tone.W, toHex(p.color(tone))})
		}
		out.Themes[n] = t
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(path, string(b)+"\n")
}

// ---- Android icons ----

const (
	overlapU  = 0.03
	pixAspect = 4.0 / 3.0 // sub-pixels are 3 wide by 4 tall
)

func rectPath(runs []int, x0, y0, pw, ph float64) string {
	var b strings.Builder
	for i := 0; i < len(runs); i += 3 {
		x := x0 + float64(runs[i])*pw
		y := y0 + float64(runs[i+1])*ph
		w := float64(runs[i+2])*pw + overlapU
		b.WriteString(fmt.Sprintf("M%s,%sh%sv%sh-%sz", num(x), num(y), num(w), num(ph+overlapU), num(w)))
	}
	return b.String()
}

// maxPathData caps one <path>'s pathData. aapt2 can't store a string over
// 32767 bytes in compiled XML: it silently swaps in "STRING_TOO_LARGE", the
// drawable fails to inflate, and the launcher falls back to the default icon.
const maxPathData = 8000

// pathElems draws runs in one color as as many <path>s as keep each under
// maxPathData. The runs are separate rectangles, so splitting changes nothing.
func pathElems(fill string, runs []int, x0, y0, pw, ph float64) string {
	var b strings.Builder
	for start := 0; start < len(runs); {
		end, n := start, 0
		for end < len(runs) {
			n += len(rectPath(runs[end:end+3], x0, y0, pw, ph))
			if n > maxPathData && end > start {
				break
			}
			end += 3
		}
		b.WriteString(fmt.Sprintf("    <path android:fillColor=\"%s\" android:pathData=\"%s\" />\n", fill, rectPath(runs[start:end], x0, y0, pw, ph)))
		start = end
	}
	return b.String()
}

func iconGeom(icon Grid, size, width float64) (pw, ph, x0, y0 float64) {
	pw = width / float64(icon.w())
	ph = pw * pixAspect
	x0 = (size - float64(icon.w())*pw) / 2
	y0 = (size - float64(len(icon))*ph) / 2
	return
}

func vector(size int, paths string) string {
	return fmt.Sprintf("<vector xmlns:android=\"http://schemas.android.com/apk/res/android\"\n    android:width=\"%ddp\"\n    android:height=\"%ddp\"\n    android:viewportWidth=\"%d\"\n    android:viewportHeight=\"%d\">\n%s</vector>\n", size, size, size, size, paths)
}

func writeAndroid(res string, pals map[string]Palette, launcher, scroll Grid) error {
	colors := []string{"<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<!-- GENERATED by tools/appart. One launcher-icon background per app theme. -->\n<resources>\n"}
	for _, n := range themeOrder {
		p := pals[n]
		colors = append(colors, fmt.Sprintf("    <color name=\"ic_bg_%s\">%s</color>\n", n, toHex(p.Bg)))
		// The foreground goes through the lens, so it's a bitmap per density
		// rather than a vector: the warped pixels have curved edges.
		if err := writeAdaptiveLayer(res, "ic_launcher_fg_"+n, launcher, func(c byte) (RGB, bool) {
			return p.color(legend[c]), c != '.'
		}); err != nil {
			return err
		}
		adaptive := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<!-- GENERATED by tools/appart. API 26+ adaptive icon for the %s theme; below
     API 26 the matching mipmap-*dpi PNGs are used. -->
<adaptive-icon xmlns:android="http://schemas.android.com/apk/res/android">
    <background android:drawable="@color/ic_bg_%s" />
    <foreground android:drawable="@mipmap/ic_launcher_fg_%s" />
    <monochrome android:drawable="@mipmap/ic_launcher_mono" />
</adaptive-icon>
`, n, n, n)
		for _, name := range []string{"ic_launcher_" + n, "ic_launcher_" + n + "_round"} {
			if err := writeFile(filepath.Join(res, "mipmap-anydpi-v26", name+".xml"), adaptive); err != nil {
				return err
			}
		}
		if err := writeLegacyPNGs(res, n, p, launcher); err != nil {
			return err
		}
	}
	colors = append(colors, "</resources>\n")
	if err := writeFile(filepath.Join(res, "values", "ic_theme_icons.xml"), strings.Join(colors, "")); err != nil {
		return err
	}

	// The themed (Android 13+) launcher icon: the map, land and grid, in one
	// color, through the lens like the colored one.
	white := RGB{255, 255, 255}
	if err := writeAdaptiveLayer(res, "ic_launcher_mono", launcher, func(c byte) (RGB, bool) {
		return white, c != '.'
	}); err != nil {
		return err
	}
	// The status-bar icon is the scroll's silhouette, with the wordmark's
	// letters cut out of it so they still read, flat, as large as fits 22 of
	// its 24dp.
	var all []int
	shape := groupOf(scroll, 0, scroll.w(), func(c byte) bool { return !isLetter(c) })
	for _, t := range sortedTones(shape) {
		all = append(all, shape[t]...)
	}
	aspect := float64(len(scroll)) * pixAspect / float64(scroll.w())
	npw, nph, nx0, ny0 := iconGeom(scroll, 24, 22/max(1, aspect))
	notif := "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<!-- GENERATED by tools/appart: the scroll as the status-bar notification icon. -->\n" +
		vector(24, pathElems("#FFFFFF", all, nx0, ny0, npw, nph))
	return writeFile(filepath.Join(res, "drawable", "ic_notification.xml"), notif)
}

// writeAdaptiveLayer writes one adaptive-icon layer as a mipmap PNG per
// density: the 108dp canvas, transparent where shade says so, with the lens
// measured against the 72dp area launchers show (their masks fit inside it).
func writeAdaptiveLayer(res, name string, icon Grid, shade func(c byte) (RGB, bool)) error {
	for _, d := range densities {
		size := d.px * 108 / 48
		r := float64(size) * 36 / 108
		img := renderLens(size, r, func(nx, ny float64) (RGB, bool) { return shade(lensCell(icon, nx, ny)) })
		if err := writePNG(filepath.Join(res, "mipmap-"+d.name, name+".png"), img); err != nil {
			return err
		}
	}
	return nil
}

var densities = []struct {
	name string
	px   int
}{{"mdpi", 48}, {"hdpi", 72}, {"xhdpi", 96}, {"xxhdpi", 144}, {"xxxhdpi", 192}}

func writeLegacyPNGs(res, theme string, p Palette, icon Grid) error {
	for _, d := range densities {
		for _, round := range []bool{false, true} {
			name := "ic_launcher_" + theme
			if round {
				name += "_round"
			}
			if err := writePNG(filepath.Join(res, "mipmap-"+d.name, name+".png"), renderIcon(p, icon, d.px, round, 0.2)); err != nil {
				return err
			}
		}
	}
	return nil
}

func writePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// The launcher icons' fisheye (variant "Gentle lens", picked in an icon lab):
// the art is sized so its corners sit lensSize of the way out to the shape's
// edge, then magnified toward the middle by lensStrength while the edge stays
// put, so the scroll swells out to fill the shape.
const (
	lensSize     = 0.95
	lensStrength = 0.3
)

// lensCell is the art cell under (nx, ny): a point measured from the
// shape's center, in units of its radius.
func lensCell(icon Grid, nx, ny float64) byte {
	hw, hh := float64(icon.w())/2, float64(len(icon))*pixAspect/2
	m := (1 + lensStrength*(nx*nx+ny*ny-1)) / lensSize * math.Hypot(hw, hh)
	return icon.at(int(math.Floor(nx*m+hw)), int(math.Floor((ny*m+hh)/pixAspect)))
}

// renderLens rasterizes a size x size image, supersampled 3x3 for clean
// edges. shade gives each sample's color from its lens coordinates (against
// a shape of radius r pixels, centered), or false to leave it transparent.
func renderLens(size int, r float64, shade func(nx, ny float64) (RGB, bool)) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	const ss = 3
	c := float64(size) / 2
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			var acc RGB
			var cover float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					fx := float64(px) + (float64(sx)+0.5)/ss
					fy := float64(py) + (float64(sy)+0.5)/ss
					col, ok := shade((fx-c)/r, (fy-c)/r)
					if !ok {
						continue
					}
					cover++
					acc[0], acc[1], acc[2] = acc[0]+col[0], acc[1]+col[1], acc[2]+col[2]
				}
			}
			if cover == 0 {
				continue
			}
			img.SetNRGBA(px, py, color.NRGBA{
				R: uint8(acc[0]/cover + 0.5), G: uint8(acc[1]/cover + 0.5), B: uint8(acc[2]/cover + 0.5),
				A: uint8(255 * cover / (ss * ss)),
			})
		}
	}
	return img
}

// renderIcon rasterizes the icon, through the lens, onto the theme's
// background in a rounded square (or circle). corner is the corner radius as
// a fraction of the size (0: a full square).
func renderIcon(p Palette, icon Grid, size int, round bool, corner float64) *image.NRGBA {
	half := float64(size) / 2
	r := corner * float64(size)
	return renderLens(size, half, func(nx, ny float64) (RGB, bool) {
		if !inShape(half+nx*half, half+ny*half, float64(size), r, round) {
			return RGB{}, false
		}
		if c := lensCell(icon, nx, ny); c != '.' {
			return p.color(legend[c]), true
		}
		return p.Bg, true
	})
}

func inShape(x, y, size, r float64, round bool) bool {
	if round {
		c := size / 2
		return (x-c)*(x-c)+(y-c)*(y-c) <= c*c
	}
	cx := math.Min(math.Max(x, r), size-r)
	cy := math.Min(math.Max(y, r), size-r)
	return (x-cx)*(x-cx)+(y-cy)*(y-cy) <= r*r
}

// ---- previews ----

// writePreviews renders, per theme, the boot animation's key frames side by
// side (closed, half open, open on the wordmark, open on the map) and the
// launcher icon.
func writePreviews(dir string, pals map[string]Palette, s scroll) error {
	const pw, ph, scale = 3, 4, 2
	travel := closedX0 - openX0
	frame := func(paper Grid, open float64) Grid {
		shift := int(math.Round(float64(travel) * (1 - open)))
		half := float64(travel)*open - revealInset
		shown := blank(s.w, s.h)
		for y, row := range paper {
			b := []byte(shown[y])
			for x := range len(row) {
				if math.Abs(float64(x)+0.5-scrollCenter) <= half {
					b[x] = row[x]
				}
			}
			shown[y] = string(b)
		}
		g := overlay(shown, s.rollerL, shift, 0)
		return overlay(g, s.rollerR, -shift, 0)
	}
	frames := []Grid{frame(s.paperText, 0), frame(s.paperText, 0.5), frame(s.paperText, 1), frame(s.paperMap, 1)}
	for _, n := range themeOrder {
		p := pals[n]
		img := image.NewNRGBA(image.Rect(0, 0, len(frames)*(s.w+4)*pw*scale, (s.h+4)*ph*scale))
		bg := p.Bg
		draw.Draw(img, img.Bounds(), &image.Uniform{color.NRGBA{clamp(bg[0]), clamp(bg[1]), clamp(bg[2]), 255}}, image.Point{}, draw.Src)
		for i, g := range frames {
			ox := i*(s.w+4) + 2
			for y, row := range g {
				for x := range len(row) {
					if row[x] == '.' {
						continue
					}
					c := p.color(legend[row[x]])
					rc := image.Rect((x+ox)*pw*scale, (y+2)*ph*scale, (x+ox+1)*pw*scale, (y+3)*ph*scale)
					draw.Draw(img, rc, &image.Uniform{color.NRGBA{clamp(c[0]), clamp(c[1]), clamp(c[2]), 255}}, image.Point{}, draw.Src)
				}
			}
		}
		if err := writePNG(filepath.Join(dir, "boot_"+n+".png"), img); err != nil {
			return err
		}
		if err := writePNG(filepath.Join(dir, "icon_"+n+".png"), renderIcon(p, s.launcher, 256, false, 0.2)); err != nil {
			return err
		}
	}
	return nil
}

// stackWord cuts the wordmark after "Super" (the first five letter slices)
// and stacks "Atlas" under it, each centered, one sub-row apart. The slices
// move with their letters and gain row bounds, since the two lines now share
// columns.
func stackWord(word Grid, slices []span) (Grid, []span) {
	cut := slices[5].x0
	top, tx, _ := cropAt(sliceGrid(word, 0, cut))
	bot, bx, _ := cropAt(sliceGrid(word, cut, word.w()))
	w := max(top.w(), bot.w())
	topOff, botOff := (w-top.w())/2, (w-bot.w())/2
	botY := len(top) + 1
	out := append(center(top, w), strings.Repeat(".", w))
	out = append(out, center(bot, w)...)
	moved := make([]span, len(slices))
	for i, s := range slices {
		if s.x0 < cut {
			moved[i] = span{s.name, s.x0 - tx + topOff, s.x1 - tx + topOff, 0, len(top)}
		} else {
			moved[i] = span{s.name, s.x0 - cut - bx + botOff, s.x1 - cut - bx + botOff, botY, len(out)}
		}
		moved[i].x0, moved[i].x1 = max(0, moved[i].x0), min(w, moved[i].x1)
	}
	return out, moved
}

// cropAt is crop, also returning where the kept area started.
func cropAt(g Grid) (Grid, int, int) {
	minX, minY := math.MaxInt, math.MaxInt
	for y, r := range g {
		for x := range len(r) {
			if r[x] != '.' {
				minX, minY = min(minX, x), min(minY, y)
			}
		}
	}
	return crop(g), minX, minY
}

// rowsOnly blanks every row outside [y0, y1); 0, 0 keeps them all.
func rowsOnly(g Grid, y0, y1 int) Grid {
	if y0 == 0 && y1 == 0 {
		return g
	}
	out := make(Grid, len(g))
	for i, r := range g {
		if i >= y0 && i < y1 {
			out[i] = r
		} else {
			out[i] = strings.Repeat(".", len(r))
		}
	}
	return out
}

func sliceGrid(g Grid, x0, x1 int) Grid {
	out := make(Grid, len(g))
	for i, r := range g {
		r += strings.Repeat(".", max(0, x1-len(r)))
		out[i] = r[x0:x1]
	}
	return out
}
