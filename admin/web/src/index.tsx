// TigerWallet Admin - Web Application
import React from 'react';
import { createRoot } from 'react-dom/client';
import AppComplete from './AppComplete';
import './styles/globals.css';

const container = document.getElementById('root');
if (container) {
  const root = createRoot(container);
  root.render(
    <React.StrictMode>
      <AppComplete />
    </React.StrictMode>
  );
}
