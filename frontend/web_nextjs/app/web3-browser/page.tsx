'use client';

import React, { useState } from 'react';

interface Bookmark {
  id: string;
  title: string;
  url: string;
  favicon: string;
}

const DEFAULT_BOOKMARKS: Bookmark[] = [
  { id: '1', title: 'Uniswap', url: 'https://app.uniswap.org', favicon: '🦄' },
  { id: '2', title: 'Aave', url: 'https://app.aave.com', favicon: '👻' },
  { id: '3', title: 'OpenSea', url: 'https://opensea.io', favicon: '🌊' },
];

export default function Web3BrowserPage() {
  const [url, setUrl] = useState('');
  const [tabs, setTabs] = useState([{ id: '1', url: '', title: 'New Tab', favicon: '' }]);
  const [activeTab, setActiveTab] = useState('1');
  const [bookmarks, setBookmarks] = useState<Bookmark[]>(DEFAULT_BOOKMARKS);
  const [showBookmarks, setShowBookmarks] = useState(false);

  const navigate = (newUrl: string) => {
    if (!newUrl) return;
    let formattedUrl = newUrl;
    if (!newUrl.startsWith('http://') && !newUrl.startsWith('https://')) {
      formattedUrl = 'https://' + newUrl;
    }
    setTabs(prev => prev.map(tab => 
      tab.id === activeTab ? { ...tab, url: formattedUrl, title: formattedUrl } : tab
    ));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    navigate(url);
  };

  const currentTab = tabs.find(t => t.id === activeTab);

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 to-slate-800">
      <div className="bg-slate-800 border-b border-slate-700">
        <div className="flex items-center gap-2 p-2">
          <button onClick={() => setShowBookmarks(!showBookmarks)} className="p-2 text-slate-400">⭐</button>
          <form onSubmit={handleSubmit} className="flex-1">
            <input
              type="text"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="Enter URL..."
              className="w-full bg-slate-700 border border-slate-600 rounded-lg px-4 py-2 text-white"
            />
          </form>
        </div>
      </div>
      <div className="h-[calc(100vh-100px)]">
        {currentTab?.url ? (
          <iframe src={currentTab.url} className="w-full h-full bg-white" title="Browser" />
        ) : (
          <div className="flex flex-col items-center justify-center h-full p-8">
            <h1 className="text-3xl font-bold text-white mb-4">Web3 Browser</h1>
            <div className="grid grid-cols-3 gap-4">
              {bookmarks.map(bookmark => (
                <div key={bookmark.id} onClick={() => navigate(bookmark.url)} 
                  className="flex items-center gap-3 p-4 bg-slate-700/50 rounded-xl cursor-pointer">
                  <span className="text-2xl">{bookmark.favicon}</span>
                  <span className="text-white">{bookmark.title}</span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
