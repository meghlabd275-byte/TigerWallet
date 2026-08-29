import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// The GUI talks ONLY to the canonical MasterWallet backend (:8450). The base
// URL is resolved at runtime in src/App.tsx from VITE_API_URL /
// window.__MASTER_API_URL__ so the same dist bundle works embedded in the
// C++ shell and standalone.
export default defineConfig({
  plugins: [react()],
  base: './',
  server: { port: 8451 },
  build: { outDir: 'dist', sourcemap: false },
});
