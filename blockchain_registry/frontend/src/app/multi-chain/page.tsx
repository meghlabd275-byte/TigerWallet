"use client";

import { useState, useCallback, useEffect } from "react";
import {
  Search,
  Grid,
  List,
  CheckCircle,
  Zap,
  Layers,
  RefreshCw,
  Plus,
  AlertCircle,
  Loader2,
} from "lucide-react";

// Chain shape returned by the canonical go/wallet_api registry
// (GET /api/v1/chains — 120 EVM + 66 non-EVM mainnet chains).
interface ChainInfo {
  chain_id: number;
  name: string;
  symbol: string;
  chain_type: string; // "evm" | "non-evm"
  rpc_url: string;
  explorer_url: string;
  decimals: number;
  coin_type: number;
  is_testnet: boolean;
}

// Frontend display model derived from the backend ChainInfo.
interface Blockchain {
  id: number;
  name: string;
  symbol: string;
  type: "evm" | "non-evm";
  rpcUrl: string;
  explorer: string;
  chainId: number;
  isTestnet: boolean;
}

const API_BASE_URL =
  typeof window !== "undefined"
    ? ""
    : (process.env.BACKEND_URL || "http://localhost:8443");

async function fetchChains(): Promise<Blockchain[]> {
  const res = await fetch(`${API_BASE_URL}/api/v1/chains`, {
    headers: { "Content-Type": "application/json" },
  });
  if (!res.ok) throw new Error(`Failed to load chains (HTTP ${res.status})`);
  const data = await res.json();
  const raw: ChainInfo[] = data.chains || data || [];
  return raw.map((c) => ({
    id: c.chain_id,
    chainId: c.chain_id,
    name: c.name,
    symbol: c.symbol,
    type: (c.chain_type || "evm").includes("non") ? "non-evm" : "evm",
    rpcUrl: c.rpc_url || "",
    explorer: c.explorer_url || "",
    isTestnet: !!c.is_testnet,
  }));
}

export default function MultiChainPage() {
  const [chains, setChains] = useState<Blockchain[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [filterType, setFilterType] = useState<"all" | "evm" | "non-evm">("all");
  const [filterTestnet, setFilterTestnet] = useState<"all" | "mainnet" | "testnet">("all");
  const [viewMode, setViewMode] = useState<"grid" | "list">("grid");
  const [selectedChains, setSelectedChains] = useState<number[]>([]);
  const [isConnecting, setIsConnecting] = useState(false);
  const [connectResult, setConnectResult] = useState<string | null>(null);
  const [theme, setTheme] = useState<"dark" | "light">("dark");

  useEffect(() => {
    if (typeof window !== "undefined") {
      const saved = localStorage.getItem("tw-theme") as "dark" | "light" | null;
      if (saved) setTheme(saved);
    }
  }, []);

  const toggleTheme = useCallback(() => {
    setTheme(prev => {
      const next = prev === "dark" ? "light" : "dark";
      if (typeof window !== "undefined") localStorage.setItem("tw-theme", next);
      return next;
    });
  }, []);

  const isDark = theme === "dark";

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchChains();
      setChains(data);
    } catch (err: any) {
      setError(err.message || "Failed to load chains from the registry");
      setChains([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  // Filter chains
  const filteredChains = chains.filter(chain => {
    const matchesSearch = chain.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      chain.symbol.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesType = filterType === "all" || chain.type === filterType;
    const matchesTestnet = filterTestnet === "all" ||
      (filterTestnet === "testnet" && chain.isTestnet) ||
      (filterTestnet === "mainnet" && !chain.isTestnet);
    return matchesSearch && matchesType && matchesTestnet;
  });

  // Stats
  const evmCount = chains.filter(c => c.type === "evm" && !c.isTestnet).length;
  const nonEvmCount = chains.filter(c => c.type === "non-evm" && !c.isTestnet).length;
  const testnetCount = chains.filter(c => c.isTestnet).length;

  // Toggle chain selection
  const toggleChain = useCallback((chainId: number) => {
    setSelectedChains(prev =>
      prev.includes(chainId)
        ? prev.filter(id => id !== chainId)
        : [...prev, chainId]
    );
  }, []);

  // "Connect" = record the user's chain enablement selection. There is no
  // server-side "connect" action (chains are always readable from the
  // registry); this confirms the selection the wallet client will activate
  // for balance/tx fetches. No fake delay.
  const connectAll = useCallback(async () => {
    setIsConnecting(true);
    setConnectResult(null);
    try {
      if (typeof window !== "undefined") {
        localStorage.setItem("tigerwallet_active_chains", JSON.stringify(selectedChains));
      }
      setConnectResult(`Activated ${selectedChains.length} chain${selectedChains.length === 1 ? "" : "s"}.`);
    } finally {
      setIsConnecting(false);
    }
  }, [selectedChains]);

  return (
    <div className={`min-h-screen p-4 md:p-8 ${isDark ? "bg-gradient-to-br from-blue-900 via-[#1a1a2e] to-black text-white" : "bg-gradient-to-br from-slate-50 to-slate-200 text-slate-900"}`}>
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <header className="flex flex-col md:flex-row justify-between items-center mb-8 gap-4">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 bg-gradient-to-br from-blue-500 to-blue-600 rounded-xl flex items-center justify-center">
              <Layers className="w-7 h-7 text-white" />
            </div>
            <div>
              <h1 className="text-2xl font-bold">TigerWallet</h1>
              <p className={`text-sm ${isDark ? "text-gray-400" : "text-slate-500"}`}>Multi-Chain Support</p>
            </div>
          </div>
          <div className="flex items-center gap-4">
            <div className={`${isDark ? "bg-gray-800/50 border-gray-700" : "bg-white border-slate-200"} border rounded-lg px-4 py-2`}>
              <p className={`text-xs ${isDark ? "text-gray-400" : "text-slate-500"}`}>Total Chains</p>
              <p className="font-bold">{chains.length}</p>
            </div>
            <div className={`${isDark ? "bg-gray-800/50 border-gray-700" : "bg-white border-slate-200"} border rounded-lg px-4 py-2`}>
              <p className={`text-xs ${isDark ? "text-gray-400" : "text-slate-500"}`}>EVM Chains</p>
              <p className="font-bold">{evmCount}</p>
            </div>
            <div className={`${isDark ? "bg-gray-800/50 border-gray-700" : "bg-white border-slate-200"} border rounded-lg px-4 py-2`}>
              <p className={`text-xs ${isDark ? "text-gray-400" : "text-slate-500"}`}>Non-EVM</p>
              <p className="font-bold">{nonEvmCount}</p>
            </div>
            <button
              onClick={toggleTheme}
              className={`${isDark ? "bg-gray-800 border-gray-700" : "bg-white border-slate-200"} border rounded-lg p-2`}
              aria-label="Toggle theme"
            >
              {isDark ? "☀️" : "🌙"}
            </button>
          </div>
        </header>

        {/* Error / Loading banners */}
        {error && (
          <div className="mb-6 flex items-center gap-2 p-4 bg-red-500/20 border border-red-500/50 rounded-xl text-red-200">
            <AlertCircle className="w-5 h-5 shrink-0" />
            <span className="flex-1">{error}</span>
            <button onClick={load} className="bg-red-500/30 hover:bg-red-500/50 px-3 py-1 rounded-lg text-sm">
              Retry
            </button>
          </div>
        )}
        {loading && (
          <div className="mb-6 flex items-center gap-2 p-4 bg-gray-800/50 border border-gray-700 rounded-xl">
            <Loader2 className="w-5 h-5 animate-spin" />
            <span>Loading chains from the registry…</span>
          </div>
        )}
        {connectResult && (
          <div className="mb-6 flex items-center gap-2 p-4 bg-green-500/20 border border-green-500/50 rounded-xl text-green-200">
            <CheckCircle className="w-5 h-5 shrink-0" />
            <span>{connectResult}</span>
          </div>
        )}

        {/* Filters */}
        <div className="flex flex-wrap gap-4 mb-6">
          <div className="flex-1 min-w-[200px]">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                type="text"
                placeholder="Search chains..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className={`w-full ${isDark ? "bg-gray-800 border-gray-700" : "bg-white border-slate-200"} border rounded-lg pl-10 pr-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500`}
              />
            </div>
          </div>

          <select
            value={filterType}
            onChange={(e) => setFilterType(e.target.value as any)}
            className={`${isDark ? "bg-gray-800 border-gray-700" : "bg-white border-slate-200"} border rounded-lg px-4 py-2`}
          >
            <option value="all">All Types</option>
            <option value="evm">EVM Only</option>
            <option value="non-evm">Non-EVM Only</option>
          </select>

          <select
            value={filterTestnet}
            onChange={(e) => setFilterTestnet(e.target.value as any)}
            className="bg-gray-800 border border-gray-700 rounded-lg px-4 py-2"
          >
            <option value="all">Mainnet & Testnet</option>
            <option value="mainnet">Mainnet Only</option>
            <option value="testnet">Testnet Only</option>
          </select>

          <div className="flex gap-2">
            <button
              onClick={() => setViewMode("grid")}
              className={`p-2 rounded-lg ${viewMode === "grid" ? "bg-blue-500" : "bg-gray-800"}`}
            >
              <Grid className="w-5 h-5" />
            </button>
            <button
              onClick={() => setViewMode("list")}
              className={`p-2 rounded-lg ${viewMode === "list" ? "bg-blue-500" : "bg-gray-800"}`}
            >
              <List className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Selected Actions */}
        {selectedChains.length > 0 && (
          <div className="mb-6 p-4 bg-blue-500/20 border border-blue-500/50 rounded-xl flex items-center justify-between">
            <p>{selectedChains.length} chains selected</p>
            <button
              onClick={connectAll}
              disabled={isConnecting}
              className="bg-blue-500 hover:bg-blue-600 px-6 py-2 rounded-lg font-medium flex items-center gap-2"
            >
              {isConnecting ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Zap className="w-4 h-4" />}
              Activate Selected
            </button>
          </div>
        )}

        {/* Chain Grid/List */}
        {!loading && viewMode === "grid" ? (
          <div className="grid md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {filteredChains.map(chain => (
              <div
                key={chain.id}
                onClick={() => toggleChain(chain.id)}
                className={`${isDark ? "bg-gray-800/50 border-gray-700" : "bg-white border-slate-200"} border rounded-xl p-4 cursor-pointer transition-all ${
                  selectedChains.includes(chain.id)
                    ? "border-blue-500 bg-blue-500/10"
                    : "border-gray-700 hover:border-gray-600"
                }`}
              >
                <div className="flex items-center justify-between mb-3">
                  <div>
                    <p className="font-bold">{chain.name}</p>
                    <p className="text-gray-400 text-sm">{chain.symbol}</p>
                  </div>
                  {selectedChains.includes(chain.id) && (
                    <CheckCircle className="w-5 h-5 text-blue-500" />
                  )}
                </div>

                <div className="flex items-center justify-between">
                  <div className="flex gap-1 flex-wrap">
                    {chain.isTestnet ? (
                      <span className="px-2 py-0.5 bg-yellow-500/20 text-yellow-400 text-xs rounded">Testnet</span>
                    ) : (
                      <span className="px-2 py-0.5 bg-green-500/20 text-green-400 text-xs rounded">Mainnet</span>
                    )}
                    <span className="px-2 py-0.5 bg-gray-700 text-gray-400 text-xs rounded capitalize">
                      {chain.type}
                    </span>
                  </div>
                  <span className="text-gray-400 text-xs">ID: {chain.id}</span>
                </div>
              </div>
            ))}
          </div>
        ) : !loading && (
          <div className={`${isDark ? "bg-gray-800/50 border-gray-700" : "bg-white border-slate-200"} border rounded-xl overflow-hidden`}>
            <table className="w-full">
              <thead className={isDark ? "bg-gray-900/50" : "bg-slate-100"}>
                <tr>
                  <th className="px-4 py-3 text-left text-gray-400">Chain</th>
                  <th className="px-4 py-3 text-left text-gray-400">Type</th>
                  <th className="px-4 py-3 text-left text-gray-400">Chain ID</th>
                  <th className="px-4 py-3 text-left text-gray-400">Status</th>
                  <th className="px-4 py-3 text-right text-gray-400">Action</th>
                </tr>
              </thead>
              <tbody>
                {filteredChains.map(chain => (
                  <tr
                    key={chain.id}
                    className={`border-t ${isDark ? "border-gray-700 hover:bg-gray-800/50" : "border-slate-200 hover:bg-slate-50"} cursor-pointer`}
                    onClick={() => toggleChain(chain.id)}
                  >
                    <td className="px-4 py-3">
                      <p className="font-bold">{chain.name}</p>
                      <p className="text-gray-400 text-sm">{chain.symbol}</p>
                    </td>
                    <td className="px-4 py-3 capitalize">{chain.type}</td>
                    <td className="px-4 py-3 font-mono">{chain.chainId}</td>
                    <td className="px-4 py-3">
                      {chain.isTestnet ? (
                        <span className="px-2 py-0.5 bg-yellow-500/20 text-yellow-400 text-xs rounded">Testnet</span>
                      ) : (
                        <span className="px-2 py-0.5 bg-green-500/20 text-green-400 text-xs rounded">Mainnet</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-right">
                      {selectedChains.includes(chain.id) ? (
                        <CheckCircle className="w-5 h-5 text-blue-500 inline" />
                      ) : (
                        <Plus className="w-5 h-5 text-gray-400 inline" />
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Empty State */}
        {!loading && filteredChains.length === 0 && (
          <div className="text-center py-12">
            <Layers className="w-12 h-12 mx-auto mb-4 text-gray-500" />
            <p className="text-gray-400">No chains found matching your filters</p>
          </div>
        )}

        {/* Quick Stats */}
        {!loading && chains.length > 0 && (
          <div className="mt-8 grid md:grid-cols-4 gap-4">
            <div className={`${isDark ? "bg-gray-800/50 border-gray-700" : "bg-white border-slate-200"} border rounded-xl p-4`}>
              <p className={`text-sm mb-1 ${isDark ? "text-gray-400" : "text-slate-500"}`}>EVM Mainnet</p>
              <p className="font-bold text-lg">{evmCount}</p>
            </div>
            <div className={`${isDark ? "bg-gray-800/50 border-gray-700" : "bg-white border-slate-200"} border rounded-xl p-4`}>
              <p className={`text-sm mb-1 ${isDark ? "text-gray-400" : "text-slate-500"}`}>Non-EVM Mainnet</p>
              <p className="font-bold text-lg">{nonEvmCount}</p>
            </div>
            <div className={`${isDark ? "bg-gray-800/50 border-gray-700" : "bg-white border-slate-200"} border rounded-xl p-4`}>
              <p className={`text-sm mb-1 ${isDark ? "text-gray-400" : "text-slate-500"}`}>Testnets</p>
              <p className="font-bold text-lg">{testnetCount}</p>
            </div>
            <div className={`${isDark ? "bg-gray-800/50 border-gray-700" : "bg-white border-slate-200"} border rounded-xl p-4`}>
              <p className={`text-sm mb-1 ${isDark ? "text-gray-400" : "text-slate-500"}`}>Total Registry</p>
              <p className="font-bold text-lg">{chains.length}</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
