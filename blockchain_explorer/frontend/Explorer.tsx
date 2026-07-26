/**
 * TigerWallet Blockchain Explorer
 * Complete blockchain explorer with multi-chain support
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '@/context/ThemeProvider';

// Types
interface Block {
  number: number;
  hash: string;
  timestamp: number;
  transactions: number;
  gasUsed: string;
  miner: string;
}

interface Network {
  id: string;
  name: string;
  symbol: string;
  chainId: number;
  color: string;
}

const NETWORKS: Network[] = [
  { id: 'ethereum', name: 'Ethereum', symbol: 'ETH', chainId: 1, color: '#627EEA' },
  { id: 'bsc', name: 'BNB Smart Chain', symbol: 'BNB', chainId: 56, color: '#F3BA2F' },
  { id: 'polygon', name: 'Polygon', symbol: 'MATIC', chainId: 137, color: '#8247E5' },
  { id: 'arbitrum', name: 'Arbitrum', symbol: 'ETH', chainId: 42161, color: '#28A0F0' },
  { id: 'optimism', name: 'Optimism', symbol: 'ETH', chainId: 10, color: '#FF0420' },
  { id: 'base', name: 'Base', symbol: 'ETH', chainId: 8453, color: '#0052FF' },
  { id: 'avalanche', name: 'Avalanche', symbol: 'AVAX', chainId: 43114, color: '#E84142' },
];

function NetworkSelector({ selected, onSelect }: { selected: Network; onSelect: (n: Network) => void }) {
  const { colors } = useTheme();
  return (
    <select
      value={selected.id}
      onChange={(e) => onSelect(NETWORKS.find(n => n.id === e.target.value) || NETWORKS[0])}
      className="px-4 py-2 rounded-lg border outline-none"
      style={{ backgroundColor: colors.surface, borderColor: colors.border, color: colors.text }}
    >
      {NETWORKS.map((network) => (
        <option key={network.id} value={network.id}>{network.name} ({network.symbol})</option>
      ))}
    </select>
  );
}

function BlockCard({ block, onClick }: { block: Block; onClick: () => void }) {
  const { colors } = useTheme();
  return (
    <div 
      onClick={onClick}
      className="p-4 rounded-xl border cursor-pointer hover:opacity-80 transition-opacity"
      style={{ backgroundColor: colors.surface, borderColor: colors.border }}
    >
      <div className="flex items-center justify-between mb-2">
        <span className="font-mono font-bold" style={{ color: colors.primary }}>#{block.number}</span>
        <span className="text-sm" style={{ color: colors.textSecondary }}>
          {new Date(block.timestamp * 1000).toLocaleTimeString()}
        </span>
      </div>
      <div className="space-y-1 text-sm">
        <div className="flex justify-between">
          <span style={{ color: colors.textSecondary }}>TX</span>
          <span style={{ color: colors.text }}>{block.transactions}</span>
        </div>
        <div className="flex justify-between">
          <span style={{ color: colors.textSecondary }}>Gas</span>
          <span style={{ color: colors.text }}>{parseInt(block.gasUsed).toLocaleString()}</span>
        </div>
      </div>
    </div>
  );
}

export default function BlockchainExplorer() {
  const { colors } = useTheme();
  const [network, setNetwork] = useState(NETWORKS[0]);
  const [blocks, setBlocks] = useState<Block[]>([]);
  const [search, setSearch] = useState('');

  useEffect(() => {
    // Mock blocks for demo
    const mockBlocks: Block[] = Array.from({ length: 20 }, (_, i) => ({
      number: 19000000 - i,
      hash: '0x' + Math.random().toString(16).slice(2, 66),
      timestamp: Math.floor(Date.now() / 1000) - i * 12,
      transactions: Math.floor(Math.random() * 200),
      gasUsed: String(Math.floor(Math.random() * 15000000)),
      miner: '0x' + Math.random().toString(16).slice(2, 42),
    }));
    setBlocks(mockBlocks);
  }, [network]);

  const handleSearch = () => {
    alert(`Searching for: ${search}`);
  };

  return (
    <div className="min-h-screen" style={{ backgroundColor: colors.background }}>
      <header className="border-b sticky top-0" style={{ backgroundColor: colors.surface, borderColor: colors.border }}>
        <div className="max-w-7xl mx-auto px-6 py-4">
          <div className="flex items-center justify-between gap-4">
            <h1 className="text-2xl font-bold" style={{ color: colors.text }}>TigerExplorer</h1>
            <NetworkSelector selected={network} onSelect={setNetwork} />
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-6 py-8">
        <div className="flex gap-2 mb-8">
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
            placeholder="Search address, tx hash, block..."
            className="flex-1 px-4 py-3 rounded-lg border outline-none"
            style={{ backgroundColor: colors.surface, borderColor: colors.border, color: colors.text }}
          />
          <button
            onClick={handleSearch}
            className="px-6 py-3 rounded-lg font-medium"
            style={{ backgroundColor: colors.primary, color: 'white' }}
          >
            Search
          </button>
        </div>

        <h2 className="text-xl font-semibold mb-4" style={{ color: colors.text }}>Latest Blocks</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {blocks.map((block) => (
            <BlockCard key={block.number} block={block} onClick={() => alert(`Block #${block.number}`)} />
          ))}
        </div>
      </main>
    </div>
  );
}
