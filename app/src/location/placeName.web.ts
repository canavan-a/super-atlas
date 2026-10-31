// The browser has no geocoder (expo-location's throws on web), so the map
// card keeps its time-based line there. See placeName.ts.
export function usePlaceName(_at: { latitude: number; longitude: number } | null): string | null {
  return null;
}
