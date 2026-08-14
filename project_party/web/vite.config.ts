import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 8106,
    proxy: {
      '/api': 'http://localhost:8106'
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
});
