import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 9091,
    proxy: {
      '/api': 'http://localhost:8082'
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
});
