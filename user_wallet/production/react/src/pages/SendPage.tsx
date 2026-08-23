import React, { useState } from 'react';
import { useWallet } from '../contexts/WalletContext';
import { useTheme } from '../contexts/ThemeContext';
import { WalletService, chainIdFor, SimulateResult } from '../services/WalletService';
import TxSubmittedBanner from '../components/TxSubmittedBanner';
import QRScanner from '../components/QRScanner';

const SendPage: React.FC = () => {
  const { activeWallet, sendTransaction } = useWallet();
  const { theme } = useTheme();
  const [walletService] = useState(() => new WalletService());
  const [toAddress, setToAddress] = useState('');
  const [amount, setAmount] = useState('');
  const [selectedToken, setSelectedToken] = useState('ETH');
  const [txHash, setTxHash] = useState('');
  const [txChainId, setTxChainId] = useState(1);
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [showQRScanner, setShowQRScanner] = useState(false);

  // ENS state: recipient input may be an ENS name (alice.eth); resolve to a
  // real 0x address, show it to the user, then send to the resolved address.
  const [ensName, setEnsName] = useState<string | null>(null);
  const [resolvedTo, setResolvedTo] = useState<string | null>(null);
  const [resolvingEns, setResolvingEns] = useState(false);
  const [ensError, setEnsError] = useState('');

  // Optional EIP-1559 gas overrides (gwei strings, forwarded to /send).
  const [maxFeeGwei, setMaxFeeGwei] = useState('');
  const [maxPriorityGwei, setMaxPriorityGwei] = useState('');

  // Simulation (pre-sign dry-run) state.
  const [sim, setSim] = useState<SimulateResult | null>(null);
  const [simulating, setSimulating] = useState(false);

  const activeChainId = chainIdFor(String(activeWallet?.chain?.id ?? '1'));

  const handleRecipientChange = async (raw: string) => {
    setToAddress(raw);
    setSim(null);
    const trimmed = raw.trim();
    if (trimmed.toLowerCase().endsWith('.eth')) {
      setResolvingEns(true);
      setEnsError('');
      try {
        const r = await walletService.resolveENS(trimmed);
        setEnsName(r.name);
        setResolvedTo(r.address);
      } catch (e: any) {
        setEnsName(null);
        setResolvedTo(null);
        setEnsError(e?.response?.data?.error || e.message || 'ENS resolution failed');
      } finally {
        setResolvingEns(false);
      }
    } else {
      setEnsName(null);
      setResolvedTo(null);
      setEnsError('');
    }
  };

  const handleSimulate = async () => {
    setError('');
    setSim(null);
    const to = (resolvedTo || toAddress.trim());
    if (!/^0x[a-fA-F0-9]{40}$/.test(to)) { setError('Enter a valid recipient address (or resolvable ENS name)'); return; }
    if (!activeWallet) { setError('No active wallet'); return; }
    setSimulating(true);
    try {
      const result = await walletService.simulateTransaction({
        chainId: activeChainId,
        from: activeWallet.address,
        to,
        value: amount || undefined,
      });
      setSim(result);
    } catch (err: any) {
      setError(err?.response?.data?.error || err.message || 'Simulation failed');
    } finally {
      setSimulating(false);
    }
  };

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setTxHash('');

    if (!activeWallet) {
      setError('No active wallet');
      return;
    }

    const to = (resolvedTo || toAddress.trim());
    if (!/^0x[a-fA-F0-9]{40}$/.test(to)) {
      setError('Enter a valid recipient address (or resolvable ENS name)');
      return;
    }

    setIsLoading(true);
    try {
      const hash = await sendTransaction(to, amount, selectedToken, {
        maxFeeGwei: maxFeeGwei.trim() || undefined,
        maxPriorityGwei: maxPriorityGwei.trim() || undefined,
      });
      setTxHash(hash);
      setTxChainId(activeChainId);
      setToAddress('');
      setAmount('');
      setEnsName(null);
      setResolvedTo(null);
      setSim(null);
    } catch (err: any) {
      setError(err.message || 'Transaction failed');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center space-x-3">
        <h1 className="text-3xl font-bold">Send Tokens</h1>
        <div className="p-2 rounded-lg bg-gradient-to-r from-primary-500 to-primary-600">
          <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
          </svg>
        </div>
      </div>

      {txHash && (
        <TxSubmittedBanner
          txHash={txHash}
          chainId={txChainId}
          onDismiss={() => setTxHash('')}
        />
      )}

      <div className={`${theme === 'dark' ? 'bg-slate-800' : 'bg-white'} rounded-xl p-6 card-shadow`}>
        <form onSubmit={handleSend} className="space-y-6">
          <div>
            <label className="block text-sm font-medium mb-2">Token</label>
            <select
              value={selectedToken}
              onChange={(e) => setSelectedToken(e.target.value)}
              className={`w-full px-4 py-3 rounded-lg border ${
                theme === 'dark'
                  ? 'bg-slate-700 border-slate-600'
                  : 'bg-gray-50 border-gray-300'
              } focus:outline-none focus:ring-2 focus:ring-primary-500`}
            >
              <option value="ETH">ETH</option>
              <option value="USDC">USDC</option>
              <option value="USDT">USDT</option>
              <option value="DAI">DAI</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">To Address or ENS name</label>
            <div className="relative">
              <input
                type="text"
                value={toAddress}
                onChange={(e) => handleRecipientChange(e.target.value)}
                placeholder="0x... or alice.eth"
                className={`w-full px-4 py-3 pr-12 rounded-lg border ${
                  theme === 'dark'
                    ? 'bg-slate-700 border-slate-600'
                    : 'bg-gray-50 border-gray-300'
                } focus:outline-none focus:ring-2 focus:ring-primary-500`}
                required
              />
              <button
                type="button"
                onClick={() => setShowQRScanner(true)}
                title="Scan QR code"
                className={`absolute right-2 top-1/2 -translate-y-1/2 p-2 rounded-lg transition-colors ${
                  theme === 'dark'
                    ? 'text-gray-400 hover:text-gray-200 hover:bg-slate-600'
                    : 'text-gray-500 hover:text-gray-700 hover:bg-gray-200'
                }`}
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" />
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
              </button>
            </div>
            {resolvingEns && (
              <p className="mt-1 text-xs text-gray-500">Resolving ENS…</p>
            )}
            {resolvedTo && ensName && (
              <p className="mt-1 text-xs text-green-500">
                ✓ {ensName} → <span className="font-mono">{resolvedTo}</span>
              </p>
            )}
            {ensError && (
              <p className="mt-1 text-xs text-red-500">⚠ {ensError}</p>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">Amount</label>
            <input
              type="text"
              value={amount}
              onChange={(e) => { setAmount(e.target.value); setSim(null); }}
              placeholder="0.0"
              className={`w-full px-4 py-3 rounded-lg border ${
                theme === 'dark'
                  ? 'bg-slate-700 border-slate-600'
                  : 'bg-gray-50 border-gray-300'
              } focus:outline-none focus:ring-2 focus:ring-primary-500`}
              required
            />
            {activeWallet && (
              <p className="mt-1 text-sm text-gray-500">
                Balance: {activeWallet.balance} {activeWallet.chain?.symbol || ''}
              </p>
            )}
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-2">Max fee (gwei, optional)</label>
              <input
                type="text"
                value={maxFeeGwei}
                onChange={(e) => setMaxFeeGwei(e.target.value)}
                placeholder="auto"
                className={`w-full px-4 py-3 rounded-lg border ${
                  theme === 'dark'
                    ? 'bg-slate-700 border-slate-600'
                    : 'bg-gray-50 border-gray-300'
                } focus:outline-none focus:ring-2 focus:ring-primary-500`}
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-2">Priority fee (gwei, optional)</label>
              <input
                type="text"
                value={maxPriorityGwei}
                onChange={(e) => setMaxPriorityGwei(e.target.value)}
                placeholder="auto"
                className={`w-full px-4 py-3 rounded-lg border ${
                  theme === 'dark'
                    ? 'bg-slate-700 border-slate-600'
                    : 'bg-gray-50 border-gray-300'
                } focus:outline-none focus:ring-2 focus:ring-primary-500`}
              />
            </div>
          </div>

          {sim && (
            <div className={`p-4 rounded-lg border ${
              sim.success && !sim.will_revert
                ? theme === 'dark' ? 'bg-green-900/20 border-green-700' : 'bg-green-50 border-green-200'
                : theme === 'dark' ? 'bg-red-900/20 border-red-700' : 'bg-red-50 border-red-200'
            }`}>
              {sim.success && !sim.will_revert ? (
                <p className="text-sm text-green-500">
                  ✓ Simulation succeeded — estimated gas: <span className="font-mono">{sim.gas_estimate}</span>
                  {sim.estimated_cost_wei && (
                    <> · est. cost <span className="font-mono">{(Number(sim.estimated_cost_wei) / 1e18).toFixed(6)}</span> native</>
                  )}
                </p>
              ) : (
                <div className="text-sm text-red-500">
                  <p className="font-medium">⚠ Transaction will revert</p>
                  <p className="font-mono break-all">{sim.revert_reason || sim.estimate_error || 'unknown reason'}</p>
                  {sim.gas_estimate > 0 && <p>Estimated gas: {sim.gas_estimate}</p>}
                </div>
              )}
            </div>
          )}

          {error && (
            <div className={`p-4 rounded-lg ${
              theme === 'dark' ? 'bg-red-900/20 border border-red-800' : 'bg-red-50 border border-red-200'
            }`}>
              <p className="text-sm text-red-600">{error}</p>
            </div>
          )}

          <div className="flex space-x-3">
            <button
              type="button"
              onClick={handleSimulate}
              disabled={simulating || isLoading}
              className={`flex-1 py-3 px-4 rounded-lg font-semibold border transition-colors disabled:opacity-50 disabled:cursor-not-allowed ${
                theme === 'dark'
                  ? 'border-slate-600 hover:bg-slate-700'
                  : 'border-gray-300 hover:bg-gray-100'
              }`}
            >
              {simulating ? 'Simulating...' : 'Simulate'}
            </button>
            <button
              type="submit"
              disabled={isLoading || simulating}
              className="flex-1 py-3 px-4 rounded-lg font-semibold text-white bg-gradient-to-r from-primary-500 to-primary-600 hover:from-primary-600 hover:to-primary-700 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isLoading ? 'Sending...' : 'Send Transaction'}
            </button>
          </div>
        </form>
      </div>

      {showQRScanner && (
        <QRScanner
          isOpen={showQRScanner}
          onScan={(address) => {
            handleRecipientChange(address);
            setShowQRScanner(false);
          }}
          onClose={() => setShowQRScanner(false)}
        />
      )}
    </div>
  );
};

export default SendPage;
