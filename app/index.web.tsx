import ReactDOM from 'react-dom/client';
import 'maplibre-gl/dist/maplibre-gl.css';
import App from './App';
import { DELETION, DELETION_UPDATED } from './src/privacy/deletion';
import PrivacyPolicy, { PolicyPage } from './src/privacy/PrivacyPolicy';
import { ThemeProvider } from './src/theme';

const root = document.getElementById('root');
if (!root) {
  throw new Error('missing #root element');
}

// /privacy is the privacy policy as a page of its own, and /delete-account
// the steps to delete an account (both URLs given to Google Play), readable
// without signing in, as in super-chat.
const path = window.location.pathname.replace(/\/+$/, '');
const page =
  path === '/privacy' ? (
    <PrivacyPolicy />
  ) : path === '/delete-account' ? (
    <PolicyPage title="Delete your Super Atlas account" updated={DELETION_UPDATED} sections={DELETION} />
  ) : null;
if (path === '/privacy') document.title = 'Privacy policy · Super Atlas';
if (path === '/delete-account') document.title = 'Delete your account · Super Atlas';

ReactDOM.createRoot(root).render(page ? <ThemeProvider>{page}</ThemeProvider> : <App />);
