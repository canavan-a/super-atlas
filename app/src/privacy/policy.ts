// The privacy policy, shown in the app (Settings → Privacy policy) and as its
// own page on the web (/privacy, the URL given to Google Play), as in
// super-chat. Keep it in step with what the server and app actually do, and
// update UPDATED when it changes.

export const UPDATED = '2026-10-25';
export const CONTACT = 'aidan@lunargen.com';

/** Inline **bold** is the only markup, in paragraphs and list items. */
export type PolicyBlock = { p: string } | { list: string[] };

export interface PolicySection {
  title: string;
  blocks: PolicyBlock[];
}

export const POLICY: PolicySection[] = [
  {
    title: 'Summary',
    blocks: [
      {
        p: 'Super Atlas shows your location on a map to the friends you choose, and theirs to you. To do that, our server keeps your email address, the name you pick, your friends list and your most recent location. Nothing else.',
      },
      {
        p: 'We do not sell, rent or share your data, and we never will. There are no ads, no analytics and no tracking, and your location is never used for anything but showing it to your friends.',
      },
    ],
  },
  {
    title: 'What we collect',
    blocks: [
      {
        list: [
          '**Email address.** Used to sign you in with a one-time code and for friends to find you. We never send you anything else.',
          '**Display name.** The name you choose when you sign up, shown to your friends and to anyone you send a friend request or invite link.',
          '**Location.** Your device sends its position while the app is open. If you allow "Allow all the time", it also sends it every 15 seconds or so while you\'re moving (Android shows a notification while it does) and every 15 minutes or so otherwise. The server keeps **only your most recent position** (and how accurate it was), replacing it with each update. There is no location history.',
          "**Friends.** Who you're friends with and any pending friend requests, plus your personal invite link, which you can reset at any time.",
          '**Sign-in sessions.** One per signed-in device, stored only as a scrambled (hashed) token.',
          '**Notification token (Google Play version only).** A device token from Google, so we can tell you about friend requests while the app is closed.',
        ],
      },
      {
        p: 'We do not collect your contacts, photos, phone number, advertising identifiers, analytics or crash reports.',
      },
    ],
  },
  {
    title: 'Who sees your location',
    blocks: [
      {
        list: [
          'Only people you have accepted as friends. Nobody else, including other users of the app, can see where you are.',
          '**Ghost mode** (on the map) hides you from all your friends at once, until you turn it off.',
          'Removing a friend stops them receiving your location straight away (a map they already have open keeps your last position until it reloads).',
          'Your friends see your most recent position even when it is old (the map marks it as stale), until you turn on ghost mode or remove them. Signing out stops new updates but does not remove the last one.',
        ],
      },
    ],
  },
  {
    title: 'Outside services',
    blocks: [
      {
        p: 'A few outside services are involved. None of them receive your location history, and we do not share your data with anyone else.',
      },
      {
        list: [
          "**Email provider.** Sign-in codes are sent through our email provider, which sees your email address and the code.",
          '**Google Firebase Cloud Messaging (Google Play version only).** Friend notifications ("<name> wants to be friends") are delivered through Google, which sees the notification text and your device token. The version installed outside Google Play does not use Google services at all; it keeps its own connection to our server instead.',
          '**OpenFreeMap (map tiles).** The map pictures are loaded from tiles.openfreemap.org, which sees your IP address and which parts of the map you look at, but not your location or account.',
          '**GitHub (version installed outside Google Play only).** "Check for updates" downloads new versions from GitHub, which sees your IP address.',
          "**Google Play** handles downloads and updates of the Google Play version under Google's own privacy policy.",
        ],
      },
    ],
  },
  {
    title: 'Server logs and security',
    blocks: [
      {
        p: "Our server's request log records the time, the address requested and your IP address, to keep the service running and stop abuse. It is kept only in the server's system log, which removes old entries over time. IP addresses are also held briefly in memory for rate limiting.",
      },
      {
        p: 'All connections to our server are encrypted (HTTPS). Sign-in codes expire after 10 minutes and are only ever stored scrambled. A device that goes unused for 90 days is signed out.',
      },
    ],
  },
  {
    title: 'Data on your device',
    blocks: [
      {
        p: "The app keeps your sign-in token in the device's secure storage (in your browser's storage on the web), a copy of your name and email so the app opens without a connection, and your settings, such as the theme. Removing the app removes them all; signing out removes the token.",
      },
    ],
  },
  {
    title: 'How long data is kept, and deleting it',
    blocks: [
      {
        list: [
          '**Location:** only your latest position is kept, and each update replaces the last.',
          '**Sign-in codes:** deleted once expired. **Sessions** and their notification tokens: deleted when you sign out, or after 90 days unused.',
          '**Everything else** (email, name, friends, latest location) is kept while you have an account.',
          `**Deleting your account:** Settings → Delete account deletes your account and all its data from the server straight away: your email, name, friends and friend requests, location, sessions and notification tokens. Your friends stop receiving your location (a map already open keeps your last position until it reloads), and every device you were signed in on is signed out. If you can't sign in any more, email ${CONTACT} from the address you signed up with and we will do it for you within 30 days.`,
        ],
      },
    ],
  },
  {
    title: 'Children',
    blocks: [
      {
        p: "Super Atlas is not directed at children under 13, and we don't knowingly collect data from them. If you think a child has an account, contact us and we will delete it.",
      },
    ],
  },
  {
    title: 'Other servers',
    blocks: [
      {
        p: "This policy covers our server at superatlas.lunargen.com. Anyone can run their own Super Atlas server; an app pointed at another server is subject to that server operator's practices.",
      },
    ],
  },
  {
    title: 'Changes',
    blocks: [
      {
        p: "If this policy changes, we'll update the date at the top and describe the change in the app's release notes.",
      },
    ],
  },
  {
    title: 'Contact',
    blocks: [{ p: `Questions about privacy, or to delete your account: ${CONTACT}.` }],
  },
];
