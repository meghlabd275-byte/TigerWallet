"use client";

import { useState } from "react";
import { Search, Grid, List, Star, ExternalLink, TrendingUp, Globe, Wallet, Gamepad2, Palette } from "lucide-react";

interface DApp {
  id: string;
  name: string;
  description: string;
  category: string;
  icon: string;
  url: string;
  rating: number;
  downloads: number;
  verified: boolean;
  featured: boolean;
}

const DAPPS: DApp[] = [
  { id: "1", name: "Uniswap", description: "Decentralized trading protocol", category: "DeFi", icon: "🦄", url: "https://uniswap.org", rating: 4.9, downloads: 5000000, chains: ["Ethereum"], verified: true, featured: true },
  { id: "2", name: "Aave", description: "Non-custodial liquidity protocol", category: "DeFi", icon: "👻", url: "https://aave.com", rating: 4.8, downloads: 3000000, chains: ["Ethereum"], verified: true, featured: true },
  { id: "3", name: "OpenSea", description: "Digital item marketplace", category: "NFT", icon: "🌊", url: "https://opensea.io", rating: 4.7, downloads: 10000000, chains: ["Ethereum"], verified: true, featured: true },
  { id: "4", name: "Magic Eden", description: "NFT marketplace", category: "NFT", icon: "🧙", url: "https://magiceden.io", rating: 4.6, downloads: 5000000, chains: ["Solana"], verified: true, featured: false },
  { id: "5", name: "Raydium", description: "AMM and liquidity platform", category: "DeFi", icon: "🌊", url: "https://raydium.io", rating: 4.5, downloads: 2000000, chains: ["Solana"], verified: true, featured: true },
  { id: "6", name: "Jupiter", description: "Solana DEX aggregator", category: "DeFi", icon: "🪐", url: "https://jup.ag", rating: 4.8, downloads: 3000000, chains: ["Solana"], verified: true, featured: true },
  { id: "7", name: "Blur", description: "NFT marketplace", category: "NFT", icon: "👁", url: "https://blur.io", rating: 4.4, downloads: 1500000, chains: ["Ethereum"], verified: true, featured: false },
  { id: "8", name: "PancakeSwap", description: "AMM on BNB Chain", category: "DeFi", icon: "🥞", url: "https://pancakeswap.finance", rating: 4.5, downloads: 8000000, chains: ["BNB"], verified: true, featured: false },
  { id: "9", name: "GMX", description: "Decentralized perpetual exchange", category: "Trading", icon: "🦊", url: "https://gmx.io", rating: 4.7, downloads: 1000000, chains: ["Arbitrum"], verified: true, featured: true },
  { id: "10", name: "dYdX", description: "Decentralized exchange", category: "Trading", icon: "📈", url: "https://dydx.exchange", rating: 4.6, downloads: 1500000, chains: ["Ethereum"], verified: true, featured: false },
  { id: "11", name: "Stargate", description: "Cross-chain protocol", category: "Bridge", icon: "🌉", url: "https://stargate.finance", rating: 4.5, downloads: 2000000, chains: ["Ethereum"], verified: true, featured: false },
  { id: "12", name: "Axie Infinity", description: "Blockchain game", category: "Game", icon: "🦎", url: "https://axieinfinity.com", rating: 4.3, downloads: 10000000, chains: ["Ronin"], verified: true, featured: true },
  { id: "13", name: "Curve", description: "Stable asset swapping", category: "DeFi", icon: "📊", url: "https://curve.fi", rating: 4.7, downloads: 2500000, chains: ["Ethereum"], verified: true, featured: true },
  { id: "14", name: "1inch", description: "DEX aggregator", category: "DeFi", icon: "🥢", url: "https://1inch.io", rating: 4.6, downloads: 6000000, chains: ["Ethereum"], verified: true, featured: false },
  { id: "15", name: "Compound", description: "Money market protocol", category: "DeFi", icon: "🔷", url: "https://compound.finance", rating: 4.6, downloads: 2000000, chains: ["Ethereum"], verified: true, featured: false },
];

const CATEGORIES = [
  { id: "all", name: "All", icon: Grid },
  { id: "DeFi", name: "DeFi", icon: TrendingUp },
  { id: "NFT", name: "NFT", icon: Palette },
  { id: "Trading", name: "Trading", icon: Wallet },
  { id: "Bridge", name: "Bridge", icon: Globe },
  { id: "Game", name: "Games", icon: Gamepad2 },
];

export default function DAppStorePage() {
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedCategory, setSelectedCategory] = useState("all");
  const [viewMode, setViewMode] = useState<"grid" | "list">("grid");

  const filteredDApps = DAPPS.filter(dapp => {
    const matchesSearch = dapp.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      dapp.description.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesCategory = selectedCategory === "all" || dapp.category === selectedCategory;
    return matchesSearch && matchesCategory;
  });

  const featuredDApps = DAPPS.filter(d => d.featured);

  return (
    <div className="min-h-screen bg-gradient-to-br from-purple-900 via-[#1a1a2e] to-black text-white p-4 md:p-8">
      <div className="max-w-7xl mx-auto">
        <header className="mb-8">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 bg-gradient-to-br from-purple-500 to-purple-600 rounded-xl flex items-center justify-center">
              <Globe className="w-7 h-7" />
            </div>
            <div>
              <h1 className="text-2xl font-bold">TigerWallet</h1>
              <p className="text-gray-400 text-sm">DApp Store</p>
            </div>
          </div>
        </header>

        <section className="mb-8">
          <h2 className="text-xl font-bold mb-4 flex items-center gap-2">
            <Star className="w-5 h-5 text-yellow-400" />
            Featured DApps
          </h2>
          <div className="grid md:grid-cols-3 gap-4">
            {featuredDApps.slice(0, 3).map(dapp => (
              <a key={dapp.id} href={dapp.url} target="_blank" rel="noopener noreferrer"
                className="bg-gray-800/50 border border-gray-700 rounded-xl p-4 hover:border-purple-500 transition-colors">
                <div className="flex items-center gap-3 mb-3">
                  <span className="text-3xl">{dapp.icon}</span>
                  <div>
                    <h3 className="font-bold flex items-center gap-2">
                      {dapp.name}
                      {dapp.verified && <span className="text-blue-400">✓</span>}
                    </h3>
                    <p className="text-gray-400 text-sm">{dapp.category}</p>
                  </div>
                </div>
                <p className="text-gray-400 text-sm mb-3">{dapp.description}</p>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-1">
                    <Star className="w-4 h-4 text-yellow-400 fill-yellow-400" />
                    <span className="text-sm">{dapp.rating}</span>
                  </div>
                  <ExternalLink className="w-4 h-4 text-gray-400" />
                </div>
              </a>
            ))}
          </div>
        </section>

        <div className="flex flex-wrap gap-4 mb-6">
          <div className="flex-1 min-w-[200px]">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input type="text" placeholder="Search DApps..." value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full bg-gray-800 border border-gray-700 rounded-lg pl-10 pr-4 py-2 focus:outline-none focus:ring-2 focus:ring-purple-500" />
            </div>
          </div>
          <div className="flex gap-2">
            {CATEGORIES.map(cat => (
              <button key={cat.id} onClick={() => setSelectedCategory(cat.id)}
                className={`flex items-center gap-2 px-4 py-2 rounded-lg whitespace-nowrap ${
                  selectedCategory === cat.id ? "bg-purple-500 text-white" : "bg-gray-800 text-gray-400 hover:text-white"
                }`}>
                <cat.icon className="w-4 h-4" />
                {cat.name}
              </button>
            ))}
          </div>
          <div className="flex gap-2">
            <button onClick={() => setViewMode("grid")} className={`p-2 rounded-lg ${viewMode === "grid" ? "bg-purple-500" : "bg-gray-800"}`}>
              <Grid className="w-5 h-5" />
            </button>
            <button onClick={() => setViewMode("list")} className={`p-2 rounded-lg ${viewMode === "list" ? "bg-purple-500" : "bg-gray-800"}`}>
              <List className="w-5 h-5" />
            </button>
          </div>
        </div>

        <div className={viewMode === "grid" ? "grid md:grid-cols-3 lg:grid-cols-4 gap-4" : "space-y-3"}>
          {filteredDApps.map(dapp => (
            <a key={dapp.id} href={dapp.url} target="_blank" rel="noopener noreferrer"
              className={`bg-gray-800/50 border border-gray-700 rounded-xl hover:border-purple-500 transition-colors ${
                viewMode === "grid" ? "p-4" : "p-3 flex items-center gap-4"
              }`}>
              <div className="flex items-center gap-3">
                <span className="text-2xl">{dapp.icon}</span>
                <div>
                  <h3 className="font-bold flex items-center gap-2">
                    {dapp.name}
                    {dapp.verified && <span className="text-blue-400">✓</span>}
                  </h3>
                  <p className="text-gray-400 text-sm">{dapp.category}</p>
                </div>
              </div>
              {viewMode === "grid" && <p className="text-gray-400 text-sm my-2">{dapp.description}</p>}
              <div className="flex items-center gap-1 mt-2">
                <Star className="w-4 h-4 text-yellow-400 fill-yellow-400" />
                <span className="text-sm">{dapp.rating}</span>
              </div>
            </a>
          ))}
        </div>
      </div>
    </div>
  );
}
