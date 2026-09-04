// WL-ProjectParty Web Application
import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ThemeProvider } from './contexts/ThemeContext';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import Layout from './components/Layout';
import Login from './pages/Login';
import Register from './pages/Register';
import Dashboard from './pages/Dashboard';
import Tokens from './pages/Tokens';
import Submit from './pages/Submit';
import Listings from './pages/Listings';
import Launchpad from './pages/Launchpad';
import MarketMaking from './pages/MarketMaking';
import Fees from './pages/Fees';
import Admin from './pages/Admin';
import Favorites from './pages/Favorites';
import Settings from './pages/Settings';

function ProtectedLayout() {
  const { token } = useAuth();
  if (!token) return <Navigate to="/login" replace />;
  return <Layout />;
}

function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<Login />} />
            <Route path="/register" element={<Register />} />
            <Route path="/" element={<ProtectedLayout />}>
              <Route index element={<Navigate to="/dashboard" replace />} />
              <Route path="dashboard" element={<Dashboard />} />
              <Route path="tokens" element={<Tokens />} />
              <Route path="submit" element={<Submit />} />
              <Route path="listings" element={<Listings />} />
              <Route path="launchpad" element={<Launchpad />} />
              <Route path="market-making" element={<MarketMaking />} />
              <Route path="fees" element={<Fees />} />
              <Route path="admin" element={<Admin />} />
              <Route path="favorites" element={<Favorites />} />
              <Route path="settings" element={<Settings />} />
            </Route>
            <Route path="*" element={<Navigate to="/dashboard" replace />} />
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </ThemeProvider>
  );
}

export default App;
