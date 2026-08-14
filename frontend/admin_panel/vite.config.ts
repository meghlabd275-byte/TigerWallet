import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Admin panel dev-server config. API calls to /api/v1/* are proxied to the
// canonical wallet_api (Go) so the standalone Vite dev server talks to the real
// PostgreSQL-backed admin endpoints. In production the same /api/v1/* paths are
// served by the reverse proxy in front of the backend.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 3002,
    proxy: {
      // Bot management is served by the dedicated bots_service (:8461).
      '/api/v1/admin/bots': {
        target: process.env.BOTS_SERVICE_URL || 'http://localhost:8461',
        changeOrigin: true,
      },
      // Everything else under /api proxies to the canonical wallet_api.
      '/api': {
        target: process.env.BACKEND_URL || 'http://localhost:8443',
        changeOrigin: true,
      },
    },
  },
});
