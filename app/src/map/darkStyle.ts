import type { StyleSpecification } from 'maplibre-gl';
import base from './ofm-dark.json';

// The dark themes' basemap. OpenFreeMap's "dark" style (ofm-dark.json, a
// copy of https://tiles.openfreemap.org/styles/dark) is near-black on
// near-black: water, parks, roads and labels all within a few shades of each
// other. This keeps its layers, tiles, fonts and sprites and recolors it per
// theme so land, water, green space, roads and labels read apart.

export interface MapPalette {
  land: string;
  residential: string;
  park: string;
  water: string;
  waterway: string;
  building: string;
  buildingOutline: string;
  path: string;
  roadMinor: string;
  roadMajor: string;
  roadCasing: string;
  motorway: string;
  rail: string;
  aeroway: string;
  boundary: string;
  label: string;
  roadLabel: string;
  waterLabel: string;
  halo: string;
}

type Paint = Record<string, unknown>;

function paints(p: MapPalette): Record<string, Paint> {
  const place = { 'text-color': p.label, 'text-halo-color': p.halo, 'text-halo-width': 1.4, 'text-halo-blur': 0.5 };
  const rail = { 'line-color': p.rail };
  const railDash = { 'line-color': p.land };
  return {
    background: { 'background-color': p.land },
    water: { 'fill-color': p.water },
    landcover_ice_shelf: { 'fill-color': p.residential },
    landcover_glacier: { 'fill-color': p.residential },
    landuse_residential: { 'fill-color': p.residential, 'fill-opacity': 1 },
    landcover_wood: { 'fill-color': p.park, 'fill-pattern': undefined },
    landuse_park: { 'fill-color': p.park },
    waterway: { 'line-color': p.waterway },
    water_name: { 'text-color': p.waterLabel, 'text-halo-color': p.halo, 'text-halo-width': 1.2 },
    building: { 'fill-color': p.building, 'fill-outline-color': p.buildingOutline },
    'aeroway-taxiway': { 'line-color': p.aeroway },
    'aeroway-runway-casing': { 'line-color': p.roadCasing },
    'aeroway-area': { 'fill-color': p.residential },
    'aeroway-runway': { 'line-color': p.aeroway },
    road_area_pier: { 'fill-color': p.roadMinor },
    road_pier: { 'line-color': p.roadMinor },
    highway_path: { 'line-color': p.path },
    highway_minor: { 'line-color': p.roadMinor, 'line-opacity': 1 },
    highway_major_casing: { 'line-color': p.roadCasing },
    highway_major_inner: { 'line-color': p.roadMajor },
    highway_major_subtle: { 'line-color': p.roadMajor },
    highway_motorway_casing: { 'line-color': p.roadCasing },
    highway_motorway_inner: { 'line-color': p.motorway },
    highway_motorway_subtle: { 'line-color': p.motorway },
    railway_transit: rail,
    railway_transit_dashline: railDash,
    railway_minor: rail,
    railway_minor_dashline: railDash,
    railway: rail,
    railway_dashline: railDash,
    highway_name_other: { 'text-color': p.roadLabel, 'text-halo-color': p.halo, 'text-halo-width': 1.2 },
    highway_name_motorway: { 'text-color': p.roadLabel, 'text-halo-color': p.halo, 'text-halo-width': 1.2 },
    boundary_state: { 'line-color': p.boundary },
    'boundary_country_z0-4': { 'line-color': p.boundary },
    'boundary_country_z5-': { 'line-color': p.boundary },
    place_other: place,
    place_suburb: place,
    place_village: place,
    place_town: place,
    place_city: place,
    place_city_large: place,
    place_state: place,
    place_country_other: place,
    place_country_minor: place,
    place_country_major: place,
  };
}

export function darkMapStyle(p: MapPalette): StyleSpecification {
  const overrides = paints(p);
  const style = base as unknown as StyleSpecification;
  return {
    ...style,
    layers: style.layers.map((layer) => {
      const o = overrides[layer.id];
      if (!o || !('paint' in layer)) return layer;
      const paint: Paint = { ...(layer.paint as Paint), ...o };
      for (const k of Object.keys(paint)) if (paint[k] === undefined) delete paint[k];
      return { ...layer, paint } as typeof layer;
    }),
  };
}
