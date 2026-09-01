// UserWallet Web Application
//
// No-registration self-custody model: the app opens to a Create/Import choice
// (Onboarding page) when no wallet exists locally. The user never sees a
// register/login form — a transparent ephemeral session is provisioned behind
// the scenes (OnboardingContext) so the JWT-backed WL backend is satisfied.
// Once a wallet is created/imported, the full Dashboard/Transactions/Send/
// Settings experience unlocks. Theme switching is global via ThemeProvider.
import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ThemeProvider } from './contexts/ThemeContext';
import { AuthProvider } from './contexts/AuthContext';
import { OnboardingProvider, useOnboarding } from './contexts/OnboardingContext';
import Onboarding from './pages/Onboarding';
import Dashboard from './pages/Dashboard';
import Wallets from './pages/Wallets';
import Transactions from './pages/Transactions';
import Settings from './pages/Settings';
import Send from './pages/Send';
import Receive from './pages/Receive';
import Swap from './pages/Swap';
import Staking from './pages/Staking';
import KYC from './pages/KYC';
import NFTs from './pages/NFTs';
import Bridge from './pages/Bridge';
import DeFi from './pages/DeFi';
import AddressBook from './pages/AddressBook';
import Devices from './pages/Devices';
import Approvals from './pages/Approvals';
import Keystore from './pages/Keystore';
import Trading from './pages/Trading';
import Launchpool from './pages/Launchpool';
import TokenSales from './pages/TokenSales';
import P2P from './pages/P2P';
import Cards from './pages/Cards';
import PriceAlerts from './pages/PriceAlerts';
import DApps from './pages/DApps';
import Fees from './pages/Fees';
import Finance from './pages/Finance';
import DAO from './pages/DAO';
import Ramp from './pages/Ramp';
import CopyTrading from './pages/CopyTrading';
import Prediction from './pages/Prediction';
import ENS from './pages/ENS';
import Security from './pages/Security';
import Terminal from './pages/Terminal';
import Multisig from './pages/Multisig';
import NonEvm from './pages/NonEvm';
import Layout from './components/Layout';

function AppRoutes() {
  const { ready, onboarded } = useOnboarding();
  if (!ready) {
    return (
      <div className="app-boot">Initializing secure wallet…</div>
    );
  }
  // Gate: if no wallet exists locally, show the no-registration onboarding
  // (Create/Import). The user never reaches the dashboard without a wallet.
  if (!onboarded) {
    return <Onboarding />;
  }
  return (
    <BrowserRouter>
      <Routes>
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
          <Route path="trading" element={<Trading />} />
          <Route path="launchpool" element={<Launchpool />} />
          <Route path="token-sales" element={<TokenSales />} />
          <Route path="p2p" element={<P2P />} />
          <Route path="cards" element={<Cards />} />
          <Route path="price-alerts" element={<PriceAlerts />} />
          <Route path="dapps" element={<DApps />} />
          <Route path="fees" element={<Fees />} />
          <Route path="finance" element={<Finance />} />
          <Route path="dao" element={<DAO />} />
          <Route path="ramp" element={<Ramp />} />
          <Route path="copy-trading" element={<CopyTrading />} />
          <Route path="prediction" element={<Prediction />} />
          <Route path="ens" element={<ENS />} />
          <Route path="security" element={<Security />} />
          <Route path="terminal" element={<Terminal />} />
          <Route path="multisig" element={<Multisig />} />
          <Route path="non-evm" element={<NonEvm />} />
          <Route path="transactions" element={<Transactions />} />
          <Route path="settings" element={<Settings />} />
          <Route path="*" element={<Navigate to="/dashboard" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

function App() {
  return (
    <ThemeProvider>
      <OnboardingProvider>
        <AuthProvider>
          <AppRoutes />
        </AuthProvider>
      </OnboardingProvider>
    </ThemeProvider>
  );
}

export default App;
