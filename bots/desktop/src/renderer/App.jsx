import React from 'react';
import { HashRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ThemeProvider } from '../contexts/ThemeContext';
import { AuthProvider } from '../contexts/AuthContext';
import Layout from '../components/Layout';
import Login from '../pages/Login';
import Dashboard from '../pages/Dashboard';
import Bots from '../pages/Bots';
import BotDetail from '../pages/BotDetail';
import Connectors from '../pages/Connectors';
import Subscriptions from '../pages/Subscriptions';
import Fees from '../pages/Fees';
import Settings from '../pages/Settings';

function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <HashRouter>
          <Routes>
            <Route path="/login" element={<Login />} />
            <Route path="/" element={<Layout />}>
              <Route index element={<Navigate to="/dashboard" replace />} />
              <Route path="dashboard" element={<Dashboard />} />
              <Route path="bots" element={<Bots />} />
              <Route path="bots/:id" element={<BotDetail />} />
              <Route path="connectors" element={<Connectors />} />
              <Route path="subscriptions" element={<Subscriptions />} />
              <Route path="fees" element={<Fees />} />
              <Route path="settings" element={<Settings />} />
            </Route>
          </Routes>
        </HashRouter>
      </AuthProvider>
    </ThemeProvider>
  );
}

export default App;
