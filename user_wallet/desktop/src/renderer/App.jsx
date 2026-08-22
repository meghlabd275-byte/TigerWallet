// UserWallet Desktop Application
//
// No-registration self-custody model: the app opens to a Create/Import choice
// (Onboarding page) when no wallet exists locally. The user never sees a
// register/login form — a transparent ephemeral session is provisioned behind
// the scenes (OnboardingContext) so the JWT-backed backend is satisfied.
// Once a wallet is created/imported, the full Dashboard/Transactions/Settings
// experience unlocks. Theme switching is global via ThemeProvider.
import React from 'react';
import { HashRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ThemeProvider } from '../contexts/ThemeContext';
import { AuthProvider } from '../contexts/AuthContext';
import { OnboardingProvider, useOnboarding } from '../contexts/OnboardingContext';
import Onboarding from '../pages/Onboarding';
import Layout from '../components/Layout';
import Dashboard from '../pages/Dashboard';
import Wallets from '../pages/Wallets';
import Transactions from '../pages/Transactions';
import Settings from '../pages/Settings';

function AppRoutes() {
  const { ready, onboarded } = useOnboarding();
  if (!ready) {
    return <div className="app-boot">Initializing secure wallet…</div>;
  }
  // Gate: if no wallet exists locally, show the no-registration onboarding
  // (Create/Import). The user never reaches the dashboard without a wallet.
  if (!onboarded) {
    return <Onboarding />;
  }
  return (
    <HashRouter>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<Dashboard />} />
          <Route path="wallets" element={<Wallets />} />
          <Route path="transactions" element={<Transactions />} />
          <Route path="settings" element={<Settings />} />
          <Route path="*" element={<Navigate to="/dashboard" replace />} />
        </Route>
      </Routes>
    </HashRouter>
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
