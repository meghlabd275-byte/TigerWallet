'use client'

import { useState, useEffect } from 'react'
import { ThemeToggle } from '../components/ThemeToggle'
import { useTheme } from '../components/ThemeProvider'

interface RWAAsset {
  id: string
  name: string
  symbol: string
  type: 'STOCK' | 'ETF' | 'COMMODITY' | 'BOND' | 'CRYPTO'
  price: number
  change24h: number
  volume24h: number
}

interface Holding {
  asset_id: string
  symbol: string
  quantity: number
  value: number
}

interface Order {
  id: string
  type: 'BUY' | 'SELL'
  asset: string
  quantity: number
  price: number
  status: 'PENDING' | 'FILLED' | 'CANCELLED'
}

export default function RWATradingPage() {
  const { theme } = useTheme()
  const [assets, setAssets] = useState<RWAAsset[]>([])
  const [holdings, setHoldings] = useState<Holding[]>([])
  const [orders, setOrders] = useState<Order[]>([])
  const [balance, setBalance] = useState({ balance: 0, available: 0 })
  const [loading, setLoading] = useState(true)
  const [selectedAsset, setSelectedAsset] = useState<RWAAsset | null>(null)
  const [orderType, setOrderType] = useState<'BUY' | 'SELL'>('BUY')
  const [quantity, setQuantity] = useState('')

  useEffect(() => {
    fetchData()
  }, [])

  const fetchData = async () => {
    try {
      const [assetsRes, portfolioRes, balanceRes, ordersRes] = await Promise.all([
        fetch('/api/v1/rwa/assets'),
        fetch('/api/v1/rwa/portfolio'),
        fetch('/api/v1/rwa/balance'),
        fetch('/api/v1/rwa/orders')
      ])
      const assetsData = await assetsRes.json()
      const portfolioData = await portfolioRes.json()
      const balanceData = await balanceRes.json()
      const ordersData = await ordersRes.json()
      
      setAssets(assetsData.assets || [])
      setHoldings(portfolioData.holdings || [])
      setBalance(balanceData)
      setOrders(ordersData.orders || [])
    } catch (error) {
      console.error('Failed to fetch data:', error)
      // Do NOT fabricate RWA asset prices or balances on failure.
      setAssets([])
      setHoldings([])
      setBalance({ balance: 0, available: 0 })
      setOrders([])
    } finally {
      setLoading(false)
    }
  }

  const handlePlaceOrder = async () => {
    if (!selectedAsset || !quantity || parseFloat(quantity) <= 0) return

    try {
      await fetch('/api/v1/rwa/orders', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          asset_id: selectedAsset.id,
          type: orderType,
          quantity: parseFloat(quantity)
        })
      })
      fetchData()
      setSelectedAsset(null)
      setQuantity('')
    } catch (error) {
      console.error('Failed to place order:', error)
    }
  }

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD'
    }).format(value)
  }

  const isDark = theme === 'dark'

  const totalPortfolioValue = holdings.reduce((acc, h) => acc + h.value, 0) + balance.balance

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900 text-white' : 'bg-gray-50 text-gray-900'}`}>
      {/* Header */}
      <header className={`${isDark ? 'bg-gray-800' : 'bg-white'} shadow-sm border-b ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center space-x-3">
              <span className="text-2xl">📊</span>
              <h1 className="text-xl font-bold">RWA Trading</h1>
            </div>
            <ThemeToggle />
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {loading ? (
          <div className="flex justify-center items-center h-64">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
          </div>
        ) : (
          <>
            {/* Portfolio Overview */}
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
              <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg shadow p-6`}>
                <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Total Value</p>
                <p className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
                  {formatCurrency(totalPortfolioValue)}
                </p>
              </div>
              <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg shadow p-6`}>
                <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Available Cash</p>
                <p className={`text-2xl font-bold ${isDark ? 'text-green-400' : 'text-green-600'}`}>
                  {formatCurrency(balance.available)}
                </p>
              </div>
              <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg shadow p-6`}>
                <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Holdings</p>
                <p className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>{holdings.length}</p>
              </div>
              <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg shadow p-6`}>
                <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Pending Orders</p>
                <p className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
                  {orders.filter(o => o.status === 'PENDING').length}
                </p>
              </div>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
              {/* Assets List */}
              <div className="lg:col-span-2">
                <h2 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                  Available Assets
                </h2>
                <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg shadow overflow-hidden`}>
                  <table className="min-w-full divide-y divide-gray-700">
                    <thead className={isDark ? 'bg-gray-700' : 'bg-gray-50'}>
                      <tr>
                        <th className={`px-6 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-500'} uppercase tracking-wider`}>
                          Asset
                        </th>
                        <th className={`px-6 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-500'} uppercase tracking-wider`}>
                          Type
                        </th>
                        <th className={`px-6 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-500'} uppercase tracking-wider`}>
                          Price
                        </th>
                        <th className={`px-6 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-500'} uppercase tracking-wider`}>
                          24h Change
                        </th>
                        <th className={`px-6 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-500'} uppercase tracking-wider`}>
                          Action
                        </th>
                      </tr>
                    </thead>
                    <tbody className={`divide-y ${isDark ? 'divide-gray-700' : 'divide-gray-200'}`}>
                      {assets.map(asset => (
                        <tr key={asset.id} className="hover:bg-opacity-50">
                          <td className="px-6 py-4 whitespace-nowrap">
                            <div className="flex items-center">
                              <div className={`p-2 rounded-full ${isDark ? 'bg-gray-700' : 'bg-gray-100'} mr-3`}>
                                {asset.type === 'STOCK' && '📈'}
                                {asset.type === 'ETF' && '📊'}
                                {asset.type === 'COMMODITY' && '🥇'}
                                {asset.type === 'BOND' && '📜'}
                                {asset.type === 'CRYPTO' && '₿'}
                              </div>
                              <div>
                                <p className={`font-medium ${isDark ? 'text-white' : 'text-gray-900'}`}>{asset.name}</p>
                                <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{asset.symbol}</p>
                              </div>
                            </div>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap">
                            <span className={`px-2 py-1 text-xs rounded ${
                              asset.type === 'STOCK' ? 'bg-blue-500' :
                              asset.type === 'ETF' ? 'bg-purple-500' :
                              asset.type === 'COMMODITY' ? 'bg-yellow-500' :
                              'bg-gray-500'
                            } text-white`}>
                              {asset.type}
                            </span>
                          </td>
                          <td className={`px-6 py-4 whitespace-nowrap ${isDark ? 'text-white' : 'text-gray-900'}`}>
                            {formatCurrency(asset.price)}
                          </td>
                          <td className={`px-6 py-4 whitespace-nowrap ${asset.change24h >= 0 ? 'text-green-500' : 'text-red-500'}`}>
                            {asset.change24h >= 0 ? '+' : ''}{asset.change24h.toFixed(2)}%
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap">
                            <button
                              onClick={() => setSelectedAsset(asset)}
                              className="text-blue-500 hover:text-blue-600"
                            >
                              Trade
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>

              {/* Portfolio & Orders */}
              <div>
                {/* Holdings */}
                <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg shadow p-6 mb-6`}>
                  <h3 className={`font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                    Your Holdings
                  </h3>
                  {holdings.length === 0 ? (
                    <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>No holdings yet</p>
                  ) : (
                    <div className="space-y-3">
                      {holdings.map(holding => (
                        <div key={holding.asset_id} className="flex justify-between">
                          <div>
                            <p className={`font-medium ${isDark ? 'text-white' : 'text-gray-900'}`}>{holding.symbol}</p>
                            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{holding.quantity} units</p>
                          </div>
                          <p className={`font-medium ${isDark ? 'text-white' : 'text-gray-900'}`}>{formatCurrency(holding.value)}</p>
                        </div>
                      ))}
                    </div>
                  )}
                </div>

                {/* Recent Orders */}
                <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg shadow p-6`}>
                  <h3 className={`font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                    Recent Orders
                  </h3>
                  {orders.length === 0 ? (
                    <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>No orders yet</p>
                  ) : (
                    <div className="space-y-3">
                      {orders.slice(0, 5).map(order => (
                        <div key={order.id} className="flex justify-between items-center">
                          <div>
                            <span className={`text-sm font-medium ${order.type === 'BUY' ? 'text-green-500' : 'text-red-500'}`}>
                              {order.type}
                            </span>
                            <span className={`ml-2 ${isDark ? 'text-white' : 'text-gray-900'}`}>{order.asset}</span>
                          </div>
                          <span className={`px-2 py-1 text-xs rounded ${
                            order.status === 'FILLED' ? 'bg-green-500' :
                            order.status === 'PENDING' ? 'bg-yellow-500' :
                            'bg-red-500'
                          } text-white`}>
                            {order.status}
                          </span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            </div>
          </>
        )}
      </main>

      {/* Trade Modal */}
      {selectedAsset && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg p-6 max-w-md w-full mx-4`}>
            <h3 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
              Trade {selectedAsset.symbol}
            </h3>
            <div className="mb-4">
              <label className={`block text-sm font-medium mb-2 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>
                Order Type
              </label>
              <div className="flex gap-3">
                <button
                  onClick={() => setOrderType('BUY')}
                  className={`flex-1 p-3 rounded-lg border-2 ${
                    orderType === 'BUY'
                      ? 'border-green-500 bg-green-500/20'
                      : 'border-gray-600'
                  } transition-colors`}
                >
                  <p className="font-semibold text-green-500">BUY</p>
                </button>
                <button
                  onClick={() => setOrderType('SELL')}
                  className={`flex-1 p-3 rounded-lg border-2 ${
                    orderType === 'SELL'
                      ? 'border-red-500 bg-red-500/20'
                      : 'border-gray-600'
                  } transition-colors`}
                >
                  <p className="font-semibold text-red-500">SELL</p>
                </button>
              </div>
            </div>
            <div className="mb-4">
              <label className={`block text-sm font-medium mb-2 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>
                Quantity
              </label>
              <input
                type="number"
                value={quantity}
                onChange={(e) => setQuantity(e.target.value)}
                className={`w-full px-3 py-2 rounded-lg border ${
                  isDark 
                    ? 'bg-gray-700 border-gray-600 text-white' 
                    : 'bg-white border-gray-300 text-gray-900'
                } focus:outline-none focus:ring-2 focus:ring-blue-500`}
                placeholder="Enter quantity"
              />
            </div>
            {quantity && parseFloat(quantity) > 0 && (
              <div className={`mb-4 p-3 rounded-lg ${isDark ? 'bg-gray-700' : 'bg-gray-100'}`}>
                <div className="flex justify-between mb-2">
                  <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Price</p>
                  <p className={isDark ? 'text-white' : 'text-gray-900'}>{formatCurrency(selectedAsset.price)}</p>
                </div>
                <div className="flex justify-between">
                  <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Total</p>
                  <p className={`font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
                    {formatCurrency(parseFloat(quantity) * selectedAsset.price)}
                  </p>
                </div>
              </div>
            )}
            <div className="flex gap-3">
              <button
                onClick={handlePlaceOrder}
                className="flex-1 bg-blue-600 hover:bg-blue-700 text-white py-2 rounded-lg transition-colors"
              >
                Place Order
              </button>
              <button
                onClick={() => setSelectedAsset(null)}
                className={`flex-1 py-2 rounded-lg transition-colors ${
                  isDark ? 'bg-gray-700 hover:bg-gray-600 text-white' : 'bg-gray-200 hover:bg-gray-300 text-gray-900'
                }`}
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
