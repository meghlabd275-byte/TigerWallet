// Send Page - Production Ready
import React, { useState, useEffect, useCallback } from 'react';
import { walletApi, transactionApi, Wallet, TokenBalance } from '../services/api';
import { QRScanner } from '../../../frontend/shared/components/QRScanner';
import './SendPage.css';

const SendPage: React.FC = () => {
  const [recipient, setRecipient] = useState('');
  const [amount, setAmount] = useState('');
  const [selectedToken, setSelectedToken] = useState('ETH');
  const [memo, setMemo] = useState('');
  const [wallets, setWallets] = useState<Wallet[]>([]);
  const [selectedWallet, setSelectedWallet] = useState<string>('');
  const [tokens, setTokens] = useState<{ symbol: string; name: string; icon: string; balance: string }[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [estimatedFee, setEstimatedFee] = useState<string>('');
  const [estimatedFeeUsd, setEstimatedFeeUsd] = useState<string>('');
  const [txHash, setTxHash] = useState<string | null>(null);
  const [showQRScanner, setShowQRScanner] = useState(false);
  const [recentAddresses, setRecentAddresses] = useState<string[]>([]);
  const [tokenPrices, setTokenPrices] = useState<Record<string, number>>({});

  // Load wallets on mount
  useEffect(() => {
    loadWallets();
  }, []);

  const loadWallets = async () => {
    try {
      const walletList = await walletApi.getWallets();
      setWallets(walletList);
      if (walletList.length > 0) {
        setSelectedWallet(walletList[0].id);
      }
    } catch (err) {
      console.error('Failed to load wallets:', err);
      // Use local wallets as fallback
      const storedWallets = localStorage.getItem('wallets');
      if (storedWallets) {
        setWallets(JSON.parse(storedWallets));
      }
    }
  };

  // Load tokens when wallet changes
  useEffect(() => {
    if (selectedWallet) {
      loadTokens();
    }
  }, [selectedWallet]);

  const loadTokens = async () => {
    try {
      const balances = await walletApi.getBalance(selectedWallet);
      const tokenList = balances.tokens.map((tb: TokenBalance) => ({
        symbol: tb.symbol,
        name: tb.name,
        icon: getTokenIcon(tb.symbol),
        balance: tb.balance,
      }));
      
      // Extract prices
      const prices: Record<string, number> = {};
      balances.tokens.forEach((tb: TokenBalance) => {
        if (tb.priceUSD) {
          prices[tb.symbol] = tb.priceUSD;
        }
      });
      setTokenPrices(prices);
      
      // Add native token
      const wallet = wallets.find(w => w.id === selectedWallet);
      if (wallet) {
        tokenList.unshift({
          symbol: wallet.chain.toUpperCase(),
          name: wallet.chain,
          icon: getTokenIcon(wallet.chain.toUpperCase()),
          balance: wallet.balance,
        });
        // Set default token to native
        if (!selectedToken || selectedToken === 'ETH') {
          setSelectedToken(wallet.chain.toUpperCase());
        }
      }
      setTokens(tokenList);
    } catch (err) {
      console.error('Failed to load tokens:', err);
      // Minimal fallback
      setTokens([
        { symbol: 'ETH', name: 'Ethereum', icon: '🔷', balance: '0' },
        { symbol: 'USDT', name: 'Tether USD', icon: '💵', balance: '0' },
        { symbol: 'USDC', name: 'USD Coin', icon: '💲', balance: '0' },
        { symbol: 'BNB', name: 'BNB', icon: '🟡', balance: '0' },
      ]);
    }
  };

  // Estimate fee when recipient or amount changes
  useEffect(() => {
    if (recipient && amount && selectedWallet && parseFloat(amount) > 0) {
      estimateFee();
    }
  }, [recipient, amount, selectedWallet]);

  const estimateFee = async () => {
    try {
      const wallet = wallets.find(w => w.id === selectedWallet);
      if (!wallet) return;
      
      const fee = await transactionApi.estimateGas(
        wallet.address,
        recipient,
        amount,
        selectedToken === wallet.chain.toUpperCase() ? undefined : 
          tokens.find(t => t.symbol === selectedToken)?.address || undefined
      );
      setEstimatedFee(fee.totalFee);
      
      // Calculate USD value
      const price = tokenPrices[selectedToken] || 0;
      const feeNum = parseFloat(fee.totalFee) || 0;
      setEstimatedFeeUsd((feeNum * price).toFixed(2));
    } catch (err) {
      console.error('Failed to estimate fee:', err);
      setEstimatedFee('~0.001');
      setEstimatedFeeUsd('~3.00');
    }
  };

  const getTokenBalance = (symbol: string): string => {
    const token = tokens.find(t => t.symbol === symbol);
    return token?.balance || '0';
  };

  const handleMaxAmount = () => {
    const balance = parseFloat(getTokenBalance(selectedToken));
    const fee = parseFloat(estimatedFee) || 0;
    const maxAmount = Math.max(0, balance - fee).toFixed(6);
    setAmount(maxAmount);
  };

  const getTokenIcon = (symbol: string): string => {
    const icons: Record<string, string> = {
      ETH: '🔷', BNB: '🟡', SOL: '☀️', USDT: '💵', USDC: '💲',
      MATIC: '🟣', WBTC: '₿', LINK: '🔗', DOGE: '🐕', XRP: '💜',
    };
    return icons[symbol.toUpperCase()] || '🪙';
  };

  const validateAddress = (address: string): boolean => {
    // Basic validation - check if address looks valid
    if (!address) return false;
    // EVM address check
    if (address.startsWith('0x') && address.length === 42) return true;
    // Solana address check (base58)
    if (address.length >= 32 && address.length <= 44) return true;
    // TRON address check
    if (address.startsWith('T') && address.length === 34) return true;
    return false;
  };

  const handleSend = async () => {
    if (!recipient || !amount || !selectedWallet) {
      setError('Please fill in all fields');
      return;
    }

    // Validate address
    if (!validateAddress(recipient)) {
      setError('Invalid recipient address');
      return;
    }

    // Validate amount
    const balance = parseFloat(getTokenBalance(selectedToken));
    const sendAmount = parseFloat(amount);
    const fee = parseFloat(estimatedFee) || 0;
    
    if (sendAmount <= 0) {
      setError('Amount must be greater than 0');
      return;
    }

    if (sendAmount + fee > balance) {
      setError('Insufficient balance for transfer + fees');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const result = await transactionApi.send(
        selectedWallet,
        recipient,
        amount,
        selectedToken
      );
      setTxHash(result.hash);
      alert(`Transaction sent! Hash: ${result.hash}`);
      setAmount('');
    } catch (err: any) {
      console.error('Send failed:', err);
      setError(err.message || 'Transaction failed. Please try again.');
    } finally {
      setLoading(false);
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
          <div style={{ display: 'flex', gap: '8px' }}>
            <input
              type="text"
              placeholder="0x... or scan QR code"
              value={recipient}
              onChange={(e) => setRecipient(e.target.value)}
              className="form-input"
              style={{ flex: 1 }}
            />
            <button 
              type="button"
              className="btn btn-secondary"
              onClick={() => setShowQRScanner(true)}
              style={{ padding: '12px 16px', fontSize: '20px' }}
              title="Scan QR Code"
            >
              📷
            </button>
          </div>
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
            <span>Balance: {getTokenBalance(selectedToken)} {selectedToken}</span>
            <button className="max-btn" onClick={handleMaxAmount}>MAX</button>
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
            <span>~{estimatedFee || '0'} {selectedToken} (${estimatedFeeUsd || '0'})</span>
          </div>
          <div className="fee-row">
            <span>Total</span>
            <span>{(parseFloat(amount || '0') + parseFloat(estimatedFee || '0')).toFixed(6)} {selectedToken}</span>
          </div>
        </div>

        {/* Send Button */}
        <button className="btn btn-primary btn-full" onClick={handleSend} disabled={loading}>
          {loading ? 'Sending...' : `Send ${selectedToken}`}
        </button>
      </div>

      {/* QR Scanner Modal */}
      <QRScanner
        isOpen={showQRScanner}
        onClose={() => setShowQRScanner(false)}
        onScan={(address, chain) => {
          setRecipient(address);
          if (chain) {
            console.log('Detected chain:', chain);
          }
        }}
        title="Scan Wallet Address"
        recentAddresses={recentAddresses}
      />
    </div>
  );
};

export default SendPage;
