// TigerWallet Desktop - Main App
import React, { useState, useEffect } from 'react';

// Icons
const Icons = {
  Dashboard: () => <span>📊</span>,
  Terminal: () => <span>💹</span>,
  Swap: () => <span>🔄</span>,
  NFT: () => <span>🖼️</span>,
  Settings: () => <span>⚙️</span>,
  Wallet: () => <span>🐯</span>,
  Send: () => <span>📤</span>,
  Receive: () => <span>📥</span>,
};

// Sidebar Component
const Sidebar = ({ currentPage, setCurrentPage }) => {
  const menuItems = [
    { id: 'dashboard', label: 'Dashboard', icon: '📊' },
    { id: 'terminal', label: 'Terminal', icon: '💹' },
    { id: 'swap', label: 'Swap', icon: '🔄' },
    { id: 'nft', label: 'NFTs', icon: '🖼️' },
    { id: 'settings', label: 'Settings', icon: '⚙️' },
  ];

  return (
    <div className="w-64 bg-gray-800 border-r border-gray-700 flex flex-col">
      <div className="p-4 border-b border-gray-700">
        <div className="flex items-center space-x-3">
          <span className="text-2xl">🐯</span>
          <span className="text-xl font-bold">TigerWallet</span>
        </div>
      </div>
      
      <nav className="flex-1 p-4">
        {menuItems.map(item => (
          <button
            key={item.id}
            onClick={() => setCurrentPage(item.id)}
            className={`w-full flex items-center space-x-3 px-4 py-3 rounded-lg mb-2 transition-colors ${
              currentPage === item.id
                ? 'bg-orange-500 text-white'
                : 'text-gray-400 hover:bg-gray-700 hover:text-white'
            }`}
          >
            <span>{item.icon}</span>
            <span>{item.label}</span>
          </button>
        ))}
      </nav>

      <div className="p-4 border-t border-gray-700">
        <div className="bg-gray-700 rounded-lg p-3">
          <div className="text-xs text-gray-400">Network</div>
          <div className="flex items-center space-x-2 mt-1">
            <span className="w-2 h-2 bg-green-500 rounded-full"></span>
            <span>Ethereum</span>
          </div>
        </div>
      </div>
    </div>
  );
};

// Header Component
const Header = () => {
  const [balance, setBalance] = useState('$12,450.00');
  const [address, setAddress] = useState('0x742d...12eB3');

  return (
    <header className="h-16 bg-gray-800 border-b border-gray-700 flex items-center justify-between px-6">
      <div className="flex items-center space-x-4">
        <input
          type="text"
          placeholder="Search..."
          className="px-4 py-2 bg-gray-700 border border-gray-600 rounded-lg text-sm w-64"
        />
      </div>
      
      <div className="flex items-center space-x-4">
        <div className="text-right">
          <div className="text-sm text-gray-400">Balance</div>
          <div className="font-bold">{balance}</div>
        </div>
        
        <div className="flex items-center space-x-2 px-3 py-2 bg-gray-700 rounded-lg">
          <span className="text-sm font-mono">{address}</span>
          <button className="text-gray-400 hover:text-white">📋</button>
        </div>
        
        <button className="p-2 bg-gray-700 rounded-lg hover:bg-gray-600">
          🔔
        </button>
      </div>
    </header>
  );
};

// Dashboard Page
const Dashboard = () => {
  const assets = [
    { symbol: 'ETH', name: 'Ethereum', balance: '4.2', value: '12,600', change: '+2.5%' },
    { symbol: 'USDT', name: 'Tether USD', balance: '1,000', value: '1,000', change: '0%' },
    { symbol: 'BNB', name: 'BNB', balance: '5.2', value: '1,560', change: '+1.2%' },
  ];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Dashboard</h1>
      
      <div className="grid grid-cols-3 gap-6">
        <div className="bg-gradient-to-br from-orange-500 to-red-500 rounded-xl p-6">
          <div className="text-white/80">Total Balance</div>
          <div className="text-3xl font-bold mt-2">$15,160.00</div>
          <div className="text-white/80 mt-2">+2.5% today</div>
        </div>
        
        <div className="bg-gray-800 rounded-xl p-6">
          <div className="text-gray-400">Portfolio</div>
          <div className="text-2xl font-bold mt-2">{assets.length} Assets</div>
          <div className="text-gray-400 mt-2">3 Networks</div>
        </div>
        
        <div className="bg-gray-800 rounded-xl p-6">
          <div className="text-gray-400">Gas Tracker</div>
          <div className="text-2xl font-bold mt-2">25 Gwei</div>
          <div className="text-gray-400 mt-2">~5 min confirmation</div>
        </div>
      </div>

      <div className="bg-gray-800 rounded-xl p-6">
        <h2 className="text-lg font-semibold mb-4">Assets</h2>
        <table className="w-full">
          <thead>
            <tr className="text-left text-gray-400 border-b border-gray-700">
              <th className="pb-3">Asset</th>
              <th className="pb-3">Balance</th>
              <th className="pb-3">Value</th>
              <th className="pb-3">24h</th>
              <th className="pb-3">Actions</th>
            </tr>
          </thead>
          <tbody>
            {assets.map(asset => (
              <tr key={asset.symbol} className="border-b border-gray-700">
                <td className="py-4">
                  <div className="flex items-center space-x-3">
                    <div className="w-10 h-10 bg-orange-500 rounded-full flex items-center justify-center">
                      {asset.symbol[0]}
                    </div>
                    <div>
                      <div className="font-semibold">{asset.symbol}</div>
                      <div className="text-sm text-gray-400">{asset.name}</div>
                    </div>
                  </div>
                </td>
                <td className="py-4">{asset.balance}</td>
                <td className="py-4">${asset.value}</td>
                <td className="py-4 text-green-500">{asset.change}</td>
                <td className="py-4">
                  <div className="flex space-x-2">
                    <button className="px-3 py-1 bg-orange-500 rounded text-sm">Send</button>
                    <button className="px-3 py-1 bg-gray-700 rounded text-sm">Receive</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Terminal Page
const Terminal = () => {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Trading Terminal</h1>
      
      <div className="grid grid-cols-4 gap-4 h-[600px]">
        <div className="col-span-3 bg-gray-800 rounded-xl p-4">
          <div className="flex space-x-2 mb-4">
            {['ETH/USDT', 'BTC/USDT', 'SOL/USDT'].map(pair => (
              <button key={pair} className="px-4 py-2 bg-gray-700 rounded-lg hover:bg-gray-600">
                {pair}
              </button>
            ))}
          </div>
          
          <div className="h-full flex items-center justify-center bg-gray-900 rounded-lg">
            <div className="text-center">
              <div className="text-4xl font-bold">$3,500.00</div>
              <div className="text-green-500 mt-2">+2.5% (24h)</div>
              <div className="text-gray-400 mt-4">Professional charting</div>
            </div>
          </div>
        </div>
        
        <div className="bg-gray-800 rounded-xl p-4">
          <h3 className="font-semibold mb-4">Order Book</h3>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between text-red-500">
              <span>3501.50</span>
              <span>20.0</span>
            </div>
            <div className="flex justify-between text-red-500">
              <span>3501.00</span>
              <span>30.0</span>
            </div>
            <div className="flex justify-between text-gray-400 border-t border-b border-gray-700 py-2">
              <span>3500.50</span>
              <span>Spread</span>
            </div>
            <div className="flex justify-between text-green-500">
              <span>3500.00</span>
              <span>15.0</span>
            </div>
            <div className="flex justify-between text-green-500">
              <span>3499.50</span>
              <span>25.0</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

// Swap Page
const Swap = () => {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Swap</h1>
      
      <div className="max-w-lg mx-auto">
        <div className="bg-gray-800 rounded-xl p-6">
          <div className="flex justify-between items-center mb-4">
            <span className="font-semibold">Exchange</span>
            <span className="text-gray-400 text-sm">Best rates across DEXs</span>
          </div>
          
          <div className="space-y-4">
            <div className="bg-gray-700 rounded-lg p-4">
              <div className="text-sm text-gray-400">You Pay</div>
              <div className="flex justify-between items-center mt-2">
                <input
                  type="number"
                  placeholder="0.0"
                  className="bg-transparent text-2xl font-bold outline-none w-32"
                />
                <div className="flex items-center space-x-2">
                  <div className="w-8 h-8 bg-orange-500 rounded-full flex items-center justify-center">E</div>
                  <span>ETH</span>
                </div>
              </div>
            </div>
            
            <div className="flex justify-center">
              <button className="p-2 bg-gray-700 rounded-full">↓</button>
            </div>
            
            <div className="bg-gray-700 rounded-lg p-4">
              <div className="text-sm text-gray-400">You Receive</div>
              <div className="flex justify-between items-center mt-2">
                <input
                  type="number"
                  placeholder="0.0"
                  className="bg-transparent text-2xl font-bold outline-none w-32"
                />
                <div className="flex items-center space-x-2">
                  <div className="w-8 h-8 bg-blue-500 rounded-full flex items-center justify-center">U</div>
                  <span>USDT</span>
                </div>
              </div>
            </div>
            
            <button className="w-full py-3 bg-orange-500 rounded-lg font-semibold hover:bg-orange-600">
              Swap Now
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

// NFT Page
const NFT = () => {
  const nfts = [
    { id: 1, name: 'CryptoPunk #1234', collection: 'CryptoPunks', image: '🧑‍🎤' },
    { id: 2, name: 'Bored Ape #5678', collection: 'BAYC', image: '🦍' },
    { id: 3, name: 'Azuki #9012', collection: 'Azuki', image: '🍡' },
  ];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">NFT Collection</h1>
      
      <div className="grid grid-cols-3 gap-6">
        {nfts.map(nft => (
          <div key={nft.id} className="bg-gray-800 rounded-xl overflow-hidden">
            <div className="h-48 bg-gray-700 flex items-center justify-center text-6xl">
              {nft.image}
            </div>
            <div className="p-4">
              <div className="text-sm text-gray-400">{nft.collection}</div>
              <div className="font-semibold mt-1">{nft.name}</div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

// Settings Page
const Settings = () => {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Settings</h1>
      
      <div className="bg-gray-800 rounded-xl p-6 space-y-6">
        <div className="flex justify-between items-center border-b border-gray-700 pb-4">
          <div>
            <div className="font-semibold">Network</div>
            <div className="text-sm text-gray-400">Select default network</div>
          </div>
          <select className="bg-gray-700 px-4 py-2 rounded-lg">
            <option>Ethereum</option>
            <option>BNB Chain</option>
            <option>Polygon</option>
          </select>
        </div>
        
        <div className="flex justify-between items-center border-b border-gray-700 pb-4">
          <div>
            <div className="font-semibold">Currency</div>
            <div className="text-sm text-gray-400">Display currency</div>
          </div>
          <select className="bg-gray-700 px-4 py-2 rounded-lg">
            <option>USD</option>
            <option>EUR</option>
            <option>GBP</option>
          </select>
        </div>
        
        <div className="flex justify-between items-center border-b border-gray-700 pb-4">
          <div>
            <div className="font-semibold">Theme</div>
            <div className="text-sm text-gray-400">Dark/Light mode</div>
          </div>
          <button className="px-4 py-2 bg-gray-700 rounded-lg">Dark</button>
        </div>
        
        <div className="flex justify-between items-center">
          <div>
            <div className="font-semibold">Security</div>
            <div className="text-sm text-gray-400">Biometric, 2FA</div>
          </div>
          <button className="px-4 py-2 bg-orange-500 rounded-lg">Configure</button>
        </div>
      </div>
    </div>
  );
};

// Main App Component
const DesktopApp = () => {
  const [currentPage, setCurrentPage] = useState('dashboard');
  const [isUnlocked, setIsUnlocked] = useState(false);

  if (!isUnlocked) {
    return <LoginScreen onUnlock={() => setIsUnlocked(true)} />;
  }

  return (
    <div className="flex h-screen bg-gray-900 text-white">
      <Sidebar currentPage={currentPage} setCurrentPage={setCurrentPage} />
      <div className="flex-1 flex flex-col overflow-hidden">
        <Header />
        <main className="flex-1 overflow-auto p-6">
          {currentPage === 'dashboard' && <Dashboard />}
          {currentPage === 'terminal' && <Terminal />}
          {currentPage === 'swap' && <Swap />}
          {currentPage === 'nft' && <NFT />}
          {currentPage === 'settings' && <Settings />}
        </main>
      </div>
    </div>
  );
};

export default DesktopApp;
