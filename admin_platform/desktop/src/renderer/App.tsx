import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ThemeProvider, useTheme } from './context/ThemeContext';
import Layout from './components/Layout';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Users from './pages/Users';
import Transactions from './pages/Transactions';
import Tokens from './pages/Tokens';
import Pairs from './pages/Pairs';
import KYC from './pages/KYC';
import Withdrawals from './pages/Withdrawals';
import Chains from './pages/Chains';
import Fees from './pages/Fees';
import WhiteLabels from './pages/WhiteLabels';

const queryClient = new QueryClient();

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const token = localStorage.getItem('admin_token');
  
  if (!token) {
    return <Navigate to="/login" replace />;
  }
  
  return <>{children}</>;
}

function AppRoutes() {
  const { isDark } = useTheme();
  
  return (
    <div className={isDark ? 'dark' : ''}>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route
          path="/*"
          element={
            <ProtectedRoute>
              <Layout>
                <Routes>
                  <Route path="/" element={<Navigate to="/dashboard" replace />} />
                  <Route path="/dashboard" element={<Dashboard />} />
                  <Route path="/users" element={<Users />} />
                  <Route path="/transactions" element={<Transactions />} />
                  <Route path="/tokens" element={<Tokens />} />
                  <Route path="/pairs" element={<Pairs />} />
                  <Route path="/kyc" element={<KYC />} />
                  <Route path="/withdrawals" element={<Withdrawals />} />
                  <Route path="/chains" element={<Chains />} />
                  <Route path="/fees" element={<Fees />} />
                  <Route path="/whitelabels" element={<WhiteLabels />} />
                </Routes>
              </Layout>
            </ProtectedRoute>
          }
        />
      </Routes>
    </div>
  );
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <BrowserRouter>
          <AppRoutes />
        </BrowserRouter>
      </ThemeProvider>
    </QueryClientProvider>
  );
}
