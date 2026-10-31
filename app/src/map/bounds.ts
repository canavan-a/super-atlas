// How much of each edge of the screen doesn't count as the middle: dragging
// the map so the person whose card is up ends up in this outer band (or off
// screen) closes their card. A small nudge keeps it.
const EDGE = 0.2;

/**
 * Whether a point is still in the middle of the map's visible
 * [west, south, east, north], away from the edges (EDGE), including a view
 * that spans the antimeridian (west > east).
 */
export function inMiddle([west, south, east, north]: [number, number, number, number], lng: number, lat: number) {
  const height = north - south;
  if (lat < south + height * EDGE || lat > north - height * EDGE) return false;
  const width = west <= east ? east - west : east + 360 - west;
  // Degrees east of the left edge, wrapped into [0, 360).
  const x = (((lng - west) % 360) + 360) % 360;
  return x >= width * EDGE && x <= width * (1 - EDGE);
}
