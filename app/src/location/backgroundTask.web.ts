// The web build never shares a location, so there's no background task.
export const BACKGROUND_LOCATION_TASK = 'superatlas-background-location';

export async function registerBackgroundLocationTask() {}

export async function unregisterBackgroundLocationTask() {}

export async function ensureBackgroundLocationTask() {}
