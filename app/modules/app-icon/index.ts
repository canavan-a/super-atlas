import { requireOptionalNativeModule } from 'expo';

export interface AppIconModule {
  getIcon(): Promise<string>;
  setIcon(name: string): Promise<boolean>;
}

// Null on platforms (and builds) without the native module.
export default requireOptionalNativeModule<AppIconModule>('AppIcon');
