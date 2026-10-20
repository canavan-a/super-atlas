import * as Linking from 'expo-linking';

// Pulls the invite code out of superatlas://invite/<code> or
// https://<server>/i/<code>.
export function inviteCodeFromURL(url: string | null): string | null {
  if (!url) return null;
  const match = url.match(/(?:\/\/invite\/|\/i\/)([a-z0-9]+)/i);
  return match ? match[1].toLowerCase() : null;
}

/** The invite code from the link that opened the app, if any. */
export function useIncomingInviteCode(): string | null {
  return inviteCodeFromURL(Linking.useLinkingURL());
}
