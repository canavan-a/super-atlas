# App Store listing

The App Store's version of `listing.md` (the Play listing). Graphics:
`appstore-header-3840x1646.png` and `appstore-search-3840x2560.png`,
generated alongside the Play ones (see `listing.md`), and `screenshots-ios/`,
1179×2556 like super-chat's (App Store Connect turned down 1320×2868), in
upload order, with no alpha channel.

The screenshots are of the web build at 393×852 at 3x,
signed in to a local dev server (`SUPERATLAS_DEV=1`) as "Alex" with a
position set and the "larp" fake friends on, and with the web-only "view
only" note hidden.

Privacy policy URL: `https://superatlas.lunargen.com/privacy`

Support URL: `https://superatlas.lunargen.com`

## App name (max 30 characters)

```
Super Atlas
```

## Subtitle (max 30 characters)

```
Friends on a map, privately
```

## Promotional text (max 170 characters)

Shown above the description; it can be changed at any time without a new
version.

```
See where your friends are on one map, and let them see you. Only friends you accept can see you, there's no location history, and your data is never sold.
```

## Keywords (max 100 characters)

Comma-separated, no spaces after commas. The name and subtitle are already
searched, so their words aren't repeated here.

```
location,share,find,family,where,nearby,tracker,ghost,locate,meetup,live,gps,people
```

## Description (max 4000 characters)

```
Super Atlas is the simplest way to see where your friends are. Add a friend, and you each appear on the other's map. That's it.

FRIENDS ON A MAP
• See your friends' latest locations on one map
• Tap a friend to see when they were last seen
• Add a profile photo, or friends see your initials
• Your location updates while you move, so friends see where you are without you opening the app

ONLY THE PEOPLE YOU CHOOSE
• Only friends you have accepted can see you; nobody else can
• Invite friends with a link, or add them by email
• Remove a friend at any time and they stop seeing you straight away
• Ghost mode hides you from everyone with one switch

PRIVATE BY DEFAULT
• We keep only your latest position: there is no location history
• No ads, no analytics, no tracking, and your data is never sold or shared
• Sign in with a one-time code sent to your email; no password, no phone number
• Delete your account and all its data from Settings at any time

MAKE IT YOURS
• Light, dark, slate, sepia and ember themes
• Open source and self-hostable: run your own server if you prefer

Read our privacy policy for exactly what our server keeps and for how long.
```

## App Review information

Sign-in required: yes. The review account signs in with a fixed code and is
never emailed (`reviewAccount` in the server's NixOS config):

- User name: `tester@lunargen.com`
- Password: `222222`

Notes:

```
Super Atlas signs in with a one-time code sent by email. For review, enter tester@lunargen.com on the sign-in screen, then the code 222222; no email is sent. A new account has no friends: tap the friends button (top right of the map), then "Invite a friend" to see the invite flow. Location is used only to show you to the friends you accept.
```
