'use client';

import React, { useState, useEffect } from 'react';
import { useTheme } from '../components/ThemeProvider';

interface GasPrice {
  slow: string;
  standard: string;
  fast: string;
  slowWait: string;
  standardWait: string;
  fastWait: string;
}

interface ChainGas {
  chainId: number;
  chainName: string;
  symbol: string;
  gasPrice: GasPrice;
}

export default function GasEstimation() {
  const [chainsGas, setChainsGas] = useState<ChainGas[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedChain, setSelectedChain] = useState<number>(1);
  const [txType, setTxType] = useState<string>('transfer');
  const [gasLimit, setGasLimit] = useState<string>('21000');
  const { isDark } = useTheme();

  useEffect(() => {
    setChainsGas([]);
    setError('Live gas estimates are unavailable until an authenticated multi-chain gas provider is configured.');
    setLoading(false);
  }, []);

  const calculateCost = (gasPrice: string, limit: string): string => {
    const price = parseFloat(gasPrice);
    const limitNum = parseFloat(limit);
    return (price * limitNum / 1000000000).toFixed(6);
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
                <div className={`p-4 ${isDark ? 'bg-gray-700' : 'bg-gray-100'} rounded-lg flex justify-between items-center`}><div><div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Slow</div><div className="text-xl font-bold">{selectedChainGas.gasPrice.slow} Gwei</div></div><div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{selectedChainGas.gasPrice.slowWait}</div></div>
                <div className={`p-4 ${isDark ? 'bg-orange-900/30' : 'bg-orange-100'} rounded-lg flex justify-between items-center border-2 border-orange-500`}><div><div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Standard (Recommended)</div><div className="text-xl font-bold text-orange-600">{selectedChainGas.gasPrice.standard} Gwei</div></div><div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{selectedChainGas.gasPrice.standardWait}</div></div>
                <div className={`p-4 ${isDark ? 'bg-gray-700' : 'bg-gray-100'} rounded-lg flex justify-between items-center`}><div><div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Fast</div><div className="text-xl font-bold">{selectedChainGas.gasPrice.fast} Gwei</div></div><div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{selectedChainGas.gasPrice.fastWait}</div></div>
              </div>
            )}
          </div>
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'} rounded-lg p-6`}>
            <h2 className="text-xl font-semibold mb-4">Estimated Cost</h2>
            {selectedChainGas && (
              <div className="space-y-4">
                <div className={`p-4 ${isDark ? 'bg-gray-700' : 'bg-gray-100'} rounded-lg`}><div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'} mb-1`}>Slow</div><div className="text-2xl font-bold">{calculateCost(selectedChainGas.gasPrice.slow, currentGasLimit)} {selectedChainGas.symbol}</div></div>
                <div className={`p-4 ${isDark ? 'bg-orange-900/30' : 'bg-orange-100'} rounded-lg border-2 border-orange-500`}><div className="text-sm text-orange-600 mb-1">Standard (Recommended)</div><div className="text-2xl font-bold text-orange-600">{calculateCost(selectedChainGas.gasPrice.standard, currentGasLimit)} {selectedChainGas.symbol}</div></div>
                <div className={`p-4 ${isDark ? 'bg-gray-700' : 'bg-gray-100'} rounded-lg`}><div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'} mb-1`}>Fast</div><div className="text-2xl font-bold">{calculateCost(selectedChainGas.gasPrice.fast, currentGasLimit)} {selectedChainGas.symbol}</div></div>
              </div>
            )}
          </div>
        </div>
        <div className="mt-8">
          <h2 className="text-xl font-semibold mb-4">All Networks Gas Prices</h2>
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'} rounded-lg overflow-hidden`}>
            <table className="w-full"><thead className={isDark ? 'bg-gray-700' : 'bg-gray-100'}><tr><th className="px-4 py-3 text-left">Network</th><th className="px-4 py-3 text-left">Symbol</th><th className="px-4 py-3 text-left">Slow</th><th className="px-4 py-3 text-left">Standard</th><th className="px-4 py-3 text-left">Fast</th></tr></thead>
            <tbody>{chainsGas.map((chain) => <tr key={chain.chainId} className={`border-t ${isDark ? 'border-gray-700' : 'border-gray-200'}`}><td className="px-4 py-3">{chain.chainName}</td><td className="px-4 py-3">{chain.symbol}</td><td className="px-4 py-3">{chain.gasPrice.slow} Gwei</td><td className="px-4 py-3 font-medium">{chain.gasPrice.standard} Gwei</td><td className="px-4 py-3">{chain.gasPrice.fast} Gwei</td></tr>)}</tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
}
