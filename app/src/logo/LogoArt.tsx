import React, {memo} from 'react';
import {StyleSheet} from 'react-native';
import Svg, {Path} from 'react-native-svg';

import {LOGO, ToneRuns} from './logoData';

// The art's sub-pixels aren't square: in the terminal a sextant is half a cell
// wide and a third of a cell tall, and a cell is about twice as tall as wide,
// so a sub-pixel is 3 wide by 4 tall. The logo keeps that shape everywhere.
export const PIXEL_W = 3;
export const PIXEL_H = 4;

/** The boot scroll's size in viewBox units: every layer shares its grid. */
export const LOGO_W = LOGO.width * PIXEL_W;
export const LOGO_H = LOGO.height * PIXEL_H;

// Adjacent rectangles overlap by a hair so no seams show between pixels when
// the logo is scaled to a fractional size.
const OVERLAP = 0.35;

const pathCache = new WeakMap<ToneRuns, string>();

/** One tone's runs as a single SVG path of rectangles. */
export function tonePath(t: ToneRuns): string {
  const cached = pathCache.get(t);
  if (cached !== undefined) return cached;
  const parts: string[] = [];
  for (let i = 0; i < t.runs.length; i += 3) {
    const x = t.runs[i] * PIXEL_W;
    const y = t.runs[i + 1] * PIXEL_H;
    const w = t.runs[i + 2] * PIXEL_W + OVERLAP;
    parts.push(`M${x} ${y}h${w}v${PIXEL_H + OVERLAP}h${-w}z`);
  }
  const d = parts.join('');
  pathCache.set(t, d);
  return d;
}

/** The runs of each tone clipped to the columns [x0, x1). Tones left empty are dropped. */
export function sliceTones(tones: ToneRuns[], x0: number, x1: number): ToneRuns[] {
  const out: ToneRuns[] = [];
  for (const t of tones) {
    const runs: number[] = [];
    for (let i = 0; i < t.runs.length; i += 3) {
      const start = Math.max(t.runs[i], x0);
      const end = Math.min(t.runs[i] + t.runs[i + 2], x1);
      if (end > start) runs.push(start, t.runs[i + 1], end - start);
    }
    if (runs.length) out.push({u: t.u, w: t.w, runs});
  }
  return out;
}

/** How many columns of paper each strip of the unrolling reveal holds. */
export const STRIP_COLS = 4;

export interface PaperStrip {
  x0: number;
  x1: number;
  /** how far open (0..1) the scroll is when this strip starts to show, and when it's fully shown */
  from: number;
  to: number;
  text: ToneRuns[];
  map: ToneRuns[];
}

/**
 * The paper cut into strips from the middle out, each with the stretch of
 * the unrolling over which a roller uncovers it: open by t, the paper shows
 * LOGO.travel * t - LOGO.revealInset columns either side of the center.
 */
export const PAPER_STRIPS: PaperStrip[] = (() => {
  const out: PaperStrip[] = [];
  const {center, travel, revealInset, width} = LOGO;
  const at = (dist: number) => Math.min(1, Math.max(0, (dist + revealInset) / travel));
  for (let near = 0; near < width / 2; near += STRIP_COLS) {
    const far = near + STRIP_COLS;
    for (const [x0, x1] of [
      [center - far, center - near],
      [center + near, center + far],
    ]) {
      const text = sliceTones(LOGO.paperText, x0, x1);
      const map = sliceTones(LOGO.paperMap, x0, x1);
      if (!text.length && !map.length) continue;
      const from = at(near);
      out.push({x0, x1, from: Math.min(from, 0.999), to: Math.max(at(far), from + 0.001), text, map});
    }
  }
  return out;
})();

// Every layer sits at the origin of its container and overlays its siblings.
// (Without this the SVGs are flex children and *stack*: on the web an <svg>
// in a flex column also shrinks to share the height.)
const styles = StyleSheet.create({
  layer: {position: 'absolute', top: 0, left: 0},
});

interface LayerProps {
  tones: ToneRuns[];
  /** the fill color for a tone */
  fill: (t: ToneRuns) => string;
  /** the part of the logo to show: x, y, width, height in viewBox units */
  view: [number, number, number, number];
  /** rendered size in pixels */
  width: number;
  height: number;
}

/** A group of tones drawn as SVG. */
export const ToneLayer = memo(function ToneLayer({tones, fill, view, width, height}: LayerProps) {
  return (
    <Svg style={styles.layer} width={width} height={height} viewBox={view.join(' ')}>
      {tones.map(t => (
        <Path key={`${t.u}/${t.w}`} d={tonePath(t)} fill={fill(t)} />
      ))}
    </Svg>
  );
});
