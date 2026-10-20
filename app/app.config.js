// Extends app.json with settings that depend on the build's environment.
//
// Invite links look like https://<server>/i/<code>. When the app is built
// against an https server (EXPO_PUBLIC_API_URL), Android is told to open
// those links straight in the app; the server proves the app owns them via
// /.well-known/assetlinks.json (SUPERATLAS_ANDROID_CERT_SHA256). The
// superatlas://invite/<code> scheme works either way.
module.exports = ({ config }) => {
  const apiUrl = process.env.EXPO_PUBLIC_API_URL;
  let intentFilters = [];
  if (apiUrl && apiUrl.startsWith('https://')) {
    const host = new URL(apiUrl).host;
    intentFilters = [
      {
        action: 'VIEW',
        autoVerify: true,
        data: [{ scheme: 'https', host, pathPrefix: '/i/' }],
        category: ['BROWSABLE', 'DEFAULT'],
      },
    ];
  }

  return {
    ...config,
    android: {
      ...config.android,
      intentFilters,
    },
  };
};
