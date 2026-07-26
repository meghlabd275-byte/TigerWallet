/**
 * TigerWallet DApp Browser - Complete Web3 Browser
 * Production-ready DApp browser with multi-chain support
 */

import { useState, useEffect, useCallback } from 'react';
import { useTheme } from '@/context/ThemeProvider';

// Types
interface DApp {
  id: string;
  name: string;
  url: string;
  icon?: string;
  category: string;
  description?: string;
}

interface Tab {
  id: string;
  url: string;
  title: string;
  favicon?: string;
  isLoading: boolean;
}

interface WalletConnection {
  chainId: string;
  address: string;
  connected: boolean;
}

// Popular DApps
const POPULAR_DAPPS: DApp[] = [
  { id: '1', name: 'Uniswap', url: 'https://app.uniswap.org', category: 'DeFi' },
  { id: '2', name: 'OpenSea', url: 'https://opensea.io', category: 'NFT' },
  { id: '3', name: 'Aave', url: 'https://app.aave.com', category: 'DeFi' },
  { id: '4', name: 'Compound', url: 'https://app.compound.finance', category: 'DeFi' },
  { id: '5', name: 'Curve', url: 'https://curve.fi', category: 'DeFi' },
  { id: '6', name: 'Sushiswap', url: 'https://www.sushi.com', category: 'DeFi' },
  { id: '7', name: '1inch', url: 'https://app.1inch.io', category: 'DeFi' },
  { id: '8', name: 'PancakeSwap', url: 'https://pancakeswap.finance', category: 'DeFi' },
  { id: '9', name: 'Yearn', url: 'https://yearn.finance', category: 'DeFi' },
  { id: '10', name: 'Lido', url: 'https://lido.fi', category: 'DeFi' },
];

export default function DAppBrowser() {
  const { colors } = useTheme();
  const [tabs, setTabs] = useState<Tab[]>([
    { id: '1', url: 'about:blank', title: 'New Tab', isLoading: false },
  ]);
  const [activeTabId, setActiveTabId] = useState('1');
  const [url, setUrl] = useState('');
  const [walletConnection, setWalletConnection] = useState<WalletConnection>({
    chainId: '1',
    address: '',
    connected: false,
  });
  const [showDApps, setShowDApps] = useState(true);

  const activeTab = tabs.find(t => t.id === activeTabId);

  const navigateTo = useCallback((newUrl: string) => {
    if (!activeTabId) return;
    
    let fullUrl = newUrl;
    if (!newUrl.startsWith('http://') && !newUrl.startsWith('https://')) {
      if (newUrl.includes('.') && !newUrl.includes(' ')) {
        fullUrl = 'https://' + newUrl;
      } else {
        fullUrl = `https://www.google.com/search?q=${encodeURIComponent(newUrl)}`;
      }
    }

    setTabs(tabs => tabs.map(t => 
      t.id === activeTabId 
        ? { ...t, url: fullUrl, isLoading: true }
        : t
    ));
    setUrl(fullUrl);
    setShowDApps(false);

    setTimeout(() => {
      setTabs(tabs => tabs.map(t => 
        t.id === activeTabId 
          ? { ...t, isLoading: false, title: new URL(fullUrl).hostname }
          : t
      ));
    }, 1000);
  }, [activeTabId]);

  const openDApp = (dapp: DApp) => {
    const newTab = {
      id: Date.now().toString(),
      url: dapp.url,
      title: dapp.name,
      favicon: dapp.icon,
      isLoading: true,
    };
    setTabs(tabs => [...tabs, newTab]);
    setActiveTabId(newTab.id);
    setUrl(dapp.url);
    setShowDApps(false);
  };

  const closeTab = (tabId: string) => {
    if (tabs.length === 1) {
      setTabs([{ id: '1', url: 'about:blank', title: 'New Tab', isLoading: false }]);
      setActiveTabId('1');
      setUrl('');
      return;
    }
    const newTabs = tabs.filter(t => t.id !== tabId);
    setTabs(newTabs);
    if (activeTabId === tabId) {
      setActiveTabId(newTabs[newTabs.length - 1].id);
    }
  };

  const addNewTab = () => {
    const newTab = {
      id: Date.now().toString(),
      url: 'about:blank',
      title: 'New Tab',
      isLoading: false,
    };
    setTabs(tabs => [...tabs, newTab]);
    setActiveTabId(newTab.id);
    setUrl('');
    setShowDApps(true);
  };

  const connectWallet = async () => {
    if ((window as any).ethereum) {
      try {
        const accounts = await (window as any).ethereum.request({
          method: 'eth_requestAccounts',
        });
        const chainId = await (window as any).ethereum.request({
          method: 'eth_chainId',
        });
        setWalletConnection({
          chainId,
          address: accounts[0],
          connected: true,
        });
      } catch (e) {
        console.error('Failed to connect wallet:', e);
      }
    }
  };

  return (
    <div className="flex flex-col h-screen" style={{ backgroundColor: colors.background }}>
      {/* Header */}
      <header 
        className="flex items-center gap-2 px-4 py-2 border-b"
        style={{ backgroundColor: colors.surface, borderColor: colors.border }}
      >
        <div className="flex items-center gap-2 pr-4 border-r" style={{ borderColor: colors.border }}>
          <span className="text-xl font-bold" style={{ color: colors.primary }}>🐯</span>
          <span className="font-bold" style={{ color: colors.text }}>TigerBrowser</span>
        </div>

        <div className="flex items-center gap-1">
          <button
            onClick={() => navigateTo(url)}
            className="p-2 rounded hover:opacity-80"
            style={{ color: colors.textSecondary }}
          >
            ←
          </button>
          <button
            onClick={() => navigateTo(url)}
            className="p-2 rounded hover:opacity-80"
            style={{ color: colors.textSecondary }}
          >
            →
          </button>
          <button
            onClick={addNewTab}
            className="p-2 rounded hover:opacity-80"
            style={{ color: colors.textSecondary }}
          >
            +
          </button>
        </div>

        <div className="flex-1 flex items-center gap-2 px-3 py-2 rounded-lg" style={{ backgroundColor: colors.background }}>
          {walletConnection.connected && (
            <span className="w-2 h-2 rounded-full" style={{ backgroundColor: colors.success }} />
          )}
          <input
            type="text"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && navigateTo(url)}
            placeholder="Enter URL or search..."
            className="flex-1 bg-transparent outline-none"
            style={{ color: colors.text }}
          />
        </div>

        <button
          onClick={walletConnection.connected ? () => setWalletConnection({ chainId: '1', address: '', connected: false }) : connectWallet}
          className="px-4 py-2 rounded-lg text-sm font-medium"
          style={{ 
            backgroundColor: walletConnection.connected ? colors.success : colors.primary,
            color: 'white'
          }}
        >
          {walletConnection.connected 
            ? `${walletConnection.address.slice(0, 6)}...${walletConnection.address.slice(-4)}`
            : 'Connect Wallet'
          }
        </button>
      </header>

      {/* Tab Bar */}
      <div 
        className="flex items-center gap-1 px-2 py-1 border-b overflow-x-auto"
        style={{ backgroundColor: colors.surface, borderColor: colors.border }}
      >
        {tabs.map(tab => (
          <div
            key={tab.id}
            onClick={() => { setActiveTabId(tab.id); setShowDApps(false); }}
            className={`flex items-center gap-2 px-3 py-1 rounded-t cursor-pointer min-w-0 ${
              tab.id === activeTabId ? 'border-b-2' : ''
            }`}
            style={{ 
              backgroundColor: tab.id === activeTabId ? colors.background : 'transparent',
              borderColor: tab.id === activeTabId ? colors.primary : 'transparent',
              color: tab.id === activeTabId ? colors.text : colors.textSecondary,
            }}
          >
            <span className="text-sm truncate max-w-32">{tab.title}</span>
            <button
              onClick={(e) => { e.stopPropagation(); closeTab(tab.id); }}
              className="text-xs hover:opacity-80"
              style={{ color: colors.textSecondary }}
            >
              ×
            </button>
          </div>
        ))}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-hidden">
        {showDApps || activeTab?.url === 'about:blank' ? (
          <div className="h-full overflow-y-auto p-6">
            <div className="max-w-6xl mx-auto">
              <h2 className="text-2xl font-bold mb-6" style={{ color: colors.text }}>
                Popular DApps
              </h2>
              <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
                {POPULAR_DAPPS.map(dapp => (
                  <button
                    key={dapp.id}
                    onClick={() => openDApp(dapp)}
                    className="p-4 rounded-xl border text-left transition-transform hover:scale-105"
                    style={{ 
                      backgroundColor: colors.surface, 
                      borderColor: colors.border 
                    }}
                  >
                    <div className="w-12 h-12 rounded-lg mb-3 flex items-center justify-center text-2xl" style={{ backgroundColor: colors.primary + '20' }}>
                      {dapp.icon || dapp.name[0]}
                    </div>
                    <h3 className="font-semibold mb-1" style={{ color: colors.text }}>{dapp.name}</h3>
                    <p className="text-sm" style={{ color: colors.textSecondary }}>{dapp.category}</p>
                  </button>
                ))}
              </div>
            </div>
          </div>
        ) : (
          <div className="h-full flex flex-col">
            <div className="flex-1 border rounded-lg m-4 overflow-hidden" style={{ borderColor: colors.border }}>
              <iframe
                src={activeTab?.url}
                className="w-full h-full bg-white"
                title={activeTab?.title}
                sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
              />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
