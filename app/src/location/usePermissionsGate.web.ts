import type { PermissionsState } from './usePermissionsGate';

// The web build never asks for location: it only shows friends.
export function usePermissionsGate(): PermissionsState {
  return {
    loading: false,
    foregroundLocationGranted: true,
    backgroundLocationGranted: true,
    allGranted: true,
    mustOpenSettings: false,
    requestAll: async () => {},
  };
}
