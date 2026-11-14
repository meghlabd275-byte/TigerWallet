import React from 'react';
import { HashRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ThemeProvider } from '../contexts/ThemeContext';
import { AuthProvider } from '../contexts/AuthContext';
import Layout from '../components/Layout';
import Login from '../pages/Login';
import Market from '../pages/Market';
import Tokens from '../pages/Tokens';
import TokenDetail from '../pages/TokenDetail';
import Listings from '../pages/Listings';
import Launchpad from '../pages/Launchpad';
import MarketMaking from '../pages/MarketMaking';
import Fees from '../pages/Fees';
import Favorites from '../pages/Favorites';
import Admin from '../pages/Admin';

function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <HashRouter>
          <Routes>
            <Route path="/login" element={<Login />} />
            <Route path="/" element={<Layout />}>
              <Route index element={<Navigate to="/market" replace />} />
              <Route path="market" element={<Market />} />
              <Route path="tokens" element={<Tokens />} />
              <Route path="tokens/:id" element={<TokenDetail />} />
              <Route path="listings" element={<Listings />} />
              <Route path="launchpad" element={<Launchpad />} />
              <Route path="market-making" element={<MarketMaking />} />
              <Route path="fees" element={<Fees />} />
              <Route path="favorites" element={<Favorites />} />
              <Route path="admin" element={<Admin />} />
            </Route>
          </Routes>
        </HashRouter>
      </AuthProvider>
    </ThemeProvider>
  );
}

export default App;
