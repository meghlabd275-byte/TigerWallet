import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// WL-ProjectParty standalone backend (mapped :8464 externally, :8106 inside the
// container). Dev requests to /api and /health are proxied to it so the SPA
// never hardcodes a host — production nginx rewrites the same paths.
const WL_BACKEND = process.env.WL_BACKEND || 'http://localhost:8464';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 8106,
    proxy: {
      '/api': { target: WL_BACKEND, changeOrigin: true },
      '/health': { target: WL_BACKEND, changeOrigin: true }
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
});
