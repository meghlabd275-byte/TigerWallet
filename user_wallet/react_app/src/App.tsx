// TigerWallet User App - Main Application
// Complete Web3 Wallet similar to Trust Wallet / MetaMask
// Light/Dark theme works everywhere

import React, { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { ThemeProvider } from './stores/ThemeStore';
import Sidebar from './components/Sidebar';
import Header from './components/Header';
import HomePage from './pages/HomePage';
import WalletPage from './pages/WalletPage';
import SendPage from './pages/SendPage';
import ReceivePage from './pages/ReceivePage';
import SwapPage from './pages/SwapPage';
import ConvertPage from './pages/ConvertPage';
import FuturesTradingPage from './pages/FuturesTradingPage';
import CopyTradingPage from './pages/CopyTradingPage';
import OptionsTradingPage from './pages/OptionsTradingPage';
import RedPacketPage from './pages/RedPacketPage';
import ClaimPage from './pages/ClaimPage';
import DAppsPage from './pages/DAppsPage';
import SettingsPage from './pages/SettingsPage';
import LoginPage from './pages/LoginPage';
import './styles/global.css';

const App: React.FC = () => {
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  useEffect(() => {
    const hasWallet = localStorage.getItem('has_wallet');
    if (hasWallet === 'true') {
      setIsAuthenticated(true);
    }
  }, []);

  const handleLogin = () => {
    setIsAuthenticated(true);
  };

  const handleLogout = () => {
    localStorage.removeItem('has_wallet');
    setIsAuthenticated(false);
  };

  if (!isAuthenticated) {
    return <LoginPage onLogin={handleLogin} />;
  }

  return (
    <ThemeProvider>
      <Router>
        <div className="app-layout">
          <Sidebar />
          <div className="main-content">
            <Header onLogout={handleLogout} />
            <main className="page-content">
              <Routes>
                <Route path="/" element={<HomePage />} />
                <Route path="/wallet" element={<WalletPage />} />
                <Route path="/send" element={<SendPage />} />
                <Route path="/receive" element={<ReceivePage />} />
                <Route path="/swap" element={<SwapPage />} />
                <Route path="/convert" element={<ConvertPage />} />
                <Route path="/futures" element={<FuturesTradingPage />} />
                <Route path="/copy-trading" element={<CopyTradingPage />} />
                <Route path="/options" element={<OptionsTradingPage />} />
                <Route path="/red-packet" element={<RedPacketPage />} />
                <Route path="/claim" element={<ClaimPage />} />
                <Route path="/dapps" element={<DAppsPage />} />
                <Route path="/settings" element={<SettingsPage />} />
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
