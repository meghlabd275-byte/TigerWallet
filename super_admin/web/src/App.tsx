/**
 * TigerWallet Super Admin - Main Application
 * Complete web frontend with all functionalities
 */

import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ThemeProvider } from './context/ThemeContext';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import Users from './pages/Users';
import KYC from './pages/KYC';
import Transactions from './pages/Transactions';
import Withdrawals from './pages/Withdrawals';
import Tokens from './pages/Tokens';
import Blockchains from './pages/Blockchains';
import TradingPairs from './pages/TradingPairs';
import Fees from './pages/Fees';
import WhiteLabels from './pages/WhiteLabels';
import Governance from './pages/Governance';
import Bots from './pages/Bots';
import BotsClients from './pages/BotsClients';
import ProjectTeams from './pages/ProjectTeams';
import MasterWallets from './pages/MasterWallets';
import UserWallets from './pages/UserWallets';
import Admins from './pages/Admins';
import Tickets from './pages/Tickets';
import KnowledgeBase from './pages/KnowledgeBase';
import Workflows from './pages/Workflows';
import Reports from './pages/Reports';
import Security from './pages/Security';
import APIKeys from './pages/APIKeys';
import Webhooks from './pages/Webhooks';
import AuditLogs from './pages/AuditLogs';
import System from './pages/System';
import Settings from './pages/Settings';
import Login from './pages/Login';
import Futures from './pages/Futures';
import Options from './pages/Options';
import CopyTrading from './pages/CopyTrading';
import Convert from './pages/Convert';
import OnRamp from './pages/OnRamp';
import OffRamp from './pages/OffRamp';
import P2PClients from './pages/P2PClients';
import P2PMerchants from './pages/P2PMerchants';
import Partners from './pages/Partners';
import Rewards from './pages/Rewards';
import Marketing from './pages/Marketing';
import AdminRoles from './pages/AdminRoles';
import './styles/globals.css';

// Protected Route Component
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const token = typeof window !== 'undefined' ? localStorage.getItem('super_admin_token') : null;
  
  if (!token) {
    return <Navigate to="/login" replace />;
  }
  
  return <>{children}</>;
}

function App() {
  return (
    <ThemeProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          
          <Route path="/" element={
            <ProtectedRoute>
              <Layout />
            </ProtectedRoute>
          }>
            <Route index element={<Dashboard />} />
            <Route path="users" element={<Users />} />
            <Route path="kyc" element={<KYC />} />
            <Route path="transactions" element={<Transactions />} />
            <Route path="withdrawals" element={<Withdrawals />} />
            <Route path="tokens" element={<Tokens />} />
            <Route path="blockchains" element={<Blockchains />} />
            <Route path="pairs" element={<TradingPairs />} />
            <Route path="fees" element={<Fees />} />
            <Route path="whitelabels" element={<WhiteLabels />} />
            <Route path="governance" element={<Governance />} />
            <Route path="bots" element={<Bots />} />
            <Route path="bots-clients" element={<BotsClients />} />
            <Route path="project-teams" element={<ProjectTeams />} />
            <Route path="master-wallets" element={<MasterWallets />} />
            <Route path="user-wallets" element={<UserWallets />} />
            <Route path="admins" element={<Admins />} />
            <Route path="tickets" element={<Tickets />} />
            <Route path="knowledge-base" element={<KnowledgeBase />} />
            <Route path="workflows" element={<Workflows />} />
            <Route path="reports" element={<Reports />} />
            <Route path="security" element={<Security />} />
            <Route path="api-keys" element={<APIKeys />} />
            <Route path="webhooks" element={<Webhooks />} />
            <Route path="audit-logs" element={<AuditLogs />} />
            <Route path="system" element={<System />} />
            <Route path="settings" element={<Settings />} />
            <Route path="futures" element={<Futures />} />
            <Route path="options" element={<Options />} />
            <Route path="copy-trading" element={<CopyTrading />} />
            <Route path="convert" element={<Convert />} />
            <Route path="onramp" element={<OnRamp />} />
            <Route path="offramp" element={<OffRamp />} />
            <Route path="p2p-clients" element={<P2PClients />} />
            <Route path="p2p-merchants" element={<P2PMerchants />} />
            <Route path="partners" element={<Partners />} />
            <Route path="rewards" element={<Rewards />} />
            <Route path="marketing" element={<Marketing />} />
            <Route path="admin-roles" element={<AdminRoles />} />
          </Route>
          
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </ThemeProvider>
  );
}

export default App;
