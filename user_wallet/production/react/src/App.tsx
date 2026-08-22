/**
 * TigerWallet - Production User Wallet Application
 *
 * No-registration self-custody model (mirrors web/src/App.tsx): the app opens
 * to a Create/Import choice (Onboarding page) when no wallet exists locally.
 * The user never sees a register/login form — a transparent ephemeral session
 * is provisioned behind the scenes (OnboardingContext) so the JWT-backed
 * wallet-api backend is satisfied. Once a wallet is created/imported, the full
 * multi-chain wallet experience unlocks.
 */

import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ThemeProvider, useTheme } from './contexts/ThemeContext';
import { WalletProvider } from './contexts/WalletContext';
import { AuthProvider } from './contexts/AuthContext';
import { OnboardingProvider, useOnboarding } from './contexts/OnboardingContext';

// Components
import Sidebar from './components/Sidebar';
import Header from './components/Header';
import LoadingSpinner from './components/LoadingSpinner';

// Pages
import Onboarding from './pages/Onboarding';
import HomePage from './pages/HomePage';
import WalletPage from './pages/WalletPage';
import SendPage from './pages/SendPage';
import ReceivePage from './pages/ReceivePage';
import SwapPage from './pages/SwapPage';
import DAppsPage from './pages/DAppsPage';
import SettingsPage from './pages/SettingsPage';
import StakingPage from './pages/StakingPage';
import NFTsPage from './pages/NFTsPage';
import HistoryPage from './pages/HistoryPage';
import BridgePage from './pages/BridgePage';

// Styles
import './styles/globals.css';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
      staleTime: 30000,
    },
  },
});

function AppLayout({ children }: { children: React.ReactNode }) {
  const { theme } = useTheme();

  return (
    <div className={`app-layout min-h-screen ${theme}`}>
      <div className="flex">
        <Sidebar />
        <div className="flex-1 ml-64">
          <Header />
          <main className="pt-16 min-h-screen">
            {children}
          </main>
        </div>
      </div>
    </div>
  );
}

function AppRoutes() {
  const { ready, onboarded } = useOnboarding();

  // Gate 1: session bootstrap in progress — show a boot screen (the user never
  // waits on a login form; the ephemeral session is provisioned transparently).
  if (!ready) {
    return (
      <div className="app-boot">
        <LoadingSpinner size="lg" />
        <span className="app-boot-text">Initializing secure wallet…</span>
      </div>
    );
  }

  // Gate 2: if no wallet exists locally, show the no-registration onboarding
  // (Create/Import). The user never reaches the dashboard without a wallet.
  // Onboarding is a full-screen flow (no sidebar/header), so it renders outside
  // the Router too.
  if (!onboarded) {
    return <Onboarding />;
  }

  return (
    <Router>
      <Routes>
        <Route path="/" element={<AppLayout><HomePage /></AppLayout>} />
        <Route path="/wallet" element={<AppLayout><WalletPage /></AppLayout>} />
        <Route path="/send" element={<AppLayout><SendPage /></AppLayout>} />
        <Route path="/receive" element={<AppLayout><ReceivePage /></AppLayout>} />
        <Route path="/swap" element={<AppLayout><SwapPage /></AppLayout>} />
        <Route path="/staking" element={<AppLayout><StakingPage /></AppLayout>} />
        <Route path="/nfts" element={<AppLayout><NFTsPage /></AppLayout>} />
        <Route path="/history" element={<AppLayout><HistoryPage /></AppLayout>} />
        <Route path="/bridge" element={<AppLayout><BridgePage /></AppLayout>} />
        <Route path="/dapps" element={<AppLayout><DAppsPage /></AppLayout>} />
        <Route path="/settings" element={<AppLayout><SettingsPage /></AppLayout>} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Router>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <OnboardingProvider>
          <AuthProvider>
            <WalletProvider>
              <AppRoutes />
            </WalletProvider>
          </AuthProvider>
        </OnboardingProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}

export default App;

