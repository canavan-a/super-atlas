import { useState } from 'react';

// On the web an invite arrives as /?invite=<code> (the server's /i/<code>
// landing page links there). It's read once and dropped from the address
// bar, so a reload doesn't ask again.
function takeInviteCode(): string | null {
  const params = new URLSearchParams(window.location.search);
  const code = params.get('invite');
  if (!code) return null;
  params.delete('invite');
  const rest = params.toString();
  window.history.replaceState(null, '', window.location.pathname + (rest ? `?${rest}` : '') + window.location.hash);
  return /^[a-z0-9]+$/i.test(code) ? code.toLowerCase() : null;
}

/** The invite code from the link that opened the page, if any. */
export function useIncomingInviteCode(): string | null {
  const [code] = useState(takeInviteCode);
  return code;
}
