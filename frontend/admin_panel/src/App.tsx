import React from 'react'
import { BrowserRouter, Routes, Route, Navigate, useOutletContext } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import Layout, { useThemeContext } from './components/Layout'
import Dashboard from './pages/Dashboard'
import Users from './pages/Users'
import Pools from './pages/Pools'
import Bridges from './pages/Bridges'
import Transactions from './pages/Transactions'
import MarketMaker from './pages/MarketMaker'
import Bots from './pages/Bots'
import Chains from './pages/Chains'
import DEXs from './pages/DEXs'
import Fees from './pages/Fees'
import Treasury from './pages/Treasury'
import Security from './pages/Security'
import Analytics from './pages/Analytics'
import Settings from './pages/Settings'
import Support from './pages/Support'
import Integrations from './pages/Integrations'
import Compliance from './pages/Compliance'
import Notifications from './pages/Notifications'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
})

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Layout />}>
            <Route index element={<Navigate to="/dashboard" replace />} />
            <Route path="dashboard" element={<Dashboard />} />
            <Route path="users" element={<Users />} />
            <Route path="pools" element={<Pools />} />
            <Route path="bridges" element={<Bridges />} />
            <Route path="transactions" element={<Transactions />} />
            <Route path="market-maker" element={<MarketMaker />} />
            <Route path="bots" element={<Bots />} />
            <Route path="chains" element={<Chains />} />
            <Route path="dexs" element={<DEXs />} />
            <Route path="fees" element={<Fees />} />
            <Route path="treasury" element={<Treasury />} />
            <Route path="support" element={<SupportPageWrapper />} />
            <Route path="integrations" element={<IntegrationsPageWrapper />} />
            <Route path="compliance" element={<CompliancePageWrapper />} />
            <Route path="notifications" element={<NotificationsPageWrapper />} />
            <Route path="security" element={<Security />} />
            <Route path="analytics" element={<Analytics />} />
            <Route path="settings" element={<SettingsPageWrapper />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}

// Wrapper component to get theme context
function SettingsPageWrapper() {
  const { darkMode, toggleTheme } = useThemeContext();
  return <Settings darkMode={darkMode} onThemeToggle={toggleTheme} />;
}

function SupportPageWrapper() {
  const { darkMode } = useThemeContext();
  return <Support darkMode={darkMode} />;
}

function IntegrationsPageWrapper() {
  const { darkMode } = useThemeContext();
  return <Integrations darkMode={darkMode} />;
}

function CompliancePageWrapper() {
  const { darkMode } = useThemeContext();
  return <Compliance darkMode={darkMode} />;
}

function NotificationsPageWrapper() {
  const { darkMode } = useThemeContext();
  return <Notifications darkMode={darkMode} />;
}

export default App