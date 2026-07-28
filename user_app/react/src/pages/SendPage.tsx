// Send Page
import React, { useState } from 'react';
import './SendPage.css';

const SendPage: React.FC = () => {
  const [recipient, setRecipient] = useState('');
  const [amount, setAmount] = useState('');
  const [selectedToken, setSelectedToken] = useState('ETH');
  const [memo, setMemo] = useState('');

  const tokens = [
    { symbol: 'ETH', name: 'Ethereum', icon: '🔷' },
    { symbol: 'USDT', name: 'Tether USD', icon: '💵' },
    { symbol: 'USDC', name: 'USD Coin', icon: '💲' },
    { symbol: 'BNB', name: 'BNB', icon: '🟡' },
  ];

  const handleSend = () => {
    if (recipient && amount) {
      alert(`Sending ${amount} ${selectedToken} to ${recipient}`);
    }
  };

  return (
    <div className="send-page">
      <div className="page-header">
        <h1>Send</h1>
      </div>

      <div className="send-form">
        {/* Token Selector */}
        <div className="form-group">
          <label>Token</label>
          <div className="token-selector">
            {tokens.map(token => (
              <button
                key={token.symbol}
                className={`token-option ${selectedToken === token.symbol ? 'selected' : ''}`}
                onClick={() => setSelectedToken(token.symbol)}
              >
                <span className="token-icon">{token.icon}</span>
                <span className="token-symbol">{token.symbol}</span>
              </button>
            ))}
          </div>
        </div>

        {/* Recipient */}
        <div className="form-group">
          <label>Recipient Address</label>
          <input
            type="text"
            placeholder="0x..."
            value={recipient}
            onChange={(e) => setRecipient(e.target.value)}
            className="form-input"
          />
        </div>

        {/* Amount */}
        <div className="form-group">
          <label>Amount</label>
          <div className="amount-input">
            <input
              type="number"
              placeholder="0.0"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              className="form-input"
            />
            <span className="token-symbol">{selectedToken}</span>
          </div>
          <div className="balance-info">
            <span>Balance: 5.5 {selectedToken}</span>
            <button className="max-btn" onClick={() => setAmount('5.5')}>MAX</button>
          </div>
        </div>

        {/* Memo (optional) */}
        <div className="form-group">
          <label>Memo (Optional)</label>
          <input
            type="text"
            placeholder="Add a note..."
            value={memo}
            onChange={(e) => setMemo(e.target.value)}
            className="form-input"
          />
        </div>

        {/* Fee Info */}
        <div className="fee-info">
          <div className="fee-row">
            <span>Network Fee</span>
            <span>~0.005 {selectedToken}</span>
          </div>
          <div className="fee-row">
            <span>Total</span>
            <span>{parseFloat(amount || '0') + 0.005} {selectedToken}</span>
          </div>
        </div>

        {/* Send Button */}
        <button className="btn btn-primary btn-full" onClick={handleSend}>
          Send {selectedToken}
        </button>
      </div>
    </div>
  );
};

export default SendPage;
