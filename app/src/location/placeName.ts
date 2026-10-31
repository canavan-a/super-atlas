import * as Location from 'expo-location';
import { useEffect, useState } from 'react';

// Names looked up so far, by position rounded to about 100 m (a friend sitting
// still drifts a little with GPS jitter; that shouldn't mean a new lookup).
// null: the phone found nothing there.
const cache = new Map<string, string | null>();
// When a lookup last failed (no network, rate-limited, no geocoder on the
// phone), by the same key: it's tried again once RETRY_MS has passed.
const failedAt = new Map<string, number>();
const RETRY_MS = 2 * 60 * 1000;
// Give up on a lookup that hasn't answered by then.
const TIMEOUT_MS = 10 * 1000;

const keyFor = (lat: number, lng: number) => `${lat.toFixed(3)},${lng.toFixed(3)}`;

/**
 * A short name for where a position is, from the phone's own geocoder
 * (Android's Geocoder; on phones with Google Play services it asks Google):
 * the business there ("Starbucks"), else the street ("Stirrup Drive area"),
 * else the neighborhood or town. null until it's known, or when there's
 * nothing to look up, no name for it, or the lookup failed; callers show
 * something else then. Never throws. (placeName.web.ts: always null.)
 */
export function usePlaceName(at: { latitude: number; longitude: number } | null): string | null {
  const key = at ? keyFor(at.latitude, at.longitude) : null;
  const [name, setName] = useState<string | null>(key ? (cache.get(key) ?? null) : null);

  useEffect(() => {
    if (!at || !key) {
      setName(null);
      return;
    }
    if (cache.has(key)) {
      setName(cache.get(key) ?? null);
      return;
    }
    setName(null);
    const failed = failedAt.get(key);
    if (failed !== undefined && Date.now() - failed < RETRY_MS) return;
    let live = true;
    lookUp(at.latitude, at.longitude).then(
      (label) => {
        failedAt.delete(key);
        cache.set(key, label);
        if (live) setName(label);
      },
      (e) => {
        failedAt.set(key, Date.now());
        if (__DEV__) console.warn('Place name lookup failed', e);
      }
    );
    return () => {
      live = false;
    };
  }, [key]); // eslint-disable-line react-hooks/exhaustive-deps

  return name;
}

// The geocoder's answer as a label, or null when it found nothing. Rejects on
// an error or after TIMEOUT_MS.
async function lookUp(latitude: number, longitude: number): Promise<string | null> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  const timeout = new Promise<never>((_, reject) => {
    timer = setTimeout(() => reject(new Error('timed out')), TIMEOUT_MS);
  });
  try {
    const found = await Promise.race([Location.reverseGeocodeAsync({ latitude, longitude }), timeout]);
    return found?.[0] ? labelFor(found[0]) : null;
  } finally {
    clearTimeout(timer);
  }
}

// Android's "name" (featureName) is a business or landmark when there is one,
// but otherwise usually just the house number or the street again, so it only
// counts when it has no digits and isn't one of the other parts.
function labelFor(a: Location.LocationGeocodedAddress): string | null {
  const name = a.name?.trim();
  if (name && !/\d/.test(name) && ![a.street, a.district, a.city, a.subregion, a.region].includes(name)) {
    return name;
  }
  if (a.street) return `${a.street} area`;
  return a.district ?? a.city ?? null;
}
