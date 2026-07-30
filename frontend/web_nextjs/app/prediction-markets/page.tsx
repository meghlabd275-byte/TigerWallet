'use client'

import { useState, useEffect } from 'react'
import { ThemeToggle } from '../components/ThemeToggle'
import { useTheme } from '../components/ThemeProvider'

interface Market {
  id: string
  question: string
  yes_price: number
  no_price: number
  volume: number
  end_time: number
}

interface Bet {
  id: string
  market_id: string
  outcome: string
  amount: number
  payout: number
}

export default function PredictionMarketsPage() {
  const { theme } = useTheme()
  const [markets, setMarkets] = useState<Market[]>([])
  const [bets, setBets] = useState<Bet[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedMarket, setSelectedMarket] = useState<Market | null>(null)
  const [betAmount, setBetAmount] = useState('')
  const [betOutcome, setBetOutcome] = useState<'YES' | 'NO'>('YES')

  useEffect(() => {
    fetchData()
  }, [])

  const fetchData = async () => {
    try {
      const [marketsRes, betsRes] = await Promise.all([
        fetch('/api/v1/prediction/markets'),
        fetch('/api/v1/prediction/bets')
      ])
      const marketsData = await marketsRes.json()
      const betsData = await betsRes.json()
      setMarkets(marketsData.markets || [])
      setBets(betsData.bets || [])
    } catch (error) {
      console.error('Failed to fetch data:', error)
      // Fallback data
      setMarkets([
        { id: 'pm1', question: 'Will ETH reach $5000 by Dec 2026?', yes_price: 0.45, no_price: 0.55, volume: 500000, end_time: 1767225600 },
        { id: 'pm2', question: 'Will BTC reach $100k by June 2026?', yes_price: 0.60, no_price: 0.40, volume: 1200000, end_time: 1759300800 }
      ])
    } finally {
      setLoading(false)
    }
  }

  const handlePlaceBet = async () => {
    if (!selectedMarket || !betAmount || parseFloat(betAmount) <= 0) return

    try {
      await fetch('/api/v1/prediction/bets', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          market_id: selectedMarket.id,
          outcome: betOutcome,
          amount: parseFloat(betAmount)
        })
      })
      fetchData()
      setSelectedMarket(null)
      setBetAmount('')
    } catch (error) {
      console.error('Failed to place bet:', error)
    }
  }

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD'
    }).format(value)
  }

  const formatDate = (timestamp: number) => {
    return new Date(timestamp * 1000).toLocaleDateString()
  }

  const isDark = theme === 'dark'

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900 text-white' : 'bg-gray-50 text-gray-900'}`}>
      {/* Header */}
      <header className={`${isDark ? 'bg-gray-800' : 'bg-white'} shadow-sm border-b ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center space-x-3">
              <span className="text-2xl">🎯</span>
              <h1 className="text-xl font-bold">Prediction Markets</h1>
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
            {/* Stats Overview */}
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
              <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg shadow p-6`}>
                <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Total Markets</p>
                <p className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>{markets.length}</p>
              </div>
              <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg shadow p-6`}>
                <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Your Bets</p>
                <p className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>{bets.length}</p>
              </div>
              <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg shadow p-6`}>
                <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Total Volume</p>
                <p className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
                  {formatCurrency(markets.reduce((acc, m) => acc + m.volume, 0))}
                </p>
              </div>
              <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg shadow p-6`}>
                <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Potential Payout</p>
                <p className={`text-2xl font-bold ${isDark ? 'text-green-400' : 'text-green-600'}`}>
                  {formatCurrency(bets.reduce((acc, b) => acc + b.payout, 0))}
                </p>
              </div>
            </div>

            {/* Markets */}
            <div className="mb-8">
              <h2 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                Active Markets
              </h2>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {markets.map(market => (
                  <div
                    key={market.id}
                    className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg shadow p-6`}
                  >
                    <h3 className={`font-semibold mb-3 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                      {market.question}
                    </h3>
                    <div className="flex justify-between items-center mb-4">
                      <div>
                        <p className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Ends</p>
                        <p className={isDark ? 'text-white' : 'text-gray-900'}>{formatDate(market.end_time)}</p>
                      </div>
                      <div>
                        <p className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Volume</p>
                        <p className={isDark ? 'text-white' : 'text-gray-900'}>{formatCurrency(market.volume)}</p>
                      </div>
                    </div>
                    <div className="grid grid-cols-2 gap-3">
                      <button
                        onClick={() => {
                          setSelectedMarket(market)
                          setBetOutcome('YES')
                        }}
                        className={`p-3 rounded-lg ${isDark ? 'bg-green-900 hover:bg-green-800' : 'bg-green-100 hover:bg-green-200'} transition-colors`}
                      >
                        <p className="text-xs mb-1">YES</p>
                        <p className={`text-lg font-bold ${isDark ? 'text-green-400' : 'text-green-600'}`}>
                          {(market.yes_price * 100).toFixed(0)}%
                        </p>
                      </button>
                      <button
                        onClick={() => {
                          setSelectedMarket(market)
                          setBetOutcome('NO')
                        }}
                        className={`p-3 rounded-lg ${isDark ? 'bg-red-900 hover:bg-red-800' : 'bg-red-100 hover:bg-red-200'} transition-colors`}
                      >
                        <p className="text-xs mb-1">NO</p>
                        <p className={`text-lg font-bold ${isDark ? 'text-red-400' : 'text-red-600'}`}>
                          {(market.no_price * 100).toFixed(0)}%
                        </p>
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Your Bets */}
            <div>
              <h2 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                Your Bets
              </h2>
              <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg shadow overflow-hidden`}>
                <table className="min-w-full divide-y divide-gray-700">
                  <thead className={isDark ? 'bg-gray-700' : 'bg-gray-50'}>
                    <tr>
                      <th className={`px-6 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-500'} uppercase tracking-wider`}>
                        Market
                      </th>
                      <th className={`px-6 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-500'} uppercase tracking-wider`}>
                        Outcome
                      </th>
                      <th className={`px-6 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-500'} uppercase tracking-wider`}>
                        Amount
                      </th>
                      <th className={`px-6 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-500'} uppercase tracking-wider`}>
                        Potential Payout
                      </th>
                    </tr>
                  </thead>
                  <tbody className={`divide-y ${isDark ? 'divide-gray-700' : 'divide-gray-200'}`}>
                    {bets.length === 0 ? (
                      <tr>
                        <td colSpan={4} className={`px-6 py-4 text-center ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                          No bets placed yet
                        </td>
                      </tr>
                    ) : (
                      bets.map(bet => {
                        const market = markets.find(m => m.id === bet.market_id)
                        return (
                          <tr key={bet.id}>
                            <td className={`px-6 py-4 whitespace-nowrap ${isDark ? 'text-white' : 'text-gray-900'}`}>
                              {market?.question || 'Unknown'}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap">
                              <span className={`px-2 py-1 text-xs rounded ${
                                bet.outcome === 'YES' 
                                  ? 'bg-green-500 text-white' 
                                  : 'bg-red-500 text-white'
                              }`}>
                                {bet.outcome}
                              </span>
                            </td>
                            <td className={`px-6 py-4 whitespace-nowrap ${isDark ? 'text-white' : 'text-gray-900'}`}>
                              {formatCurrency(bet.amount)}
                            </td>
                            <td className={`px-6 py-4 whitespace-nowrap ${isDark ? 'text-green-400' : 'text-green-600'}`}>
                              {formatCurrency(bet.payout)}
                            </td>
                          </tr>
                        )
                      })
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </>
        )}
      </main>

      {/* Bet Modal */}
      {selectedMarket && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg p-6 max-w-md w-full mx-4`}>
            <h3 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
              Place Bet
            </h3>
            <p className={`mb-4 ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>
              {selectedMarket.question}
            </p>
            <div className="mb-4">
              <label className={`block text-sm font-medium mb-2 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>
                Amount (USD)
              </label>
              <input
                type="number"
                value={betAmount}
                onChange={(e) => setBetAmount(e.target.value)}
                className={`w-full px-3 py-2 rounded-lg border ${
                  isDark 
                    ? 'bg-gray-700 border-gray-600 text-white' 
                    : 'bg-white border-gray-300 text-gray-900'
                } focus:outline-none focus:ring-2 focus:ring-blue-500`}
                placeholder="Enter amount"
              />
            </div>
            <div className="mb-4">
              <label className={`block text-sm font-medium mb-2 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>
                Outcome
              </label>
              <div className="flex gap-3">
                <button
                  onClick={() => setBetOutcome('YES')}
                  className={`flex-1 p-3 rounded-lg border-2 ${
                    betOutcome === 'YES'
                      ? 'border-green-500 bg-green-500/20'
                      : 'border-gray-600'
                  } transition-colors`}
                >
                  <p className="font-semibold">YES</p>
                  <p className="text-sm">{(selectedMarket.yes_price * 100).toFixed(0)}%</p>
                </button>
                <button
                  onClick={() => setBetOutcome('NO')}
                  className={`flex-1 p-3 rounded-lg border-2 ${
                    betOutcome === 'NO'
                      ? 'border-red-500 bg-red-500/20'
                      : 'border-gray-600'
                  } transition-colors`}
                >
                  <p className="font-semibold">NO</p>
                  <p className="text-sm">{(selectedMarket.no_price * 100).toFixed(0)}%</p>
                </button>
              </div>
            </div>
            {betAmount && parseFloat(betAmount) > 0 && (
              <div className={`mb-4 p-3 rounded-lg ${isDark ? 'bg-gray-700' : 'bg-gray-100'}`}>
                <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>
                  Potential Payout
                </p>
                <p className={`text-xl font-bold ${isDark ? 'text-green-400' : 'text-green-600'}`}>
                  {formatCurrency(
                    parseFloat(betAmount) * (
                      betOutcome === 'YES' 
                        ? (1 / selectedMarket.yes_price) 
                        : (1 / selectedMarket.no_price)
                    )
                  )}
                </p>
              </div>
            )}
            <div className="flex gap-3">
              <button
                onClick={handlePlaceBet}
                className="flex-1 bg-blue-600 hover:bg-blue-700 text-white py-2 rounded-lg transition-colors"
              >
                Place Bet
              </button>
              <button
                onClick={() => setSelectedMarket(null)}
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
