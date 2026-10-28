// How to delete a Super Atlas account, as its own page on the web
// (/delete-account, the account deletion URL given to Google Play). Google
// asks that it names the app, puts the steps first, and says what is deleted
// or kept and for how long. Keep it in step with auth.Store.DeleteUser and the
// privacy policy (policy.ts), and update DELETION_UPDATED when it changes.

import { CONTACT, type PolicySection } from './policy';

export const DELETION_UPDATED = '2026-10-28';

export const DELETION: PolicySection[] = [
  {
    title: 'Delete your account in the app',
    blocks: [
      {
        p: 'You can delete your Super Atlas account yourself at any time, from the Android app or on the web at superatlas.lunargen.com:',
      },
      {
        list: [
          '**1.** Open Super Atlas and sign in.',
          '**2.** On the map, tap the **Friends** button (top right).',
          '**3.** Tap the **gear** icon (top right) to open **Settings**.',
          '**4.** Under **Account**, tap **Delete account**, then **Delete** to confirm.',
        ],
      },
      {
        p: 'Your account is deleted straight away and every device you were signed in on is signed out. This cannot be undone.',
      },
    ],
  },
  {
    title: "If you can't sign in",
    blocks: [
      {
        p: `Email **${CONTACT}** from the address you signed up with and ask for your account to be deleted. We will delete it within 30 days and reply to confirm.`,
      },
    ],
  },
  {
    title: 'What is deleted',
    blocks: [
      {
        p: 'Everything our server holds about you is deleted immediately and permanently:',
      },
      {
        list: [
          'Your email address, display name and profile photo',
          'Your location (we only ever keep your latest position)',
          'Your friends list, friend requests and invite link',
          'Your sign-in sessions, pending sign-in codes and notification tokens',
          'Your settings, such as ghost mode',
        ],
      },
      {
        p: 'Your friends stop receiving your location at once. A map a friend already has open keeps your last position until it reloads.',
      },
    ],
  },
  {
    title: 'What is kept',
    blocks: [
      {
        list: [
          "**Server request logs.** Our server's log of requests (time, address requested and IP address) is not linked to your account and is not deleted on request. It is removed automatically as the system log rotates out old entries.",
          '**Nothing else.** We keep no backups or copies of account data after deletion.',
        ],
      },
      {
        p: 'Removing the app from your phone deletes the data it keeps on the device (your sign-in token, a copy of your name and email, and your settings), but does not delete your account. Use the steps above to do that.',
      },
    ],
  },
];
