/**
 * The browser's share sheet where there is one (phones), otherwise the link
 * is copied. Returns a note to show, if any.
 */
export async function shareInvite(url: string): Promise<string | null> {
  const text = `Add me on Super Atlas so we can see each other on the map: ${url}`;
  if (navigator.share) {
    try {
      await navigator.share({ text });
      return null;
    } catch (e: any) {
      if (e?.name === 'AbortError') return null; // closed the sheet
    }
  }
  await navigator.clipboard.writeText(url);
  return 'Invite link copied. Send it to a friend.';
}
