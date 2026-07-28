import React from 'react';

export const WalletsPage: React.FC = () => (
  <div className="page">
    <h1>Wallets Management</h1>
    <p>Manage master wallet and user wallets</p>
  </div>
);

export const BlockchainPage: React.FC = () => (
  <div className="page">
    <h1>Blockchain Management</h1>
    <p>Configure blockchain networks and RPC endpoints</p>
  </div>
);

export const PairsPage: React.FC = () => (
  <div className="page">
    <h1>Trading Pairs</h1>
    <p>Manage trading pairs and liquidity</p>
  </div>
);

export const LiquidityPage: React.FC = () => (
  <div className="page">
    <h1>Liquidity Management</h1>
    <p>Manage liquidity pools and providers</p>
  </div>
);

export const FeesPage: React.FC = () => (
  <div className="page">
    <h1>Fee Management</h1>
    <p>Configure withdrawal, swap, and transaction fees</p>
  </div>
);

export const KYCPage: React.FC = () => (
  <div className="page">
    <h1>KYC Management</h1>
    <p>Review and manage user KYC submissions</p>
  </div>
);

export const TransactionsPage: React.FC = () => (
  <div className="page">
    <h1>Transactions</h1>
    <p>View all platform transactions</p>
  </div>
);

export const AnalyticsPage: React.FC = () => (
  <div className="page">
    <h1>Analytics</h1>
    <p>Platform analytics and reporting</p>
  </div>
);

export const SettingsPage: React.FC = () => (
  <div className="page">
    <h1>Settings</h1>
    <p>Admin panel settings</p>
  </div>
);

export const LoginPage: React.FC = ({ onLogin }: { onLogin: (token: string) => void }) => (
  <div className="login-page">
    <div className="login-card">
      <h1>🐯 TigerWallet Admin</h1>
      <p>Sign in to continue</p>
      <form onSubmit={(e) => { e.preventDefault(); onLogin('token'); }}>
        <input type="text" placeholder="Username" className="form-input" />
        <input type="password" placeholder="Password" className="form-input" />
        <button type="submit" className="btn btn-primary">Sign In</button>
      </form>
    </div>
  </div>
);

export { WalletsPage, BlockchainPage, PairsPage, LiquidityPage, FeesPage, KYCPage, TransactionsPage, AnalyticsPage, SettingsPage, LoginPage };
