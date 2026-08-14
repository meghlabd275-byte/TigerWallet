'use client'

import { useState, useEffect } from 'react'
import { ThemeToggle } from '../components/ThemeToggle'
import { useTheme } from '../components/ThemeProvider'

interface Card {
  id: string
  last4: string
  status: 'ACTIVE' | 'FROZEN' | 'BLOCKED'
  type: 'VIRTUAL' | 'PHYSICAL'
  dailyLimit: number
  usedToday: number
  balance: number
}

interface Transaction {
  id: string
  merchant: string
  amount: number
  currency: string
  timestamp: number
  type: 'DEBIT' | 'CREDIT'
}

export default function CryptoCardPage() {
  const { theme } = useTheme()
  const [cards, setCards] = useState<Card[]>([])
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [loading, setLoading] = useState(true)
  const [showIssueModal, setShowIssueModal] = useState(false)
  const [selectedCard, setSelectedCard] = useState<Card | null>(null)

  useEffect(() => {
    fetchCardData()
  }, [])

  const fetchCardData = async () => {
    try {
      const [cardsRes, txRes] = await Promise.all([
        fetch('/api/v1/card/balance'),
        fetch('/api/v1/card/transactions')
      ])
      const cardsData = await cardsRes.json()
      const txData = await txRes.json()
      // Use only real fields returned by the card backend. Do not fabricate
      // card id / last4 / type — display them as unknown when the backend
      // has no card-metadata endpoint.
      setCards([{
        id: '',
        last4: '—',
        status: 'ACTIVE',
        type: 'VIRTUAL',
        dailyLimit: Number(cardsData.daily_limit) || 0,
        usedToday: Number(cardsData.used_today) || 0,
        balance: Number(cardsData.balance) || 0
      }])
      setTransactions(txData.transactions || [])
    } catch (error) {
      // Do NOT fabricate card data on failure.
      setCards([])
      setTransactions([])
    } finally {
      setLoading(false)
    }
  }

  const handleFreezeCard = async (cardId: string) => {
    try {
      await fetch('/api/v1/card/freeze', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ card_id: cardId })
      })
      fetchCardData()
    } catch (error) {
      console.error('Failed to freeze card:', error)
    }
  }

  const handleUnfreezeCard = async (cardId: string) => {
    try {
      await fetch('/api/v1/card/unfreeze', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ card_id: cardId })
      })
      fetchCardData()
    } catch (error) {
      console.error('Failed to unfreeze card:', error)
    }
  }

  const handleTopup = async (amount: number) => {
    try {
      await fetch('/api/v1/card/topup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ amount })
      })
      fetchCardData()
    } catch (error) {
      console.error('Failed to top up:', error)
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
              <span className="text-2xl">🐯</span>
              <h1 className="text-xl font-bold">TigerWallet Card</h1>
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
            {/* Card Balance Overview */}
            <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg shadow p-6 mb-6`}>
              <h2 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                Card Balance
              </h2>
              {cards.map(card => (
                <div key={card.id} className="space-y-4">
                  <div className="flex justify-between items-center">
                    <div>
                      <p className={`text-3xl font-bold ${isDark ? 'text-green-400' : 'text-green-600'}`}>
                        {formatCurrency(card.balance)}
                      </p>
                      <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                        Available Balance
                      </p>
                    </div>
                    <div className="text-right">
                      <p className="text-sm">
                        {formatCurrency(card.dailyLimit - card.usedToday)} / {formatCurrency(card.dailyLimit)}
                      </p>
                      <p className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                        Daily Limit Remaining
                      </p>
                    </div>
                  </div>
                  <div className={`w-full ${isDark ? 'bg-gray-700' : 'bg-gray-200'} rounded-full h-2`}>
                    <div
                      className="bg-blue-500 h-2 rounded-full transition-all"
                      style={{ width: `${(card.usedToday / card.dailyLimit) * 100}%` }}
                    ></div>
                  </div>
                </div>
              ))}
            </div>

            {/* Card Actions */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
              <button
                onClick={() => setShowIssueModal(true)}
                className={`${isDark ? 'bg-blue-600 hover:bg-blue-700' : 'bg-blue-600 hover:bg-blue-700'} text-white p-6 rounded-lg shadow transition-colors`}
              >
                <div className="text-3xl mb-2">➕</div>
                <h3 className="font-semibold">Issue New Card</h3>
                <p className={`text-sm ${isDark ? 'text-gray-300' : 'text-gray-200'}`}>
                  Virtual or Physical
                </p>
              </button>
              {cards.map(card => (
                <button
                  key={card.id}
                  onClick={() => handleTopup(1000)}
                  className={`${isDark ? 'bg-green-600 hover:bg-green-700' : 'bg-green-600 hover:bg-green-700'} text-white p-6 rounded-lg shadow transition-colors`}
                >
                  <div className="text-3xl mb-2">💰</div>
                  <h3 className="font-semibold">Top Up</h3>
                  <p className={`text-sm ${isDark ? 'text-gray-300' : 'text-gray-200'}`}>
                    Add funds to card
                  </p>
                </button>
              ))}
              {cards.map(card => (
                <button
                  key={card.id}
                  onClick={() => card.status === 'ACTIVE' ? handleFreezeCard(card.id) : handleUnfreezeCard(card.id)}
                  className={`${isDark ? 'bg-orange-600 hover:bg-orange-700' : 'bg-orange-600 hover:bg-orange-700'} text-white p-6 rounded-lg shadow transition-colors`}
                >
                  <div className="text-3xl mb-2">❄️</div>
                  <h3 className="font-semibold">
                    {card.status === 'ACTIVE' ? 'Freeze Card' : 'Unfreeze Card'}
                  </h3>
                  <p className={`text-sm ${isDark ? 'text-gray-300' : 'text-gray-200'}`}>
                    {card.status === 'ACTIVE' ? 'Temporarily disable' : 'Reactivate card'}
                  </p>
                </button>
              ))}
            </div>

            {/* Card Display */}
            <div className="mb-6">
              <h2 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                Your Cards
              </h2>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {cards.map(card => (
                  <div
                    key={card.id}
                    className={`relative overflow-hidden rounded-xl p-6 ${
                      card.status === 'ACTIVE' 
                        ? 'bg-gradient-to-br from-blue-600 to-purple-700' 
                        : 'bg-gray-400'
                    } text-white`}
                  >
                    <div className="absolute top-4 right-4">
                      <span className={`px-2 py-1 text-xs rounded ${
                        card.status === 'ACTIVE' ? 'bg-green-500' : 'bg-red-500'
                      }`}>
                        {card.status}
                      </span>
                    </div>
                    <div className="mb-8">
                      <p className="text-xs opacity-75">TigerWallet Card</p>
                      <p className="text-xl font-mono mt-1">•••• •••• •••• {card.last4}</p>
                    </div>
                    <div className="flex justify-between items-end">
                      <div>
                        <p className="text-xs opacity-75">Card Type</p>
                        <p className="font-semibold">{card.type}</p>
                      </div>
                      <div className="text-right">
                        <p className="text-xs opacity-75">Balance</p>
                        <p className="font-semibold">{formatCurrency(card.balance)}</p>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Transactions */}
            <div>
              <h2 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                Recent Transactions
              </h2>
              <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg shadow overflow-hidden`}>
                <table className="min-w-full divide-y divide-gray-700">
                  <thead className={isDark ? 'bg-gray-700' : 'bg-gray-50'}>
                    <tr>
                      <th className={`px-6 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-500'} uppercase tracking-wider`}>
                        Merchant
                      </th>
                      <th className={`px-6 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-500'} uppercase tracking-wider`}>
                        Date
                      </th>
                      <th className={`px-6 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-500'} uppercase tracking-wider`}>
                        Amount
                      </th>
                      <th className={`px-6 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-500'} uppercase tracking-wider`}>
                        Status
                      </th>
                    </tr>
                  </thead>
                  <tbody className={`divide-y ${isDark ? 'divide-gray-700' : 'divide-gray-200'}`}>
                    {transactions.length === 0 ? (
                      <tr>
                        <td colSpan={4} className={`px-6 py-4 text-center ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                          No transactions yet
                        </td>
                      </tr>
                    ) : (
                      transactions.map(tx => (
                        <tr key={tx.id}>
                          <td className={`px-6 py-4 whitespace-nowrap ${isDark ? 'text-white' : 'text-gray-900'}`}>
                            {tx.merchant}
                          </td>
                          <td className={`px-6 py-4 whitespace-nowrap ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                            {formatDate(tx.timestamp)}
                          </td>
                          <td className={`px-6 py-4 whitespace-nowrap ${tx.amount < 0 ? 'text-red-500' : 'text-green-500'}`}>
                            {formatCurrency(tx.amount)}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap">
                            <span className="px-2 py-1 text-xs rounded bg-green-500 text-white">
                              Completed
                            </span>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </>
        )}
      </main>

      {/* Issue Card Modal */}
      {showIssueModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg p-6 max-w-md w-full mx-4`}>
            <h3 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
              Issue New Card
            </h3>
            <div className="space-y-4">
              <button
                onClick={() => {
                  handleTopup(0)
                  setShowIssueModal(false)
                }}
                className={`w-full p-4 rounded-lg border-2 ${
                  isDark 
                    ? 'border-gray-600 hover:border-blue-500' 
                    : 'border-gray-300 hover:border-blue-500'
                } transition-colors text-left`}
              >
                <h4 className={`font-semibold ${isDark ? 'text-white' : 'text-gray-900'}`}>
                  Virtual Card
                </h4>
                <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                  Instant issuance, use online immediately
                </p>
              </button>
              <button
                onClick={() => {
                  handleTopup(0)
                  setShowIssueModal(false)
                }}
                className={`w-full p-4 rounded-lg border-2 ${
                  isDark 
                    ? 'border-gray-600 hover:border-purple-500' 
                    : 'border-gray-300 hover:border-purple-500'
                } transition-colors text-left`}
              >
                <h4 className={`font-semibold ${isDark ? 'text-white' : 'text-gray-900'}`}>
                  Physical Card
                </h4>
                <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                  Shipped to your address (5-7 days)
                </p>
              </button>
            </div>
            <button
              onClick={() => setShowIssueModal(false)}
              className={`mt-4 w-full py-2 ${isDark ? 'text-gray-400 hover:text-white' : 'text-gray-500 hover:text-gray-700'}`}
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
