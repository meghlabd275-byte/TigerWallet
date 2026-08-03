// TigerWallet Admin Panel - Main Application
// Complete React TypeScript Admin Dashboard with Light/Dark Theme

import React, { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { ThemeProvider } from './stores/ThemeStore';
import Sidebar from './components/Sidebar';
import Header from './components/Header';
import Dashboard from './pages/Dashboard';
import UsersPage from './pages/UsersPage';
import { WalletsPage, BlockchainPage, PairsPage, LiquidityPage, KYCPage, AnalyticsPage, SettingsPage, LoginPage, SendPage } from './pages/PageStubs';
import TradingManagementPage from './pages/TradingManagementPage';
import MarginTradingPage from './pages/MarginTradingPage';
import FeesPage from './pages/FeesPage';
import P2PTradingPage from './pages/P2PTradingPage';
import P2PMerchantPage from './pages/P2PMerchantPage';
import FiatOnRampPage from './pages/FiatOnRampPage';
import CryptoCardPage from './pages/CryptoCardPage';
import WhiteLabelPage from './pages/WhiteLabelPage';
import TransactionsPage from './pages/TransactionsPage';
import './styles/global.css';

const App: React.FC = () => {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    // Check authentication
    const token = localStorage.getItem('admin_token');
    if (token) {
      setIsAuthenticated(true);
    }
    setIsLoading(false);
  }, []);

  const handleLogin = (token: string) => {
    localStorage.setItem('admin_token', token);
    setIsAuthenticated(true);
  };

  const handleLogout = () => {
    localStorage.removeItem('admin_token');
    setIsAuthenticated(false);
  };

  if (isLoading) {
    return (
      <div className="loading-screen">
        <div className="spinner"></div>
        <p>Loading TigerWallet Admin...</p>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <LoginPage onLogin={handleLogin} />;
  }

  return (
    <ThemeProvider>
      <Router>
        <div className="admin-layout">
          <Sidebar />
          <div className="main-content">
            <Header onLogout={handleLogout} />
            <main className="page-content">
              <Routes>
                <Route path="/" element={<Dashboard />} />
                <Route path="/users" element={<UsersPage />} />
                <Route path="/wallets" element={<WalletsPage />} />
                <Route path="/blockchain" element={<BlockchainPage />} />
                <Route path="/pairs" element={<PairsPage />} />
                <Route path="/trading" element={<TradingManagementPage />} />
                <Route path="/margin" element={<MarginTradingPage />} />
                <Route path="/liquidity" element={<LiquidityPage />} />
                <Route path="/fees" element={<FeesPage />} />
                <Route path="/p2p" element={<P2PTradingPage />} />
                <Route path="/p2p-merchants" element={<P2PMerchantPage />} />
                <Route path="/fiat" element={<FiatOnRampPage />} />
                <Route path="/card" element={<CryptoCardPage />} />
                <Route path="/whitelabel" element={<WhiteLabelPage />} />
                <Route path="/kyc" element={<KYCPage />} />
                <Route path="/transactions" element={<TransactionsPage />} />
                <Route path="/analytics" element={<AnalyticsPage />} />
                <Route path="/settings" element={<SettingsPage />} />
                <Route path="/send" element={<SendPage />} />
                <Route path="*" element={<Navigate to="/" replace />} />
              </Routes>
            </main>
          </div>
        </div>
      </Router>
    </ThemeProvider>
  );
};

export default App;
