import type { ExpressionSpecification, FilterSpecification, LayerSpecification, StyleSpecification } from 'maplibre-gl';

// Businesses and house numbers. OpenFreeMap's tiles carry both (the
// OpenMapTiles "poi" and "housenumber" layers, from OpenStreetMap), but
// most of its styles draw neither: dark, fiord and positron have no business
// labels, and none of them draw house numbers. withPlaceLabels adds what a
// style is missing, in the theme's colors.

export interface PlaceLabelColors {
  // Business names and the dot under each.
  text: string;
  dot: string;
  halo: string;
  // House numbers, drawn fainter than names.
  number: string;
}

const NAME: ExpressionSpecification = ['coalesce', ['get', 'name_en'], ['get', 'name']];
const IS_POINT: ExpressionSpecification = ['match', ['geometry-type'], ['Point', 'MultiPoint'], true, false];

// Better-known places (a lower OpenMapTiles rank) show from further out.
// CLOSE_ZOOM (16) shows them all.
const POI_BANDS: {
  id: string;
  minzoom: number;
  rank: ExpressionSpecification;
}[] = [
  { id: 'r1', minzoom: 14, rank: ['<', ['get', 'rank'], 7] },
  {
    id: 'r7',
    minzoom: 15,
    rank: ['all', ['>=', ['get', 'rank'], 7], ['<', ['get', 'rank'], 20]],
  },
  { id: 'r20', minzoom: 16, rank: ['>=', ['get', 'rank'], 20] },
];

function poiLayers(c: PlaceLabelColors): {
  dots: LayerSpecification[];
  labels: LayerSpecification[];
} {
  const dots: LayerSpecification[] = [];
  const labels: LayerSpecification[] = [];
  for (const b of POI_BANDS) {
    const filter: FilterSpecification = ['all', IS_POINT, ['has', 'name'], b.rank];
    dots.push({
      id: `place_poi_dot_${b.id}`,
      type: 'circle',
      source: 'openmaptiles',
      'source-layer': 'poi',
      minzoom: b.minzoom,
      filter,
      paint: {
        'circle-color': c.dot,
        'circle-radius': ['interpolate', ['linear'], ['zoom'], 14, 2, 18, 3.5],
        'circle-stroke-color': c.halo,
        'circle-stroke-width': 1,
      },
    } as LayerSpecification);
    labels.push({
      id: `place_poi_label_${b.id}`,
      type: 'symbol',
      source: 'openmaptiles',
      'source-layer': 'poi',
      minzoom: b.minzoom,
      filter,
      layout: {
        'text-field': NAME,
        'text-font': ['Noto Sans Regular'],
        'text-size': 11,
        'text-max-width': 8,
        'text-anchor': 'top',
        'text-offset': [0, 0.5],
      },
      paint: {
        'text-color': c.text,
        'text-halo-color': c.halo,
        'text-halo-width': 1.2,
        'text-halo-blur': 0.5,
      },
    } as LayerSpecification);
  }
  return { dots, labels };
}

function houseNumbers(c: PlaceLabelColors): LayerSpecification {
  return {
    id: 'place_housenumber',
    type: 'symbol',
    source: 'openmaptiles',
    'source-layer': 'housenumber',
    minzoom: 17,
    layout: {
      'text-field': ['get', 'housenumber'],
      'text-font': ['Noto Sans Regular'],
      'text-size': 10,
    },
    paint: {
      'text-color': c.number,
      'text-halo-color': c.halo,
      'text-halo-width': 1,
    },
  } as LayerSpecification;
}

function sourceLayer(l: LayerSpecification): string | undefined {
  return (l as { 'source-layer'?: string })['source-layer'];
}

export function withPlaceLabels(style: StyleSpecification, c: PlaceLabelColors): StyleSpecification {
  const layers = [...style.layers];
  const hasPois = layers.some((l) => sourceLayer(l) === 'poi');

  // MapLibre places the topmost labels first and drops any later one that
  // would overlap, so house numbers go under every other label (they give
  // way to all of them) and business names just under the town and city
  // names. The dots go above the roads and buildings, under the labels.
  let lastShape = -1;
  layers.forEach((l, i) => {
    if (l.type !== 'symbol') lastShape = i;
  });
  const extra = hasPois ? { dots: [], labels: [] } : poiLayers(c);
  layers.splice(lastShape + 1, 0, ...extra.dots, houseNumbers(c));

  if (hasPois) {
    // The style draws businesses with its own icons; show them from one zoom
    // further out than it does, as ours are.
    return {
      ...style,
      layers: layers.map((l) =>
        sourceLayer(l) === 'poi' && l.minzoom !== undefined ? { ...l, minzoom: Math.max(l.minzoom - 1, 14) } : l,
      ),
    };
  }
  const firstPlace = layers.findIndex((l) => sourceLayer(l) === 'place');
  layers.splice(firstPlace === -1 ? layers.length : firstPlace, 0, ...extra.labels);
  return { ...style, layers };
}
