import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 9091,
    proxy: {
      '/api': 'http://localhost:8082',
      // License control plane (license_service). Requests to /license-api/* are
      // forwarded to http://localhost:8460 with the /license-api prefix stripped.
      '/license-api': {
        target: 'http://localhost:8460',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/license-api/, ''),
      },
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
});
