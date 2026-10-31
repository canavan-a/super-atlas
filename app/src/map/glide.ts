import { useEffect, useRef, useState } from 'react';

export type LngLat = [number, number];

// How long a dot takes to slide to a new position. The camera following
// someone eases over the same time, linearly, so it stays on their dot.
export const GLIDE_MS = 1000;
// Farther than this (in degrees, ~5 km) a dot jumps instead: sliding across
// town after a long gap looks like travel that never happened.
const MAX_GLIDE_DEG = 0.05;

/** Whether a move from a to b is short enough to slide rather than jump. */
export function shouldGlide(a: LngLat, b: LngLat): boolean {
  return Math.abs(a[0] - b[0]) < MAX_GLIDE_DEG && Math.abs(a[1] - b[1]) < MAX_GLIDE_DEG;
}

/**
 * Slides from `from` to `to` over GLIDE_MS at a steady speed, calling onFrame
 * with each in-between position. Returns a function that stops it.
 */
export function glide(from: LngLat, to: LngLat, onFrame: (p: LngLat) => void): () => void {
  const start = Date.now();
  let frame = 0;
  const step = () => {
    const t = Math.min((Date.now() - start) / GLIDE_MS, 1);
    onFrame([from[0] + (to[0] - from[0]) * t, from[1] + (to[1] - from[1]) * t]);
    if (t < 1) frame = requestAnimationFrame(step);
  };
  frame = requestAnimationFrame(step);
  return () => cancelAnimationFrame(frame);
}

/**
 * A position that slides to each new `target` instead of jumping there,
 * starting from wherever it is shown now (mid-slide included).
 */
export function useGlide(target: LngLat): LngLat {
  const [shown, setShown] = useState(target);
  const shownRef = useRef(target);
  const [lng, lat] = target;

  useEffect(() => {
    const show = (p: LngLat) => {
      shownRef.current = p;
      setShown(p);
    };
    const from = shownRef.current;
    if (from[0] === lng && from[1] === lat) return;
    if (!shouldGlide(from, [lng, lat])) {
      show([lng, lat]);
      return;
    }
    return glide(from, [lng, lat], show);
  }, [lng, lat]);

  return shown;
}
