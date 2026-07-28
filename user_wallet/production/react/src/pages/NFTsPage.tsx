/**
 * NFTs Page - View and manage NFT collections
 */

import React, { useState } from 'react';
import { useTheme } from '../contexts/ThemeContext';

function NFTsPage() {
  const { theme } = useTheme();
  const [activeTab, setActiveTab] = useState('collected');

  const nfts = [
    { id: 1, name: 'Bored Ape #1234', collection: 'Bored Ape Yacht Club', image: '🦍', floor: '45 ETH' },
    { id: 2, name: 'CryptoPunk #5678', collection: 'CryptoPunks', image: '👽', floor: '35 ETH' },
    { id: 3, name: 'Azuki #9012', collection: 'Azuki', image: '🥷', floor: '12 ETH' },
  ];

  const collections = [
    { name: 'Bored Ape Yacht Club', count: 1, floor: '45 ETH' },
    { name: 'CryptoPunks', count: 1, floor: '35 ETH' },
    { name: 'Azuki', count: 1, floor: '12 ETH' },
  ];

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">NFTs</h1>

      <div className="flex gap-2 mb-6">
        <button onClick={() => setActiveTab('collected')} className={`px-4 py-2 rounded-lg ${activeTab === 'collected' ? 'bg-amber-500 text-black' : theme === 'dark' ? 'bg-slate-800' : 'bg-gray-200'}`}>Collected</button>
        <button onClick={() => setActiveTab('collections')} className={`px-4 py-2 rounded-lg ${activeTab === 'collections' ? 'bg-amber-500 text-black' : theme === 'dark' ? 'bg-slate-800' : 'bg-gray-200'}`}>Collections</button>
        <button onClick={() => setActiveTab('activity')} className={`px-4 py-2 rounded-lg ${activeTab === 'activity' ? 'bg-amber-500 text-black' : theme === 'dark' ? 'bg-slate-800' : 'bg-gray-200'}`}>Activity</button>
      </div>

      {activeTab === 'collected' && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {nfts.map((nft) => (
            <div key={nft.id} className={`card ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
              <div className="w-full h-48 bg-gray-700 rounded-lg mb-4 flex items-center justify-center text-6xl">
                {nft.image}
              </div>
              <h3 className="font-bold">{nft.name}</h3>
              <p className="text-sm opacity-60">{nft.collection}</p>
              <p className="text-amber-500 mt-2">Floor: {nft.floor}</p>
            </div>
          ))}
        </div>
      )}

      {activeTab === 'collections' && (
        <div className="space-y-4">
          {collections.map((col, i) => (
            <div key={i} className={`card flex justify-between items-center ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
              <div><h3 className="font-bold">{col.name}</h3><p className="text-sm opacity-60">{col.count} items</p></div>
              <div className="text-right"><p className="font-bold text-amber-500">{col.floor}</p></div>
            </div>
          ))}
        </div>
      )}

      {activeTab === 'activity' && (
        <div className={`card ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
          <p className="text-center opacity-60">No recent activity</p>
        </div>
      )}
    </div>
  );
}

export default NFTsPage;
