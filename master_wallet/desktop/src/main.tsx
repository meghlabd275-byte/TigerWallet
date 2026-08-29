import React from 'react';
import { createRoot } from 'react-dom/client';
import MasterDesktopApp from './App';

// The C++ desktop shell can inject a backend URL before this bundle loads via
// <script>window.__MASTER_API_URL__ = "http://host:8450"</script>; App.tsx
// reads it (falling back to VITE_API_URL then http://localhost:8450).
const container = document.getElementById('root');
if (!container) throw new Error('#root element missing from index.html');

createRoot(container).render(
  <React.StrictMode>
    <MasterDesktopApp />
  </React.StrictMode>
);
