import { createContext, useContext } from 'react';
import type { PermissionsState } from './usePermissionsGate';

// RootNavigator owns the one usePermissionsGate(); screens read it from here
// (BackgroundLocationBanner).
export const PermissionsContext = createContext<PermissionsState | null>(null);

export function usePermissions(): PermissionsState | null {
  return useContext(PermissionsContext);
}
