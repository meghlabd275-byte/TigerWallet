import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 8472,
    host: true,
    proxy: {
      '/api': {
        target: 'http://localhost:8463',
        changeOrigin: true,
      },
      '/health': {
        target: 'http://localhost:8463',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
});
