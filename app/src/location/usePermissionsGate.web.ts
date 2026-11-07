import type { PermissionsState } from './usePermissionsGate';

// The web build never asks for location: it only shows friends.
export function usePermissionsGate(): PermissionsState {
  return {
    loading: false,
    showGate: false,
    foregroundLocationGranted: true,
    backgroundLocationGranted: true,
    preciseLocationOff: false,
    mustOpenSettings: false,
    backgroundMustOpenSettings: false,
    requestAll: async () => {},
    requestBackground: async () => {},
  };
}
