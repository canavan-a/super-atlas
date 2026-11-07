import type { LngLat } from './glide';

// Markers closer together than this on screen (in pixels, centre to centre)
// merge into one group marker. A marker is 34px across (30 plus its ring), so
// this is "touching, with a little room".
export const CLUSTER_PX = 44;
// Tapping a group zooms in until it comes apart, unless that would mean going
// in past this (people in the same building): then it lists them instead.
export const MAX_SPLIT_ZOOM = 18;
// MapLibre's world is 512px wide at zoom 0.
const TILE_PX = 512;

export type Cluster<T> = {
  /** Stable while the same people are in it: their ids, sorted. */
  key: string;
  /** In the order they were given. */
  members: T[];
  /** Where the group marker sits: the middle of its members. */
  lngLat: LngLat;
  /** The zoom at which every member stands on their own (Infinity when two
   *  are at the very same spot). */
  splitZoom: number;
};

/** A point's position in screen pixels on the whole world at `zoom`. */
function project([lng, lat]: LngLat, zoom: number): [number, number] {
  const scale = TILE_PX * 2 ** zoom;
  const s = Math.min(Math.max(Math.sin((lat * Math.PI) / 180), -0.9999), 0.9999);
  return [((lng + 180) / 360) * scale, (0.5 - Math.log((1 + s) / (1 - s)) / (4 * Math.PI)) * scale];
}

const dist = (a: [number, number], b: [number, number]) => Math.hypot(a[0] - b[0], a[1] - b[1]);

/**
 * Groups items whose markers would overlap at `zoom`: repeatedly merges the
 * two closest groups (by their centres on screen) while they're within
 * CLUSTER_PX. A lone item comes back as a group of one.
 */
export function clusterPoints<T>(items: T[], at: (t: T) => LngLat, id: (t: T) => string, zoom: number): Cluster<T>[] {
  const pts = items.map((t) => project(at(t), zoom));
  let groups = items.map((_, i) => ({ idx: [i], c: pts[i] }));

  for (;;) {
    let best: [number, number] | null = null;
    let bestD = CLUSTER_PX;
    for (let i = 0; i < groups.length; i++) {
      for (let j = i + 1; j < groups.length; j++) {
        const d = dist(groups[i].c, groups[j].c);
        if (d < bestD) {
          bestD = d;
          best = [i, j];
        }
      }
    }
    if (!best) break;
    const [i, j] = best;
    const idx = [...groups[i].idx, ...groups[j].idx].sort((a, b) => a - b);
    const c: [number, number] = [
      idx.reduce((s, k) => s + pts[k][0], 0) / idx.length,
      idx.reduce((s, k) => s + pts[k][1], 0) / idx.length,
    ];
    groups = groups.filter((_, k) => k !== i && k !== j);
    groups.push({ idx, c });
  }

  return groups.map(({ idx }) => {
    const members = idx.map((k) => items[k]);
    const coords = members.map(at);
    let closest = Infinity;
    for (let a = 0; a < idx.length; a++) {
      for (let b = a + 1; b < idx.length; b++) closest = Math.min(closest, dist(pts[idx[a]], pts[idx[b]]));
    }
    return {
      key: members.map(id).sort().join(','),
      members,
      lngLat: [
        coords.reduce((s, p) => s + p[0], 0) / coords.length,
        coords.reduce((s, p) => s + p[1], 0) / coords.length,
      ],
      // Screen distances double with each zoom level.
      splitZoom: members.length < 2 ? zoom : closest === 0 ? Infinity : zoom + Math.log2(CLUSTER_PX / closest),
    };
  });
}
