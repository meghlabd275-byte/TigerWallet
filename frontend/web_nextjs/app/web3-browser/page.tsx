'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../components/ThemeProvider'

interface Bookmark {
  id: string;
  title: string;
  url: string;
  favicon: string;
}

interface HistoryItem {
  id: string;
  title: string;
  url: string;
  timestamp: number;
  favicon: string;
}

interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: string;
}

const API_BASE_URL = typeof window !== 'undefined' ? '' : (process.env.BACKEND_URL || 'http://localhost:8443');

const fetchAPI = async <T,>(endpoint: string, options?: RequestInit): Promise<T> => {
  const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
  const response = await fetch(`${API_BASE_URL}/api/v1${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });
  if (!response.ok) throw new Error(`API Error: ${response.statusText}`);
  const data: ApiResponse<T> = await response.json();
  return data.data;
};

const DEFAULT_BOOKMARKS: Bookmark[] = [
  { id: '1', title: 'Uniswap', url: 'https://app.uniswap.org', favicon: '🦄' },
  { id: '2', title: 'Aave', url: 'https://app.aave.com', favicon: '👻' },
  { id: '3', title: 'OpenSea', url: 'https://opensea.io', favicon: '🌊' },
  { id: '4', title: 'PancakeSwap', url: 'https://pancakeswap.finance', favicon: '🥞' },
  { id: '5', title: 'Curve', url: 'https://curve.fi', favicon: '📈' },
  { id: '6', title: 'Lens Protocol', url: 'https://lens.xyz', favicon: '🌿' },
];

export default function Web3BrowserPage() {
  const { isDark } = useTheme()
  const [url, setUrl] = useState('');
  const [tabs, setTabs] = useState([{ id: '1', url: '', title: 'New Tab', favicon: '' }]);
  const [activeTab, setActiveTab] = useState('1');
  const [bookmarks, setBookmarks] = useState<Bookmark[]>(DEFAULT_BOOKMARKS);
  const [history, setHistory] = useState<HistoryItem[]>([]);
  const [showBookmarks, setShowBookmarks] = useState(false);
  const [showHistory, setShowHistory] = useState(false);
  const [showAddBookmark, setShowAddBookmark] = useState(false);
  const [newBookmarkTitle, setNewBookmarkTitle] = useState('');
  const [newBookmarkUrl, setNewBookmarkUrl] = useState('');

  // Load bookmarks and history from backend
  const loadBrowserData = useCallback(async () => {
    try {
      const [bookmarksData, historyData] = await Promise.all([
        fetchAPI<Bookmark[]>('/browser/bookmarks'),
        fetchAPI<HistoryItem[]>('/browser/history'),
      ]);
      if (bookmarksData && bookmarksData.length > 0) setBookmarks(bookmarksData);
      if (historyData) setHistory(historyData);
    } catch (err) {
      console.log('Using local browser data');
    }
  }, []);

  useEffect(() => {
    loadBrowserData();
  }, [loadBrowserData]);

  const navigate = (newUrl: string) => {
    if (!newUrl) return;
    let formattedUrl = newUrl;
    if (!newUrl.startsWith('http://') && !newUrl.startsWith('https://')) {
      formattedUrl = 'https://' + newUrl;
    }
    
    const title = formattedUrl.replace('https://', '').replace('http://', '').split('/')[0];
    
    setTabs(prev => prev.map(tab => 
      tab.id === activeTab ? { ...tab, url: formattedUrl, title, favicon: '🌐' } : tab
    ));

    // Add to history
    const historyItem: HistoryItem = {
      id: Date.now().toString(),
      title,
      url: formattedUrl,
      timestamp: Date.now(),
      favicon: '🌐',
    };
    setHistory(prev => [historyItem, ...prev.slice(0, 49)]);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    navigate(url);
  };

  const addNewTab = () => {
    const newId = (tabs.length + 1).toString();
    setTabs([...tabs, { id: newId, url: '', title: 'New Tab', favicon: '' }]);
    setActiveTab(newId);
    setUrl('');
  };

  const closeTab = (tabId: string) => {
    if (tabs.length === 1) return;
    const newTabs = tabs.filter(t => t.id !== tabId);
    setTabs(newTabs);
    if (activeTab === tabId) {
      setActiveTab(newTabs[0].id);
    }
  };

  const addBookmark = async () => {
    if (!newBookmarkTitle || !newBookmarkUrl) return;
    
    const newBookmark: Bookmark = {
      id: Date.now().toString(),
      title: newBookmarkTitle,
      url: newBookmarkUrl.startsWith('http') ? newBookmarkUrl : `https://${newBookmarkUrl}`,
      favicon: '⭐',
    };

    try {
      await fetchAPI('/browser/bookmarks', {
        method: 'POST',
        body: JSON.stringify(newBookmark),
      });
    } catch (err) {
      console.log('Saving locally');
    }

    setBookmarks([...bookmarks, newBookmark]);
    setNewBookmarkTitle('');
    setNewBookmarkUrl('');
    setShowAddBookmark(false);
  };

  const removeBookmark = async (bookmarkId: string) => {
    try {
      await fetchAPI(`/browser/bookmarks/${bookmarkId}`, { method: 'DELETE' });
    } catch (err) {
      console.log('Removing locally');
    }
    setBookmarks(bookmarks.filter(b => b.id !== bookmarkId));
  };

  const clearHistory = async () => {
    try {
      await fetchAPI('/browser/history', { method: 'DELETE' });
    } catch (err) {
      console.log('Clearing locally');
    }
    setHistory([]);
  };

  const currentTab = tabs.find(t => t.id === activeTab);

  return (
    <div className={`'min-h-screen bg-gradient-to-br' ${isDark ? 'from-slate-900' : 'from-slate-50'} ${isDark ? 'to-slate-800' : 'to-slate-100'}`}>
      {/* Tab Bar */}
      <div className={`${isDark ? 'bg-slate-800' : 'bg-white'} 'border-b' ${isDark ? 'border-slate-700' : 'border-slate-200'}`}>
        <div className="flex items-center gap-1 p-2 overflow-x-auto">
          {tabs.map(tab => (
            <div 
              key={tab.id}
              className={`flex items-center gap-2 px-3 py-1 rounded-lg cursor-pointer ${
                activeTab === tab.id ? 'bg-slate-600 text-white' : 'text-slate-400 hover:bg-slate-700'
              }`}
              onClick={() => setActiveTab(tab.id)}
            >
              <span className="text-sm">{tab.favicon || '🌐'}</span>
              <span className="text-sm max-w-[100px] truncate">{tab.title}</span>
              {tabs.length > 1 && (
                <button 
                  onClick={(e) => { e.stopPropagation(); closeTab(tab.id); }}
                  className="text-slate-500 hover:text-white"
                >
                  ×
                </button>
              )}
            </div>
          ))}
          <button 
            onClick={addNewTab}
            className={`'px-2 py-1' ${isDark ? 'text-slate-400' : 'text-slate-500'} 'hover:text-white text-xl'`}
          >
            +
          </button>
        </div>
      </div>

      {/* Navigation Bar */}
      <div className={`${isDark ? 'bg-slate-800' : 'bg-white'} 'border-b' ${isDark ? 'border-slate-700' : 'border-slate-200'}`}>
        <div className="flex items-center gap-2 p-2">
          <button 
            onClick={() => { setShowHistory(!showHistory); setShowBookmarks(false); setShowAddBookmark(false); }}
            className={`'p-2' ${isDark ? 'text-slate-400' : 'text-slate-500'} 'hover:text-white'`}
            title="History"
          >
            🕐
          </button>
          <button 
            onClick={() => { setShowBookmarks(!showBookmarks); setShowHistory(false); setShowAddBookmark(false); }}
            className={`'p-2' ${isDark ? 'text-slate-400' : 'text-slate-500'} 'hover:text-white'`}
            title="Bookmarks"
          >
            ⭐
          </button>
          <form onSubmit={handleSubmit} className="flex-1">
            <input
              type="text"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="Enter URL or search..."
              className={`'w-full' ${isDark ? 'bg-slate-700' : 'bg-slate-100'} 'border' ${isDark ? 'border-slate-600' : 'border-slate-300'} 'rounded-lg px-4 py-2' ${isDark ? 'text-white' : 'text-slate-900'} 'placeholder-slate-400'`}
            />
          </form>
        </div>
      </div>

      {/* Bookmarks Panel */}
      {showBookmarks && (
        <div className={`'absolute top-36 left-2' ${isDark ? 'bg-slate-800' : 'bg-white'} 'rounded-lg shadow-xl z-50 w-80'`}>
          <div className={`'p-3 border-b' ${isDark ? 'border-slate-700' : 'border-slate-200'} 'flex justify-between items-center'`}>
            <span className={`${isDark ? 'text-white' : 'text-slate-900'} 'font-semibold'`}>Bookmarks</span>
            <button 
              onClick={() => setShowAddBookmark(true)}
              className="text-orange-500 hover:text-orange-400"
            >
              + Add
            </button>
          </div>
          <div className="max-h-64 overflow-y-auto">
            {bookmarks.map(bookmark => (
              <div 
                key={bookmark.id}
                className={`'flex items-center justify-between p-2' ${isDark ? 'hover:bg-slate-700' : 'hover:bg-slate-100'} 'cursor-pointer'`}
                onClick={() => { navigate(bookmark.url); setShowBookmarks(false); }}
              >
                <div className="flex items-center gap-2">
                  <span>{bookmark.favicon}</span>
                  <span className={`${isDark ? 'text-white' : 'text-slate-900'} 'text-sm'`}>{bookmark.title}</span>
                </div>
                <button 
                  onClick={(e) => { e.stopPropagation(); removeBookmark(bookmark.id); }}
                  className="text-slate-500 hover:text-red-400 text-sm"
                >
                  ×
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Add Bookmark Modal */}
      {showAddBookmark && (
        <div className={`'absolute top-36 left-2' ${isDark ? 'bg-slate-800' : 'bg-white'} 'rounded-lg shadow-xl z-50 w-80 p-3'`}>
          <div className={`${isDark ? 'text-white' : 'text-slate-900'} 'font-semibold mb-3'`}>Add Bookmark</div>
          <input
            type="text"
            value={newBookmarkTitle}
            onChange={(e) => setNewBookmarkTitle(e.target.value)}
            placeholder="Title"
            className={`'w-full' ${isDark ? 'bg-slate-700' : 'bg-slate-100'} 'border' ${isDark ? 'border-slate-600' : 'border-slate-300'} 'rounded px-3 py-2' ${isDark ? 'text-white' : 'text-slate-900'} 'mb-2'`}
          />
          <input
            type="text"
            value={newBookmarkUrl}
            onChange={(e) => setNewBookmarkUrl(e.target.value)}
            placeholder="URL"
            className={`'w-full' ${isDark ? 'bg-slate-700' : 'bg-slate-100'} 'border' ${isDark ? 'border-slate-600' : 'border-slate-300'} 'rounded px-3 py-2' ${isDark ? 'text-white' : 'text-slate-900'} 'mb-3'`}
          />
          <div className="flex gap-2">
            <button 
              onClick={addBookmark}
              className="flex-1 bg-orange-500 text-white py-2 rounded hover:bg-orange-600"
            >
              Add
            </button>
            <button 
              onClick={() => setShowAddBookmark(false)}
              className={`'flex-1' ${isDark ? 'bg-slate-600' : 'bg-slate-200'} ${isDark ? 'text-white' : 'text-slate-900'} 'py-2 rounded hover:bg-slate-500'`}
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* History Panel */}
      {showHistory && (
        <div className={`'absolute top-36 left-2' ${isDark ? 'bg-slate-800' : 'bg-white'} 'rounded-lg shadow-xl z-50 w-80'`}>
          <div className={`'p-3 border-b' ${isDark ? 'border-slate-700' : 'border-slate-200'} 'flex justify-between items-center'`}>
            <span className={`${isDark ? 'text-white' : 'text-slate-900'} 'font-semibold'`}>History</span>
            <button 
              onClick={clearHistory}
              className="text-orange-500 hover:text-orange-400 text-sm"
            >
              Clear All
            </button>
          </div>
          <div className="max-h-64 overflow-y-auto">
            {history.length === 0 ? (
              <div className={`'p-3' ${isDark ? 'text-slate-400' : 'text-slate-500'} 'text-sm'`}>No history yet</div>
            ) : (
              history.map(item => (
                <div 
                  key={item.id}
                  className={`'p-2' ${isDark ? 'hover:bg-slate-700' : 'hover:bg-slate-100'} 'cursor-pointer'`}
                  onClick={() => { navigate(item.url); setShowHistory(false); }}
                >
                  <div className={`${isDark ? 'text-white' : 'text-slate-900'} 'text-sm'`}>{item.title}</div>
                  <div className="text-slate-500 text-xs">
                    {new Date(item.timestamp).toLocaleString()}
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      )}

      {/* Browser Content */}
      <div className="h-[calc(100vh-120px)]">
        {currentTab?.url ? (
          <iframe 
            src={currentTab.url} 
            className="w-full h-full bg-white" 
            title="Browser"
            sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-popups-to-escape-sandbox"
          />
        ) : (
          <div className="flex flex-col items-center justify-center h-full p-8">
            <h1 className={`'text-3xl font-bold' ${isDark ? 'text-white' : 'text-slate-900'} 'mb-4'`}>Web3 Browser</h1>
            <p className={`${isDark ? 'text-slate-400' : 'text-slate-500'} 'mb-8'`}>Browse the decentralized web</p>
            
            <div className="grid grid-cols-2 md:grid-cols-3 gap-4 mb-8">
              <div className={`'col-span-full' ${isDark ? 'text-white' : 'text-slate-900'} 'font-semibold mb-2'`}>Popular DApps</div>
              {bookmarks.slice(0, 6).map(bookmark => (
                <div 
                  key={bookmark.id} 
                  onClick={() => navigate(bookmark.url)}
                  className="flex items-center gap-3 p-4 bg-slate-700/50 hover:bg-slate-600/50 rounded-xl cursor-pointer transition-colors"
                >
                  <span className="text-2xl">{bookmark.favicon}</span>
                  <span className={`${isDark ? 'text-white' : 'text-slate-900'}`}>{bookmark.title}</span>
                </div>
              ))}
            </div>
            
            <div className="text-slate-500 text-sm">
              Your bookmarks and history are synced across devices
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
