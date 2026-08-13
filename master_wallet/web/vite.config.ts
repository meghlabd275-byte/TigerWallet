import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5180,
    host: true,
  },
  define: {
    'process.env.MASTER_WALLET_API_URL': JSON.stringify(
      process.env.MASTER_WALLET_API_URL || 'http://localhost:8450'
    ),
  },
});
