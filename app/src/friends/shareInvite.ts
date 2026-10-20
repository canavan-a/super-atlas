import { Share } from 'react-native';

/** Opens the share sheet with the invite link. Returns a note to show, if any. */
export async function shareInvite(url: string): Promise<string | null> {
  await Share.share({ message: `Add me on Super Atlas so we can see each other on the map: ${url}` });
  return null;
}
