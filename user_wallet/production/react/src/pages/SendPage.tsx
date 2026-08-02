/**
 * Send Page - Send tokens to other addresses
 */

import React, { useState } from 'react';
import { useWallet } from '../contexts/WalletContext';
import { useTheme } from '../contexts/ThemeContext';
import { QRScanner } from '../../../../frontend/shared/components/QRScanner';

function SendPage() {
  const { sendTransaction, getAddress, activeWallet } = useWallet();
  const { theme } = useTheme();
  const [toAddress, setToAddress] = useState('');
  const [amount, setAmount] = useState('');
  const [selectedToken, setSelectedToken] = useState('ETH');
  const [isLoading, setIsLoading] = useState(false);
  const [txHash, setTxHash] = useState('');
  const [error, setError] = useState('');
  const [showQRScanner, setShowQRScanner] = useState(false);
  const [recentAddresses] = useState<string[]>([]);

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setIsLoading(true);

    try {
      const hash = await sendTransaction(toAddress, amount, selectedToken);
      setTxHash(hash);
    } catch (err: any) {
      setError(err.message || 'Transaction failed');
    } finally {
      setIsLoading(false);
    }
  };

  const copyAddress = () => {
    const address = getAddress(activeWallet?.chain as any);
    navigator.clipboard.writeText(address);
  };

  return (
    <div className="p-6 max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Send</h1>

      {/* Success Message */}
      {txHash && (
        <div className={`card mb-6 bg-green-500/20 border-green-500 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
          <h3 className="font-semibold text-green-500 mb-2">Transaction Submitted!</h3>
          <p className="text-sm opacity-70">Tx Hash:</p>
          <p className="font-mono text-xs break-all">{txHash}</p>
        </div>
      )}

      {/* Error Message */}
      {error && (
        <div className={`card mb-6 bg-red-500/20 border-red-500 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
          <p className="text-red-500">{error}</p>
        </div>
      )}

      {/* Send Form */}
      <form onSubmit={handleSend} className={`card ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
        {/* Token Selector */}
        <div className="mb-4">
          <label className="label">Token</label>
          <select
            value={selectedToken}
            onChange={(e) => setSelectedToken(e.target.value)}
            className="input"
          >
            <option value="ETH">Ethereum (ETH)</option>
            <option value="MATIC">Polygon (MATIC)</option>
            <option value="BNB">BNB Chain (BNB)</option>
            <option value="AVAX">Avalanche (AVAX)</option>
            <option value="SOL">Solana (SOL)</option>
          </select>
        </div>

        {/* To Address */}
        <div className="mb-4">
          <label className="label">Recipient Address</label>
          <div className="flex gap-2">
            <input
              type="text"
              value={toAddress}
              onChange={(e) => setToAddress(e.target.value)}
              placeholder="0x... or scan QR code"
              className="input font-mono flex-1"
              required
            />
            <button
              type="button"
              onClick={() => setShowQRScanner(true)}
              className="btn btn-secondary px-4"
              title="Scan QR Code"
            >
              📷
            </button>
          </div>
        </div>

        {/* Amount */}
        <div className="mb-6">
          <label className="label">Amount</label>
          <input
            type="number"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            placeholder="0.00"
            step="0.000001"
            className="input"
            required
          />
        </div>

        <button
          type="submit"
          disabled={isLoading || !toAddress || !amount}
          className="btn btn-primary w-full"
        >
          {isLoading ? 'Sending...' : 'Send'}
        </button>
      </form>

      {/* Your Address */}
      <div className={`card mt-6 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
        <h3 className="font-semibold mb-2">Your Address</h3>
        <div className="flex items-center gap-2">
          <input
            type="text"
            value={getAddress(activeWallet?.chain as any) || ''}
            readOnly
            className="input flex-1 font-mono text-sm"
          />
          <button onClick={copyAddress} className="btn btn-secondary">
            Copy
          </button>
        </div>
      </div>

      {/* QR Scanner Modal */}
      <QRScanner
        isOpen={showQRScanner}
        onClose={() => setShowQRScanner(false)}
        onScan={(address, chain) => {
          setToAddress(address);
          if (chain) {
            console.log('Detected chain:', chain);
          }
        }}
        title="Scan Wallet Address"
        recentAddresses={recentAddresses}
      />
    </div>
  );
}

export default SendPage;
