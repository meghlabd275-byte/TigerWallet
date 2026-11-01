'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../components/ThemeProvider';

// A fee level derived by the backend from eth_feeHistory percentiles: what
// transactions in the last 20 blocks actually paid to be included. There is no
// client-side multiplier here, because inventing one would mean quoting the
// user a price the chain never offered.
interface GasTier {
  priority_fee_wei?: string;
  max_fee_wei?: string;
  base_fee_wei?: string;
}

interface ChainGas {
  chainId: number;
  chainName: string;
  symbol: string;
  gasPriceWei: string;
  tiers: {
    slow: GasTier;
    standard: GasTier;
    fast: GasTier;
  } | null;
}

const API_BASE_URL = typeof window !== 'undefined' ? '' : (process.env.BACKEND_URL || 'http://localhost:8443');

const GAS_CHAINS: { id: number; name: string; symbol: string }[] = [
  { id: 1, name: 'Ethereum', symbol: 'ETH' },
  { id: 56, name: 'BNB Smart Chain', symbol: 'BNB' },
  { id: 137, name: 'Polygon', symbol: 'MATIC' },
  { id: 42161, name: 'Arbitrum One', symbol: 'ARB' },
  { id: 10, name: 'Optimism', symbol: 'OP' },
  { id: 8453, name: 'Base', symbol: 'ETH' },
  { id: 43114, name: 'Avalanche', symbol: 'AVAX' },
  { id: 250, name: 'Fantom', symbol: 'FTM' },
];

// Renders one fee level. The wait estimate is derived from the percentile the
// tip came from: a 90th-percentile tip lands in the next block, a 10th-percentile
// tip may wait several.
function FeeRow({
  label,
  tier,
  isDark,
  highlight = false,
}: {
  label: string;
  tier: GasTier;
  isDark: boolean;
  highlight?: boolean;
}) {
  const gwei = (wei?: string) => {
    if (!wei) return '-';
    try {
      const value = BigInt(wei);
      const whole = value / 1000000000n;
      const frac = (value % 1000000000n).toString().padStart(9, '0').replace(/0+$/, '');
      return frac ? `${whole}.${frac}` : whole.toString();
    } catch {
      return '-';
    }
  };
  return (
    <div
      className={`p-4 rounded-lg flex justify-between items-center ${highlight ? (isDark ? 'bg-orange-900/30 border-2 border-orange-500' : 'bg-orange-100 border-2 border-orange-500') : isDark ? 'bg-gray-700' : 'bg-gray-100'}`}
    >
      <div>
        <div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{label}</div>
        <div className={`text-xl font-bold ${highlight ? 'text-orange-600' : ''}`}>{gwei(tier.max_fee_wei)} Gwei max fee</div>
      </div>
      <div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
        tip {gwei(tier.priority_fee_wei)} Gwei
      </div>
    </div>
  );
}

export default function GasEstimation() {
  const [chainsGas, setChainsGas] = useState<ChainGas[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedChain, setSelectedChain] = useState<number>(1);
  const [txType, setTxType] = useState<string>('transfer');
  const [gasLimit, setGasLimit] = useState<string>('21000');
  const { isDark } = useTheme();

  const loadGas = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const results = await Promise.allSettled(
        GAS_CHAINS.map(async (ch) => {
          const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
          const res = await fetch(`${API_BASE_URL}/api/v1/gas?chain_id=${ch.id}`, {
            headers: token ? { Authorization: `Bearer ${token}` } : {},
          });
          if (!res.ok) throw new Error(`gas fetch failed for ${ch.name}`);
          const data = await res.json();
          // gas_price is the raw wei value from eth_gasPrice; tiers come from
          // eth_feeHistory and are absent on chains that do not support it.
          if (!data.gas_price) throw new Error(`no gas price returned for ${ch.name}`);
          return {
            chainId: ch.id,
            chainName: ch.name,
            symbol: ch.symbol,
            gasPriceWei: String(data.gas_price),
            tiers: data.tiers
              ? {
                  slow: data.tiers.slow || {},
                  standard: data.tiers.standard || {},
                  fast: data.tiers.fast || {},
                }
              : null,
          } as ChainGas;
        })
      );
      const ok: ChainGas[] = [];
      let lastErr: string | null = null;
      for (const r of results) {
        if (r.status === 'fulfilled') ok.push(r.value);
        else lastErr = r.reason instanceof Error ? r.reason.message : 'gas fetch failed';
      }
      setChainsGas(ok);
      if (ok.length === 0 && lastErr) setError(lastErr);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load gas prices');
      setChainsGas([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadGas();
  }, [loadGas]);

  // Fee in wei per gas -> native cost. BigInt keeps wei exact; floating point
  // loses precision well before 18 decimals.
  const costFromWei = (feeWei: string | undefined, limit: string): string | null => {
    if (!feeWei) return null;
    let limitBig: bigint;
    try {
      limitBig = BigInt(limit);
    } catch {
      return null;
    }
    let feeBig: bigint;
    try {
      feeBig = BigInt(feeWei);
    } catch {
      return null;
    }
    const total = feeBig * limitBig;
    const whole = total / 1000000000000000000n;
    const frac = (total % 1000000000000000000n).toString().padStart(18, '0').replace(/0+$/, '');
    return frac ? `${whole}.${frac}` : whole.toString();
  };

  const weiToGwei = (wei: string | undefined): string | null => {
    if (!wei) return null;
    try {
      const value = BigInt(wei);
      const whole = value / 1000000000n;
      const frac = (value % 1000000000n).toString().padStart(9, '0').replace(/0+$/, '');
      return frac ? `${whole}.${frac}` : whole.toString();
    } catch {
      return null;
    }
  };

  const getGasLimit = (type: string): string => {
    const limits: Record<string, string> = { transfer: '21000', swap: '150000', nftTransfer: '85000', contractDeploy: '3000000', stake: '100000', approve: '50000' };
    return limits[type] || '21000';
  };

  const selectedChainGas = chainsGas.find(c => c.chainId === selectedChain);
  const currentGasLimit = getGasLimit(txType);

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900 text-white' : 'bg-gray-50 text-gray-900'}`}>
      <header className={`${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'} border-b ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
        <div className="max-w-7xl mx-auto px-4"><div className="flex items-center justify-between h-16"><div className="flex items-center gap-4"><a href="/" className="text-2xl">🐯</a><h1 className="text-xl font-bold">Gas Estimator</h1></div></div></div>
      </header>
      <div className="max-w-7xl mx-auto px-4 py-8">
        {error && <div className={`mb-6 rounded-lg border px-4 py-3 ${isDark ? 'border-amber-700 bg-amber-900/30 text-amber-200' : 'border-amber-400 bg-amber-50 text-amber-900'}`}>{error}</div>}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'} rounded-lg p-6`}>
            <h2 className="text-xl font-semibold mb-4">Select Network & Transaction</h2>
            <div className="mb-4"><label className={`block text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'} mb-2`}>Blockchain</label><select value={selectedChain} onChange={(e) => setSelectedChain(Number(e.target.value))} className={`w-full ${isDark ? 'bg-gray-700' : 'bg-gray-100'} rounded-lg px-4 py-3`}>{chainsGas.map(c => <option key={c.chainId} value={c.chainId}>{c.chainName} ({c.symbol})</option>)}</select></div>
            <div className="mb-4"><label className={`block text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'} mb-2`}>Transaction Type</label><select value={txType} onChange={(e) => { setTxType(e.target.value); setGasLimit(getGasLimit(e.target.value)); }} className={`w-full ${isDark ? 'bg-gray-700' : 'bg-gray-100'} rounded-lg px-4 py-3`}><option value="transfer">Token Transfer</option><option value="swap">Token Swap</option><option value="nftTransfer">NFT Transfer</option><option value="approve">Token Approval</option><option value="stake">Staking</option><option value="contractDeploy">Contract Deploy</option></select></div>
            <div className="mb-4"><label className={`block text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'} mb-2`}>Gas Limit</label><input type="text" value={gasLimit} onChange={(e) => setGasLimit(e.target.value)} className={`w-full ${isDark ? 'bg-gray-700' : 'bg-gray-100'} rounded-lg px-4 py-3`} /></div>
            {selectedChainGas && (
              <div className="space-y-3">
                {selectedChainGas.tiers ? (
                  <>
                    <FeeRow label="Slow (10th percentile tip)" tier={selectedChainGas.tiers.slow} isDark={isDark} />
                    <FeeRow label="Standard (median tip)" tier={selectedChainGas.tiers.standard} isDark={isDark} highlight />
                    <FeeRow label="Fast (90th percentile tip)" tier={selectedChainGas.tiers.fast} isDark={isDark} />
                  </>
                ) : (
                  <div className={`p-4 rounded-lg ${isDark ? 'bg-gray-700' : 'bg-gray-100'}`}>
                    <div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Base fee (this chain does not expose fee history)</div>
                    <div className="text-xl font-bold">{weiToGwei(selectedChainGas.gasPriceWei)} Gwei</div>
                  </div>
                )}
              </div>
            )}
          </div>
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'} rounded-lg p-6`}>
            <h2 className="text-xl font-semibold mb-4">Estimated Cost</h2>
            {selectedChainGas && (
              <div className="space-y-4">
                {(['slow', 'standard', 'fast'] as const).map((level) => {
                  const tier = selectedChainGas.tiers?.[level];
                  const fee = tier?.max_fee_wei || (level === 'standard' ? selectedChainGas.gasPriceWei : undefined);
                  const cost = costFromWei(fee, currentGasLimit);
                  return (
                    <div
                      key={level}
                      className={`p-4 rounded-lg ${level === 'standard' ? (isDark ? 'bg-orange-900/30 border-2 border-orange-500' : 'bg-orange-100 border-2 border-orange-500') : (isDark ? 'bg-gray-700' : 'bg-gray-100')}`}
                    >
                      <div className={`text-sm mb-1 ${level === 'standard' ? 'text-orange-600' : isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                        {level === 'standard' ? 'Standard (Recommended)' : level.charAt(0).toUpperCase() + level.slice(1)}
                      </div>
                      <div className={`text-2xl font-bold ${level === 'standard' ? 'text-orange-600' : ''}`}>
                        {cost === null ? 'unavailable' : `${cost} ${selectedChainGas.symbol}`}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
        <div className="mt-8">
          <h2 className="text-xl font-semibold mb-4">All Networks Gas Prices</h2>
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'} rounded-lg overflow-hidden`}>
            <table className="w-full"><thead className={isDark ? 'bg-gray-700' : 'bg-gray-100'}><tr><th className="px-4 py-3 text-left">Network</th><th className="px-4 py-3 text-left">Symbol</th><th className="px-4 py-3 text-left">Slow</th><th className="px-4 py-3 text-left">Standard</th><th className="px-4 py-3 text-left">Fast</th></tr></thead>
            <tbody>{chainsGas.map((chain) => <tr key={chain.chainId} className={`border-t ${isDark ? 'border-gray-700' : 'border-gray-200'}`}><td className="px-4 py-3">{chain.chainName}</td><td className="px-4 py-3">{chain.symbol}</td><td className="px-4 py-3">{chain.tiers ? `${weiToGwei(chain.tiers.slow.max_fee_wei) ?? '-'} Gwei` : '-'}</td><td className="px-4 py-3 font-medium">{chain.tiers ? `${weiToGwei(chain.tiers.standard.max_fee_wei) ?? '-'} Gwei` : `${weiToGwei(chain.gasPriceWei) ?? '-'} Gwei`}</td><td className="px-4 py-3">{chain.tiers ? `${weiToGwei(chain.tiers.fast.max_fee_wei) ?? '-'} Gwei` : '-'}</td></tr>)}</tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
}
