// TigerWallet Admin - Web App Main Component
import React, { useState } from 'react';
import { ThemeProvider, useTheme } from './contexts/ThemeContext';
import './styles/globals.css';

// Import Pages
import LoginPage from './pages/LoginPage';
import DashboardPage from './pages/DashboardPage';
import UsersPage from './pages/UsersPage';
import TransactionsPage from './pages/TransactionsPage';
import KycPage from './pages/KycPage';
import TokensPage from './pages/TokensPage';
import FeesPage from './pages/FeesPage';
import SystemPage from './pages/SystemPage';
import SettingsPage from './pages/SettingsPage';
import WithdrawalsPage from './pages/WithdrawalsPage';
import WhiteLabelsPage from './pages/WhiteLabelsPage';
import AnalyticsPage from './pages/AnalyticsPage';
import PairsPage from './pages/PairsPage';
import AuditLogsPage from './pages/AuditLogsPage';
import ChainsPage from './pages/ChainsPage';
import ReportsPage from './pages/ReportsPage';
import AdminsPage from './pages/AdminsPage';

// Domain admin pages (orphan + new domain pages)
import MarginTradingPage from './pages/MarginTrading';
import CryptoCardsPage from './pages/CryptoCards';
import LiquidityPage from './pages/Liquidity';
import P2PMerchantPage from './pages/P2PMerchant';
import FeaturesPage from './pages/Features';
import FuturesPage from './pages/Futures';
import OptionsPage from './pages/Options';
import CopyTradingPage from './pages/CopyTrading';
import ConvertPage from './pages/Convert';
import OnRampPage from './pages/OnRamp';
import OffRampPage from './pages/OffRamp';
import P2PClientsPage from './pages/P2PClients';
import PartnersPage from './pages/Partners';
import RewardsPage from './pages/Rewards';
import MarketingPage from './pages/Marketing';
import AdminRolesPage from './pages/AdminRoles';
import BotsPage from './pages/Bots';
import BotsClientsPage from './pages/BotsClients';
import ProjectTeamsPage from './pages/ProjectTeams';
import LiquiditySourcesPage from './pages/LiquiditySources';

// Sidebar Component
const Sidebar: React.FC<{ 
  currentPage: string; 
  setCurrentPage: (page: string) => void;
}> = ({ currentPage, setCurrentPage }) => {
  const menuItems = [
    { id: 'dashboard', label: 'Dashboard', icon: '📊' },
    { id: 'analytics', label: 'Analytics', icon: '📈' },
    { id: 'users', label: 'Users', icon: '👥' },
    { id: 'transactions', label: 'Transactions', icon: '📜' },
    { id: 'kyc', label: 'KYC Verification', icon: '✅' },
    { id: 'tokens', label: 'Tokens', icon: '🪙' },
    { id: 'pairs', label: 'Trading Pairs', icon: '🔄' },
    { id: 'chains', label: 'Blockchains', icon: '⛓️' },
    { id: 'withdrawals', label: 'Withdrawals', icon: '💸' },
    { id: 'whitelabels', label: 'White Labels', icon: '🏢' },
    { id: 'fees', label: 'Fees', icon: '💰' },
    { id: 'system', label: 'System', icon: '🖥️' },
    { id: 'auditlogs', label: 'Audit Logs', icon: '📝' },
    { id: 'reports', label: 'Reports', icon: '📋' },
    { id: 'admins', label: 'Admins', icon: '👤' },
    { id: 'admin-roles', label: 'Roles & RBAC', icon: '🔐' },
    { id: 'margin-trading', label: 'Margin Trading', icon: '📈' },
    { id: 'futures', label: 'Futures', icon: '🔥' },
    { id: 'options', label: 'Options', icon: '🎯' },
    { id: 'copy-trading', label: 'Copy Trading', icon: '📋' },
    { id: 'convert', label: 'Convert', icon: '💱' },
    { id: 'liquidity', label: 'Liquidity', icon: '💧' },
    { id: 'onramp', label: 'OnRamp', icon: '⬇️' },
    { id: 'offramp', label: 'OffRamp', icon: '⬆️' },
    { id: 'p2p-merchants', label: 'P2P Merchants', icon: '🤝' },
    { id: 'p2p-clients', label: 'P2P Clients', icon: '👥' },
    { id: 'crypto-cards', label: 'Crypto Cards', icon: '💳' },
    { id: 'partners', label: 'Partners', icon: '🌐' },
    { id: 'rewards', label: 'Rewards', icon: '🎁' },
    { id: 'marketing', label: 'Marketing', icon: '📣' },
    { id: 'features', label: 'Feature Flags', icon: '🚩' },
    { id: 'bots', label: 'Bots', icon: '🤖' },
    { id: 'bots-clients', label: 'Bots Clients', icon: '🔗' },
    { id: 'project-teams', label: 'Project Teams', icon: '👥' },
    { id: 'liquidity-sources', label: 'Liquidity Sources', icon: '🏊' },
    { id: 'settings', label: 'Settings', icon: '⚙️' },
  ];

  return (
    <aside className="w-64 flex flex-col min-h-screen" style={{ 
      backgroundColor: 'var(--sidebar-bg)',
      borderRight: '1px solid var(--border-primary)'
    }}>
      <div className="p-6 border-b" style={{ 
        borderColor: 'var(--border-primary)',
        backgroundColor: 'var(--color-primary)'
      }}>
        <div className="flex items-center space-x-3">
          <span className="text-3xl">🔧</span>
          <div>
            <h1 className="text-xl font-bold" style={{ color: 'var(--text-inverse)' }}>Admin Panel</h1>
            <p className="text-xs" style={{ color: 'rgba(255,255,255,0.7)' }}>Platform Management</p>
          </div>
        </div>
      </div>
      
      <nav className="flex-1 p-4">
        {menuItems.map(item => (
          <button
            key={item.id}
            onClick={() => setCurrentPage(item.id)}
            className="w-full flex items-center space-x-3 px-4 py-3 rounded-lg mb-2 transition-colors"
            style={{
              backgroundColor: currentPage === item.id ? 'var(--sidebar-active)' : 'transparent',
              color: currentPage === item.id ? 'var(--text-inverse)' : 'var(--sidebar-text)',
            }}
          >
            <span>{item.icon}</span>
            <span>{item.label}</span>
          </button>
        ))}
      </nav>

      <div className="p-4 border-t" style={{ borderColor: 'var(--border-primary)' }}>
        <div className="p-3 rounded-lg" style={{ backgroundColor: 'var(--sidebar-hover)' }}>
          <div className="text-xs" style={{ color: 'var(--sidebar-text)' }}>Admin Level</div>
          <div className="font-medium" style={{ color: 'var(--sidebar-text)' }}>Super Admin</div>
        </div>
      </div>
    </aside>
  );
};

// Header Component
const Header: React.FC<{
  toggleTheme: () => void;
  resolvedTheme: 'light' | 'dark';
}> = ({ toggleTheme, resolvedTheme }) => {
  const [searchTerm, setSearchTerm] = useState('');

  return (
    <header className="h-16 flex items-center justify-between px-6" style={{ 
      backgroundColor: 'var(--bg-card)',
      borderBottom: '1px solid var(--border-primary)'
    }}>
      <div className="flex items-center space-x-4">
        <input
          type="text"
          placeholder="Search users, transactions, tokens..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="px-4 py-2 rounded-lg text-sm w-96"
          style={{
            backgroundColor: 'var(--bg-secondary)',
            border: '1px solid var(--border-primary)',
            color: 'var(--text-primary)'
          }}
        />
      </div>
      
      <div className="flex items-center space-x-4">
        <button 
          onClick={toggleTheme}
          className="p-2 rounded-lg"
          style={{ backgroundColor: 'var(--bg-secondary)' }}
          title={resolvedTheme === 'dark' ? 'Switch to Light Mode' : 'Switch to Dark Mode'}
        >
          {resolvedTheme === 'dark' ? '☀️' : '🌙'}
        </button>
        <div className="w-8 h-8 rounded-full flex items-center justify-center" style={{ 
          backgroundColor: 'var(--color-primary)',
          color: 'var(--text-inverse)'
        }}>
          <span className="text-sm font-bold">A</span>
        </div>
      </div>
    </header>
  );
};

// Page Router
const PageRouter: React.FC<{ currentPage: string }> = ({ currentPage }) => {
  switch (currentPage) {
    case 'dashboard':
      return <DashboardPage />;
    case 'analytics':
      return <AnalyticsPage />;
    case 'users':
      return <UsersPage />;
    case 'transactions':
      return <TransactionsPage />;
    case 'kyc':
      return <KycPage />;
    case 'tokens':
      return <TokensPage />;
    case 'pairs':
      return <PairsPage />;
    case 'withdrawals':
      return <WithdrawalsPage />;
    case 'whitelabels':
      return <WhiteLabelsPage />;
    case 'fees':
      return <FeesPage />;
    case 'system':
      return <SystemPage />;
    case 'auditlogs':
      return <AuditLogsPage />;
    case 'chains':
      return <ChainsPage />;
    case 'reports':
      return <ReportsPage />;
    case 'admins':
      return <AdminsPage />;
    case 'admin-roles':
      return <AdminRolesPage />;
    case 'margin-trading':
      return <MarginTradingPage />;
    case 'futures':
      return <FuturesPage />;
    case 'options':
      return <OptionsPage />;
    case 'copy-trading':
      return <CopyTradingPage />;
    case 'convert':
      return <ConvertPage />;
    case 'liquidity':
      return <LiquidityPage />;
    case 'onramp':
      return <OnRampPage />;
    case 'offramp':
      return <OffRampPage />;
    case 'p2p-merchants':
      return <P2PMerchantPage />;
    case 'p2p-clients':
      return <P2PClientsPage />;
    case 'crypto-cards':
      return <CryptoCardsPage />;
    case 'partners':
      return <PartnersPage />;
    case 'rewards':
      return <RewardsPage />;
    case 'marketing':
      return <MarketingPage />;
    case 'features':
      return <FeaturesPage />;
    case 'bots':
      return <BotsPage />;
    case 'bots-clients':
      return <BotsClientsPage />;
    case 'project-teams':
      return <ProjectTeamsPage />;
    case 'liquidity-sources':
      return <LiquiditySourcesPage />;
    case 'settings':
      return <SettingsPage />;
    default:
      return <DashboardPage />;
  }
};

// Main App Component
const AppContent: React.FC = () => {
  const [currentPage, setCurrentPage] = useState('dashboard');
  const { resolvedTheme, toggleTheme } = useTheme();

  // Auth gate: require a stored admin token before exposing the panel. LoginPage
  // persists the token and reloads, so a render-time read is sufficient.
  const authed = typeof window !== 'undefined' && !!localStorage.getItem('admin_token');

  if (!authed) {
    return (
      <LoginPage />
    );
  }

  return (
    <div className="flex min-h-screen" style={{ backgroundColor: 'var(--bg-secondary)' }}>
      <Sidebar currentPage={currentPage} setCurrentPage={setCurrentPage} />
      <div className="flex-1 flex flex-col">
        <Header toggleTheme={toggleTheme} resolvedTheme={resolvedTheme} />
        <main className="flex-1 overflow-auto">
          <PageRouter currentPage={currentPage} />
        </main>
      </div>
    </div>
  );
};

// App with Theme Provider
const App: React.FC = () => {
  return (
    <ThemeProvider>
      <AppContent />
    </ThemeProvider>
  );
};

export default App;
