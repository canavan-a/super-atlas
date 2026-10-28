// In-app updates for the sideloaded Android APK, ported from super-badger's
// appUpdater.ts. There's no app store: "check for updates" means looking at
// this repo's GitHub Releases, and "install" means downloading the APK and
// handing it to Android's package installer (see AppUpdaterModule.kt).
import { NativeEventEmitter, NativeModules } from 'react-native';

export interface VersionInfo {
  versionName: string;
  versionCode: number;
  packageName: string;
}

export interface DownloadProgress {
  bytesDownloaded: number;
  bytesTotal: number;
}

// AppUpdaterModule.kt, in the sideload build only.
interface AppUpdaterNative {
  getVersionInfo(): Promise<VersionInfo>;
  canInstallPackages(): Promise<boolean>;
  openInstallPermissionSettings(): Promise<void>;
  downloadApk(url: string, versionLabel: string): Promise<string>;
  installApk(path: string): Promise<void>;
}

const AppUpdater = NativeModules.AppUpdater as AppUpdaterNative | undefined;

const REPO = 'canavan-a/super-atlas';
const TAG_PREFIX = 'app-v';
// release/release.sh uploads the APK as superatlas-app-v<version>.apk.
const APK_NAME = /^superatlas.*\.apk$/;

export const updatesSupported = AppUpdater != null;

function native() {
  if (!AppUpdater) throw new Error('Updates are only available in the Android app');
  return AppUpdater;
}

export const getVersionInfo = (): Promise<VersionInfo> => native().getVersionInfo();

export function onDownloadProgress(cb: (p: DownloadProgress) => void): () => void {
  if (!AppUpdater) return () => {};
  const sub = new NativeEventEmitter(NativeModules.AppUpdater).addListener('AppUpdaterProgress', cb);
  return () => sub.remove();
}

export interface Release {
  tag: string;
  version: string;
  apkUrl: string;
  publishedAt: string;
}

interface GhRelease {
  tag_name: string;
  published_at: string;
  draft: boolean;
  assets: { name: string; browser_download_url: string }[];
}

export async function fetchLatestRelease(): Promise<Release | null> {
  const res = await fetch(`https://api.github.com/repos/${REPO}/releases?per_page=30`, {
    headers: { Accept: 'application/vnd.github+json' },
  });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new Error(body?.message ?? `${res.status} ${res.statusText}`);
  }
  const releases = ((await res.json()) as GhRelease[])
    .filter((r) => !r.draft && r.tag_name.startsWith(TAG_PREFIX))
    .map((r): Release | null => {
      const apk = r.assets.find((a) => APK_NAME.test(a.name));
      if (!apk) return null;
      return {
        tag: r.tag_name,
        version: r.tag_name.slice(TAG_PREFIX.length),
        apkUrl: apk.browser_download_url,
        publishedAt: r.published_at,
      };
    })
    .filter((r): r is Release => r !== null)
    .sort((a, b) => compareVersions(b.version, a.version));
  return releases[0] ?? null;
}

// Returns > 0 if a is newer than b, < 0 if older, 0 if equal. Non-numeric
// segments (e.g. "1.2.0-rc1") compare lexically as a tie-breaker.
export function compareVersions(a: string, b: string): number {
  const pa = a.split(/[.-]/);
  const pb = b.split(/[.-]/);
  for (let i = 0; i < Math.max(pa.length, pb.length); i++) {
    const na = Number(pa[i]);
    const nb = Number(pb[i]);
    if (!Number.isNaN(na) && !Number.isNaN(nb)) {
      if (na !== nb) return na - nb;
    } else {
      const sa = pa[i] ?? '';
      const sb = pb[i] ?? '';
      if (sa !== sb) return sa < sb ? -1 : 1;
    }
  }
  return 0;
}

// Whether Android lets this app install APKs ("Install unknown apps"), and
// the settings page that allows it.
export const canInstallUpdates = (): Promise<boolean> => native().canInstallPackages();
export const openInstallSettings = (): Promise<void> => native().openInstallPermissionSettings();

// Each version's downloaded APK, so backing out of the installer and tapping
// Install again opens it straight away instead of downloading it again.
const downloaded = new Map<string, string>();

// Download (once per version), then hand off to the installer. Callers check
// canInstallUpdates first.
export async function installRelease(release: Release): Promise<void> {
  const n = native();
  let path = downloaded.get(release.version);
  if (!path) {
    path = await n.downloadApk(release.apkUrl, release.version);
    downloaded.set(release.version, path);
  }
  try {
    await n.installApk(path);
  } catch (e) {
    downloaded.delete(release.version); // e.g. the file was cleared: download it again next time
    throw e;
  }
}
