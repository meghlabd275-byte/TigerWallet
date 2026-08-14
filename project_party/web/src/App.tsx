// ProjectParty Web Application
import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ThemeProvider } from './contexts/ThemeContext';
import { AuthProvider } from './contexts/AuthContext';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Coins from './pages/Coins';
import Tokens from './pages/Tokens';
import Favorites from './pages/Favorites';
import Submit from './pages/Submit';
import Settings from './pages/Settings';
import Listings from './pages/Listings';
import Launchpad from './pages/Launchpad';
import MarketMaking from './pages/MarketMaking';
import Pricing from './pages/Pricing';
import Analytics from './pages/Analytics';
import Compliance from './pages/Compliance';
import Fees from './pages/Fees';
import Layout from './components/Layout';

function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<Login />} />
            <Route path="/" element={<Layout />}>
              <Route index element={<Navigate to="/dashboard" replace />} />
              <Route path="dashboard" element={<Dashboard />} />
              <Route path="coins" element={<Coins />} />
              <Route path="tokens" element={<Tokens />} />
              <Route path="favorites" element={<Favorites />} />
              <Route path="submit" element={<Submit />} />
              <Route path="listings" element={<Listings />} />
              <Route path="launchpad" element={<Launchpad />} />
              <Route path="market-making" element={<MarketMaking />} />
              <Route path="pricing" element={<Pricing />} />
              <Route path="analytics" element={<Analytics />} />
              <Route path="compliance" element={<Compliance />} />
              <Route path="fees" element={<Fees />} />
              <Route path="settings" element={<Settings />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </ThemeProvider>
  );
}

export default App;
