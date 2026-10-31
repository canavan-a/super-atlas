// The browser has no geocoder (expo-location's throws on web), so the map
// card keeps its time-based line there, and Settings has no "Place names"
// switch. See placeName.ts.
export const placeNamesSupported = false;

export function usePlaceNamesSetting(): [boolean, (on: boolean) => void] {
  return [false, () => {}];
}

export function usePlaceName(_at: { latitude: number; longitude: number } | null): string | null {
  return null;
}
