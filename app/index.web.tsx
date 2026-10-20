import ReactDOM from 'react-dom/client';
import 'maplibre-gl/dist/maplibre-gl.css';
import App from './App';

const root = document.getElementById('root');
if (!root) {
  throw new Error('missing #root element');
}

ReactDOM.createRoot(root).render(<App />);
