// UserWallet Web Application
import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ThemeProvider } from './contexts/ThemeContext';
import { AuthProvider } from './contexts/AuthContext';
import Login from './pages/Login';
import Register from './pages/Register';
import Dashboard from './pages/Dashboard';
import Wallets from './pages/Wallets';
import Transactions from './pages/Transactions';
import Settings from './pages/Settings';
import Send from './pages/Send';
import Receive from './pages/Receive';
import Swap from './pages/Swap';
import Staking from './pages/Staking';
import NFTs from './pages/NFTs';
import KYC from './pages/KYC';
import Bridge from './pages/Bridge';
import AddressBook from './pages/AddressBook';
import Devices from './pages/Devices';
import Approvals from './pages/Approvals';
import Keystore from './pages/Keystore';
import DeFi from './pages/DeFi';
import Layout from './components/Layout';

function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<Login />} />
            <Route path="/register" element={<Register />} />
            <Route path="/" element={<Layout />}>
              <Route index element={<Navigate to="/dashboard" replace />} />
              <Route path="dashboard" element={<Dashboard />} />
              <Route path="wallets" element={<Wallets />} />
              <Route path="send" element={<Send />} />
              <Route path="receive" element={<Receive />} />
              <Route path="swap" element={<Swap />} />
              <Route path="staking" element={<Staking />} />
              <Route path="kyc" element={<KYC />} />
              <Route path="nfts" element={<NFTs />} />
              <Route path="bridge" element={<Bridge />} />
              <Route path="defi" element={<DeFi />} />
              <Route path="address-book" element={<AddressBook />} />
              <Route path="devices" element={<Devices />} />
              <Route path="approvals" element={<Approvals />} />
              <Route path="keystore" element={<Keystore />} />
              <Route path="transactions" element={<Transactions />} />
              <Route path="settings" element={<Settings />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </ThemeProvider>
  );
}

export default App;
