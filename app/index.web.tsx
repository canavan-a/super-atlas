import ReactDOM from 'react-dom/client';
import 'maplibre-gl/dist/maplibre-gl.css';
import App from './App';
import PrivacyPolicy from './src/privacy/PrivacyPolicy';
import { ThemeProvider } from './src/theme';

const root = document.getElementById('root');
if (!root) {
  throw new Error('missing #root element');
}

// /privacy is the privacy policy as a page of its own (the URL given to
// Google Play), readable without signing in, as in super-chat.
const isPrivacy = window.location.pathname.replace(/\/+$/, '') === '/privacy';
if (isPrivacy) document.title = 'Privacy policy · Super Atlas';

ReactDOM.createRoot(root).render(
  isPrivacy ? (
    <ThemeProvider>
      <PrivacyPolicy />
    </ThemeProvider>
  ) : (
    <App />
  )
);
