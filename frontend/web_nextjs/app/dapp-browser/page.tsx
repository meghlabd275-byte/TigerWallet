'use client'

import { useState, useEffect, useRef } from 'react'
import { ThemeToggle } from '../components/ThemeToggle'
import { useTheme } from '../components/ThemeProvider'

interface DApp {
  id: string
  name: string
  url: string
  category: string
  logo: string
  description: string
}

interface Bookmark {
  id: string
  name: string
  url: string
  logo: string
}

interface Connection {
  id: string
  name: string
  url: string
  connected: boolean
  lastUsed: number
}

export default function DAppBrowserPage() {
  const { theme } = useTheme()
  const [url, setUrl] = useState('')
  const [currentUrl, setCurrentUrl] = useState('')
  const [iframeRef, setIframeRef] = useState<HTMLIFrameElement | null>(null)
  const [connected, setConnected] = useState(false)
  const [bookmarks, setBookmarks] = useState<Bookmark[]>([])
  const [showBookmarks, setShowBookmarks] = useState(false)
  const [showConnections, setShowConnections] = useState(false)
  const [connections, setConnections] = useState<Connection[]>([])
  const [activeTab, setActiveTab] = useState<'discover' | 'bookmarks' | 'history'>('discover')
  const [searchQuery, setSearchQuery] = useState('')
  const [popularDApps, setPopularDApps] = useState<DApp[]>([])
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    fetch('/api/v1/dapps', { cache: 'no-store' })
      .then(res => res.ok ? res.json() : Promise.reject())
      .then(data => setPopularDApps((data.dapps || []).map((d: { id: string; name: string; url: string; category: string; logo: string; description: string }) => ({
        id: d.id, name: d.name, url: d.url, category: d.category, logo: d.logo, description: d.description,
      }))))
      .catch(() => setPopularDApps([]))
  }, [])

  useEffect(() => {
    const savedBookmarks = localStorage.getItem('dapp_bookmarks')
    if (savedBookmarks) {
      setBookmarks(JSON.parse(savedBookmarks))
    }
    const savedConnections = localStorage.getItem('dapp_connections')
    if (savedConnections) {
      setConnections(JSON.parse(savedConnections))
    }
  }, [])

  const handleNavigate = (navigateUrl?: string) => {
    const targetUrl = navigateUrl || url
    if (!targetUrl) return
    
    let formattedUrl = targetUrl
    if (!targetUrl.startsWith('http://') && !targetUrl.startsWith('https://')) {
      formattedUrl = 'https://' + targetUrl
    }
    setCurrentUrl(formattedUrl)
    setUrl(formattedUrl)
  }

  const handleBack = () => {
    if (iframeRef?.contentWindow) {
      iframeRef.contentWindow.history.back()
    }
  }

  const handleForward = () => {
    if (iframeRef?.contentWindow) {
      iframeRef.contentWindow.history.forward()
    }
  }

  const handleRefresh = () => {
    if (iframeRef) {
      iframeRef.src = iframeRef.src
    }
  }

  const addBookmark = () => {
    const newBookmark: Bookmark = {
      id: Date.now().toString(),
      name: currentUrl.split('//')[1]?.split('/')[0] || 'DApp',
      url: currentUrl,
      logo: '🔗'
    }
    const updatedBookmarks = [...bookmarks, newBookmark]
    setBookmarks(updatedBookmarks)
    localStorage.setItem('dapp_bookmarks', JSON.stringify(updatedBookmarks))
  }

  const removeBookmark = (id: string) => {
    const updatedBookmarks = bookmarks.filter(b => b.id !== id)
    setBookmarks(updatedBookmarks)
    localStorage.setItem('dapp_bookmarks', JSON.stringify(updatedBookmarks))
  }

  const filteredDApps = popularDApps.filter(dapp => 
    dapp.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    dapp.category.toLowerCase().includes(searchQuery.toLowerCase())
  )

  const disconnectDApp = (id: string) => {
    const updatedConnections = connections.map(c => 
      c.id === id ? { ...c, connected: false } : c
    )
    setConnections(updatedConnections)
    localStorage.setItem('dapp_connections', JSON.stringify(updatedConnections))
  }

  const isDark = theme === 'dark'

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900 text-white' : 'bg-gray-50 text-gray-900'}`}>
      {/* Header */}
      <header className={`${isDark ? 'bg-gray-800' : 'bg-white'} shadow-sm border-b ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
        <div className="max-w-full mx-auto px-4 py-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <span className="text-2xl">🌐</span>
              <h1 className="text-xl font-bold hidden md:block">DApp Browser</h1>
            </div>
            
            {/* Navigation Bar */}
            <div className="flex-1 max-w-3xl mx-4">
              <div className={`flex items-center ${isDark ? 'bg-gray-700' : 'bg-gray-100'} rounded-lg px-2`}>
                <button
                  onClick={handleBack}
                  className={`p-2 ${isDark ? 'hover:bg-gray-600' : 'hover:bg-gray-200'} rounded`}
                  title="Back"
                >
                  ←
                </button>
                <button
                  onClick={handleForward}
                  className={`p-2 ${isDark ? 'hover:bg-gray-600' : 'hover:bg-gray-200'} rounded`}
                  title="Forward"
                >
                  →
                </button>
                <button
                  onClick={handleRefresh}
                  className={`p-2 ${isDark ? 'hover:bg-gray-600' : 'hover:bg-gray-200'} rounded`}
                  title="Refresh"
                >
                  ↻
                </button>
                <input
                  ref={inputRef}
                  type="text"
                  value={url}
                  onChange={(e) => setUrl(e.target.value)}
                  onKeyPress={(e) => e.key === 'Enter' && handleNavigate()}
                  placeholder="Enter URL or search..."
                  className={`flex-1 px-3 py-2 bg-transparent outline-none ${
                    isDark ? 'text-white placeholder-gray-400' : 'text-gray-900 placeholder-gray-500'
                  }`}
                />
                <button
                  onClick={addBookmark}
                  className={`p-2 ${isDark ? 'hover:bg-gray-600' : 'hover:bg-gray-200'} rounded`}
                  title="Add Bookmark"
                >
                  ⭐
                </button>
              </div>
            </div>

            <div className="flex items-center space-x-2">
              <button
                onClick={() => setShowConnections(!showConnections)}
                className={`px-3 py-1.5 rounded-lg text-sm ${
                  connected 
                    ? 'bg-green-500 text-white' 
                    : isDark ? 'bg-gray-700 hover:bg-gray-600' : 'bg-gray-200 hover:bg-gray-300'
                }`}
              >
                {connected ? 'Connected' : 'Connect'}
              </button>
              <ThemeToggle />
            </div>
          </div>
        </div>
      </header>

      <div className="flex h-[calc(100vh-70px)]">
        {/* Sidebar */}
        <div className={`w-64 ${isDark ? 'bg-gray-800' : 'bg-white'} border-r ${isDark ? 'border-gray-700' : 'border-gray-200'} flex flex-col`}>
          {/* Tabs */}
          <div className={`flex border-b ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
            <button
              onClick={() => setActiveTab('discover')}
              className={`flex-1 py-3 text-sm font-medium ${
                activeTab === 'discover' 
                  ? isDark ? 'text-blue-400 border-b-2 border-blue-400' : 'text-blue-600 border-b-2 border-blue-600'
                  : isDark ? 'text-gray-400' : 'text-gray-500'
              }`}
            >
              Discover
            </button>
            <button
              onClick={() => setActiveTab('bookmarks')}
              className={`flex-1 py-3 text-sm font-medium ${
                activeTab === 'bookmarks' 
                  ? isDark ? 'text-blue-400 border-b-2 border-blue-400' : 'text-blue-600 border-b-2 border-blue-600'
                  : isDark ? 'text-gray-400' : 'text-gray-500'
              }`}
            >
              Bookmarks
            </button>
          </div>

          {/* Content */}
          <div className="flex-1 overflow-y-auto p-4">
            {activeTab === 'discover' && (
              <>
                <div className="mb-4">
                  <input
                    type="text"
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    placeholder="Search DApps..."
                    className={`w-full px-3 py-2 rounded-lg border ${
                      isDark 
                        ? 'bg-gray-700 border-gray-600 text-white' 
                        : 'bg-white border-gray-300'
                    }`}
                  />
                </div>
                <div className="space-y-2">
                  {filteredDApps.map(dapp => (
                    <button
                      key={dapp.id}
                      onClick={() => handleNavigate(dapp.url)}
                      className={`w-full p-3 rounded-lg text-left ${
                        isDark ? 'hover:bg-gray-700' : 'hover:bg-gray-100'
                      }`}
                    >
                      <div className="flex items-center space-x-3">
                        <span className="text-2xl">{dapp.logo}</span>
                        <div>
                          <p className="font-medium">{dapp.name}</p>
                          <p className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                            {dapp.category}
                          </p>
                        </div>
                      </div>
                    </button>
                  ))}
                </div>
              </>
            )}

            {activeTab === 'bookmarks' && (
              <div className="space-y-2">
                {bookmarks.length === 0 ? (
                  <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                    No bookmarks yet. Navigate to a DApp and click ⭐ to save it.
                  </p>
                ) : (
                  bookmarks.map(bookmark => (
                    <div
                      key={bookmark.id}
                      className={`flex items-center justify-between p-3 rounded-lg ${
                        isDark ? 'hover:bg-gray-700' : 'hover:bg-gray-100'
                      }`}
                    >
                      <button
                        onClick={() => handleNavigate(bookmark.url)}
                        className="flex items-center space-x-3 flex-1"
                      >
                        <span className="text-xl">{bookmark.logo}</span>
                        <span className="font-medium truncate">{bookmark.name}</span>
                      </button>
                      <button
                        onClick={() => removeBookmark(bookmark.id)}
                        className="text-red-500 hover:text-red-600 px-2"
                      >
                        ×
                      </button>
                    </div>
                  ))
                )}
              </div>
            )}
          </div>

          {/* Connection Status */}
          {connections.some(c => c.connected) && (
            <div className={`p-4 border-t ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
              <h3 className={`text-sm font-semibold mb-2 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                Connected DApps
              </h3>
              <div className="space-y-1">
                {connections.filter(c => c.connected).map(c => (
                  <div
                    key={c.id}
                    className={`flex items-center justify-between p-2 rounded ${
                      isDark ? 'bg-gray-700' : 'bg-gray-100'
                    }`}
                  >
                    <span className="text-sm truncate">{c.name}</span>
                    <button
                      onClick={() => disconnectDApp(c.id)}
                      className="text-xs text-red-500 hover:text-red-600"
                    >
                      Disconnect
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Browser Frame */}
        <div className="flex-1">
          {currentUrl ? (
            <iframe
              ref={(ref) => setIframeRef(ref)}
              src={currentUrl}
              className="w-full h-full border-0"
              sandbox="allow-scripts allow-same-origin allow-popups allow-forms allow-top-navigation"
              title="DApp Browser"
            />
          ) : (
            <div className={`h-full flex flex-col items-center justify-center ${isDark ? 'bg-gray-900' : 'bg-gray-50'}`}>
              <div className="text-6xl mb-4">🌐</div>
              <h2 className={`text-xl font-semibold mb-2 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                Welcome to TigerWallet DApp Browser
              </h2>
              <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                Browse decentralized applications securely
              </p>
              <div className="mt-8 grid grid-cols-2 md:grid-cols-4 gap-4">
                {popularDApps.slice(0, 4).map(dapp => (
                  <button
                    key={dapp.id}
                    onClick={() => handleNavigate(dapp.url)}
                    className={`p-4 rounded-lg ${isDark ? 'bg-gray-800 hover:bg-gray-700' : 'bg-white hover:bg-gray-100'} shadow`}
                  >
                    <span className="text-3xl block mb-2">{dapp.logo}</span>
                    <span className="font-medium">{dapp.name}</span>
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Connections Modal */}
      {showConnections && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg p-6 max-w-md w-full mx-4`}>
            <h3 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
              Connected DApps
            </h3>
            {connections.length === 0 ? (
              <p className={`${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                No DApps connected yet. Visit a DApp to connect.
              </p>
            ) : (
              <div className="space-y-2">
                {connections.map(c => (
                  <div
                    key={c.id}
                    className={`flex items-center justify-between p-3 rounded-lg ${
                      isDark ? 'bg-gray-700' : 'bg-gray-100'
                    }`}
                  >
                    <div>
                      <p className="font-medium">{c.name}</p>
                      <p className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{c.url}</p>
                    </div>
                    <button
                      onClick={() => disconnectDApp(c.id)}
                      className="text-red-500 hover:text-red-600 text-sm"
                    >
                      Disconnect
                    </button>
                  </div>
                ))}
              </div>
            )}
            <button
              onClick={() => setShowConnections(false)}
              className={`mt-4 w-full py-2 ${isDark ? 'bg-gray-700 hover:bg-gray-600' : 'bg-gray-200 hover:bg-gray-300'} rounded-lg`}
            >
              Close
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
