import React from 'react';

interface AccountPanelProps {
  account: any;
}

export const AccountPanel: React.FC<AccountPanelProps> = ({ account }) => {
  if (!account) {
    return (
      <div className="account-panel">
        <p>Loading account...</p>
      </div>
    );
  }

  const equity = parseFloat(account.totalEquity);
  const available = parseFloat(account.available);
  const usedMargin = parseFloat(account.usedMargin);
  const unrealizedPnl = parseFloat(account.unrealizedPnl);

  return (
    <div className="account-panel">
      <h3>Account</h3>
      <div className="balance-row">
        <span className="label">Total Equity</span>
        <span className="value">${equity.toFixed(2)}</span>
      </div>
      <div className="balance-row">
        <span className="label">Available</span>
        <span className="value">${available.toFixed(2)}</span>
      </div>
      <div className="balance-row">
        <span className="label">Used Margin</span>
        <span className="value">${usedMargin.toFixed(2)}</span>
      </div>
      <div className="balance-row">
        <span className="label">Unrealized P&L</span>
        <span className={`value ${unrealizedPnl >= 0 ? 'profit' : 'loss'}`}>
          ${unrealizedPnl.toFixed(2)}
        </span>
      </div>
    </div>
  );
};