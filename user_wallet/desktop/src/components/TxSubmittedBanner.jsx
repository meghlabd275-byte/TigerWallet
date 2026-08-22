// TxSubmittedBanner — the post-send confirmation shown on every outgoing
// transaction. Displays "Transaction submitted to the blockchain network"
// alongside the real tx hash + a link to the block explorer (per chain).
// Auto-dismisses after 30s; the user can also dismiss manually.
import React, { useEffect, useState } from 'react';
import { useTheme } from '../contexts/ThemeContext';

const EXPLORERS = {
  1: 'https://etherscan.io/tx/',
  56: 'https://bscscan.com/tx/',
  137: 'https://polygonscan.com/tx/',
  42161: 'https://arbiscan.io/tx/',
  10: 'https://optimistic.etherscan.io/tx/',
  8453: 'https://basescan.org/tx/',
  43114: 'https://snowtrace.io/tx/',
};

export default function TxSubmittedBanner({ txHash, chainId, onDismiss }) {
  const { isDark } = useTheme();
  const [visible, setVisible] = useState(true);

  useEffect(() => {
    const t = setTimeout(() => setVisible(false), 30000);
    return () => clearTimeout(t);
  }, []);

  if (!visible) return null;

  const explorer = EXPLORERS[chainId] || '';

  return (
    <div className={`tx-submitted-banner ${isDark ? 'dark' : 'light'}`}>
      <div className="banner-icon">⛓️</div>
      <div className="banner-content">
        <strong>Transaction submitted to the blockchain network</strong>
        <div className="banner-hash">
          {explorer ? (
            <a href={`${explorer}${txHash}`} target="_blank" rel="noopener noreferrer">
              {txHash.slice(0, 10)}…{txHash.slice(-8)} ↗
            </a>
          ) : (
            <code>{txHash.slice(0, 16)}…</code>
          )}
        </div>
        <small>Awaiting on-chain confirmation. This may take a few moments depending on network congestion.</small>
      </div>
      <button
        className="banner-close"
        onClick={() => { setVisible(false); if (onDismiss) onDismiss(); }}
        aria-label="Dismiss"
      >
        ×
      </button>
    </div>
  );
}
