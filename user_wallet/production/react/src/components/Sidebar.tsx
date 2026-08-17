/**
 * Sidebar - left navigation rail with the full UserWallet route set.
 * Highlights the active route and is themed via CSS variables (light/dark).
 */

import React from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { useWallet } from '../contexts/WalletContext';

interface NavItem {
  label: string;
  path: string;
  icon: React.ReactNode;
}

const icon = (path: string) => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    {path}
  </svg>
);

const NAV_ITEMS: NavItem[] = [
  { label: 'Home', path: '/', icon: <>{icon('M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 0 0 1 1h3m10-11l2 2m-2-2v10a1 1 0 0 1-1 1h-3m-6 0h6')}</> },
  { label: 'Wallet', path: '/wallet', icon: <>{icon('M21 12V7H5a2 2 0 0 1 0-4h14v4M3 5v14a2 2 0 0 0 2 2h16v-5M18 12a2 2 0 0 0 0 4h4v-4z')}</> },
  { label: 'Send', path: '/send', icon: <>{icon('M12 19l9 2-9-18-9 18 9-2zm0 0v-8')}</> },
  { label: 'Receive', path: '/receive', icon: <>{icon('M12 5v14m-7-7h14')}</> },
  { label: 'Swap', path: '/swap', icon: <>{icon('M7 16V4m0 0L3 8m4-4l4 4m6 4v12m0 0l4-4m-4 4l-4-4')}</> },
  { label: 'Bridge', path: '/bridge', icon: <>{icon('M7 7h10M7 7l3-3M7 7l3 3M17 17H7m10 0l-3-3m3 3l-3 3')}</> },
  { label: 'Staking', path: '/staking', icon: <>{icon('M12 2v20M2 7l10 5 10-5')}</> },
  { label: 'NFTs', path: '/nfts', icon: <>{icon('M4 4h16v16H4zM4 12h16M12 4v16')}</> },
  { label: 'History', path: '/history', icon: <>{icon('M12 8v4l3 3m6-3a9 9 0 1 1-18 0 9 9 0 0 1 18 0z')}</> },
  { label: 'KYC', path: '/kyc', icon: <>{icon('M9 12l2 2 4-4m6 2a9 9 0 1 1-18 0 9 9 0 0 1 18 0z')}</> },
  { label: 'Address Book', path: '/address-book', icon: <>{icon('M17 20h5v-2a4 4 0 0 0-3-3.87M9 20H4v-2a4 4 0 0 1 3-3.87m6-2.13a4 4 0 1 0-4-4 4 4 0 0 0 4 4zm6 0a4 4 0 1 0-3-3.87')}</> },
  { label: 'Approvals', path: '/approvals', icon: <>{icon('M9 12l2 2 4-4m6 2a9 9 0 1 1-18 0 9 9 0 0 1 18 0z')}</> },
  { label: 'Devices', path: '/devices', icon: <>{icon('M9 17v-2a4 4 0 0 1 4-4h2M7 4h10a1 1 0 0 1 1 1v7a1 1 0 0 1-1 1H7a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1zM9 21h6')}</> },
  { label: 'Keystore', path: '/keystore', icon: <>{icon('M21 2l-2 2m-7.5 7.5l-2 2M7 17l-3 3M16 6l2 2M12 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8m-4-8h4v4')}</> },
  { label: 'DeFi', path: '/defi', icon: <>{icon('M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 0 0 1 1h3m10-11l2 2m-2-2v10a1 1 0 0 1-1 1h-3m-6 0h6m-6 0v-5a2 2 0 0 1 2-2h2a2 2 0 0 1 2 2v5')}</> },
  { label: 'DApps', path: '/dapps', icon: <>{icon('M4 6a2 2 0 0 1 2-2h2M4 6v2a2 2 0 0 0 2 2M4 6h2m12 0a2 2 0 0 0 2-2V2M20 6v2a2 2 0 0 1-2 2M20 6h-2M4 18a2 2 0 0 0 2 2h2M4 18v-2a2 2 0 0 1 2-2M4 18h2m12 0a2 2 0 0 1-2 2h-2M20 18v-2a2 2 0 0 0-2-2M20 18h-2')}</> },
  { label: 'Settings', path: '/settings', icon: <>{icon('M10.3 3.6a1.5 1.5 0 0 1 3.4 0 1.5 1.5 0 0 0 2.1 1.7 1.5 1.5 0 0 1 2.4 1.3 1.5 1.5 0 0 0 1 2.5 1.5 1.5 0 0 1-1 2.5 1.5 1.5 0 0 0-2.4 1.3 1.5 1.5 0 0 1-2.1 1.7 1.5 1.5 0 0 1-3.4 0 1.5 1.5 0 0 0-2.1-1.7 1.5 1.5 0 0 1-2.4-1.3 1.5 1.5 0 0 0-1-2.5 1.5 1.5 0 0 1 1-2.5 1.5 1.5 0 0 0 2.4-1.3zM12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z')}</> },
];

const Sidebar: React.FC = () => {
  const location = useLocation();
  const navigate = useNavigate();
  const { activeWallet } = useWallet();

  const isActive = (path: string) =>
    path === '/' ? location.pathname === '/' : location.pathname.startsWith(path);

  return (
    <aside
      className="fixed top-0 bottom-0 left-0 z-40 flex w-64 flex-col"
      style={{
        background: 'var(--color-bg-secondary)',
        borderRight: '1px solid var(--color-border)',
      }}
    >
      {/* Brand */}
      <div className="flex items-center gap-2 px-6 h-16" style={{ borderBottom: '1px solid var(--color-border)' }}>
        <div
          className="flex items-center justify-center w-8 h-8 rounded-lg font-bold text-white"
          style={{ background: 'var(--color-primary)' }}
        >
          T
        </div>
        <span className="text-lg font-bold" style={{ color: 'var(--color-text-primary)' }}>
          TigerWallet
        </span>
      </div>

      {/* Active wallet indicator */}
      {activeWallet && (
        <div className="px-4 py-3" style={{ borderBottom: '1px solid var(--color-border)' }}>
          <p className="text-xs uppercase tracking-wide" style={{ color: 'var(--color-text-tertiary)' }}>
            Active wallet
          </p>
          <p className="mt-1 text-sm font-mono truncate" style={{ color: 'var(--color-text-secondary)' }}>
            {activeWallet.address}
          </p>
          <p className="text-xs" style={{ color: 'var(--color-text-tertiary)' }}>
            {activeWallet.chain?.name || '—'}
          </p>
        </div>
      )}

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto py-4">
        {NAV_ITEMS.map((item) => {
          const active = isActive(item.path);
          return (
            <button
              key={item.path}
              onClick={() => navigate(item.path)}
              className="flex items-center w-full gap-3 px-6 py-3 text-sm transition-colors"
              style={{
                color: active ? 'var(--color-primary)' : 'var(--color-text-secondary)',
                background: active ? 'var(--color-bg-tertiary)' : 'transparent',
                borderLeft: active ? '3px solid var(--color-primary)' : '3px solid transparent',
                fontWeight: active ? 600 : 400,
              }}
            >
              {item.icon}
              {item.label}
            </button>
          );
        })}
      </nav>
    </aside>
  );
};

export default Sidebar;
