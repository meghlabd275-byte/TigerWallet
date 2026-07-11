'use client';

import React, { useState } from 'react';

interface NFT {
  id: string;
  name: string;
  collection: string;
  price: string;
  image: string;
  chain: string;
  seller: string;
}

const MOCK_NFTS: NFT[] = [
  { id: '1', name: 'Bored Ape #7854', collection: 'Bored Ape YC', price: '45.5 ETH', image: '🦧', chain: 'Ethereum', seller: '0x742d...E1E' },
  { id: '2', name: 'CryptoPunk #3456', collection: 'CryptoPunks', price: '32 ETH', image: '👽', chain: 'Ethereum', seller: '0xabcd...1234' },
  { id: '3', name: 'Azuki #7821', collection: 'Azuki', price: '18.5 ETH', image: '🥷', chain: 'Ethereum', seller: '0xdef1...5678' },
  { id: '4', name: 'DeGod #999', collection: 'DeGods', price: '420 SOL', image: '👻', chain: 'Solana', seller: 'G3d...xyz' },
  { id: '5', name: 'MadLads #123', collection: 'MadLads', price: '380 SOL', image: '🎭', chain: 'Solana', seller: 'A7x...123' },
  { id: '6', name: 'Milady #4567', collection: 'Milady', price: '4.2 ETH', image: '💄', chain: 'Ethereum', seller: '0x9876...abcd' },
  { id: '7', name: 'Pudgy #8821', collection: 'Pudgy Penguins', price: '3.8 ETH', image: '🐧', chain: 'Ethereum', seller: '0x2468...efgh' },
  { id: '8', name: 'StepN #234', collection: 'StepN', price: '8.5 SOL', image: '👟', chain: 'Solana', seller: 'B2k...789' },
];

export default function NFTMarketplace() {
  const [nfts] = useState<NFT[]>(MOCK_NFTS);
  const [search, setSearch] = useState('');
  const [filter, setFilter] = useState('all');
  
  const filteredNFTs = nfts.filter(nft => {
    const matchesSearch = nft.name.toLowerCase().includes(search.toLowerCase()) || nft.collection.toLowerCase().includes(search.toLowerCase());
    if (filter === 'all') return matchesSearch;
    return matchesSearch && nft.chain.toLowerCase() === filter;
  });

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-white">
      <header className="bg-white dark:bg-slate-800 border-b p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4"><a href="/" className="text-2xl">🐯</a><h1 className="text-xl font-bold">NFT Marketplace</h1></div>
          <a href="/wallet" className="text-orange-500">My NFTs</a>
        </div>
      </header>
      <div className="max-w-7xl mx-auto p-8">
        <div className="flex flex-wrap gap-4 mb-6">
          <input type="text" placeholder="Search NFTs..." value={search} onChange={(e) => setSearch(e.target.value)} className="flex-1 min-w-[200px] bg-white dark:bg-slate-800 border rounded-lg px-4 py-2" />
          {['all', 'Ethereum', 'Solana'].map(chain => <button key={chain} onClick={() => setFilter(chain)} className={`px-4 py-2 rounded-lg ${filter === chain ? 'bg-orange-500 text-white' : 'bg-slate-200 dark:bg-slate-700'}`}>{chain}</button>)}
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
          {filteredNFTs.map(nft => (
            <div key={nft.id} className="bg-white dark:bg-slate-800 rounded-lg overflow-hidden shadow-sm hover:shadow-lg transition">
              <div className="h-48 bg-gradient-to-br from-orange-400 to-pink-500 flex items-center justify-center text-8xl">{nft.image}</div>
              <div className="p-4">
                <div className="text-xs text-slate-500">{nft.collection}</div>
                <div className="font-semibold mb-2">{nft.name}</div>
                <div className="flex justify-between items-center">
                  <span className="text-lg font-bold text-orange-500">{nft.price}</span>
                  <span className="text-xs bg-slate-200 dark:bg-slate-700 px-2 py-1 rounded">{nft.chain}</span>
                </div>
                <button className="w-full mt-3 bg-orange-500 hover:bg-orange-600 text-white py-2 rounded-lg">Buy Now</button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
