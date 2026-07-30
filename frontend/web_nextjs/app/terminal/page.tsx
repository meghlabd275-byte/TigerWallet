'use client'

import { useState, useEffect, useRef } from 'react'
import { ThemeToggle } from '../components/ThemeToggle'
import { useTheme } from '../components/ThemeProvider'

interface OrderBookEntry {
  price: number
  amount: number
  total: number
}

interface Trade {
  id: string
  price: number
  amount: number
  time: number
  side: 'buy' | 'sell'
}

interface Kline {
  time: number
  open: number
  high: number
  low: number
  close: number
  volume: number
}

interface Ticker {
  price: number
  change24h: number
  high24h: number
  low24h: number
  volume24h: number
}

const SYMBOLS = ['ETH/USDT', 'BTC/USDT', 'SOL/USDT', 'BNB/USDT', 'XRP/USDT']

export default function TerminalPage() {
  const { theme } = useTheme()
  const [symbol, setSymbol] = useState('ETH/USDT')
  const [orderBook, setOrderBook] = useState<{ bids: OrderBookEntry[]; asks: OrderBookEntry[] }>({ bids: [], asks: [] })
  const [trades, setTrades] = useState<Trade[]>([])
  const [klines, setKlines] = useState<Kline[]>([])
  const [ticker, setTicker] = useState<Ticker | null>(null)
  const [loading, setLoading] = useState(true)
  const wsRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    fetchData()
    return () => {
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [symbol])

  const fetchData = async () => {
    setLoading(true)
    try {
      const [orderbookRes, tradesRes, klineRes, tickerRes] = await Promise.all([
        fetch(`/api/v1/terminal/orderbook/${symbol.replace('/', '-')}`),
        fetch(`/api/v1/terminal/trades/${symbol.replace('/', '-')}`),
        fetch(`/api/v1/terminal/kline/${symbol.replace('/', '-')}`),
        fetch(`/api/v1/terminal/ticker/${symbol.replace('/', '-')}`)
      ])
      const obData = await orderbookRes.json()
      const tradesData = await tradesRes.json()
      const klineData = await klineRes.json()
      const tickerData = await tickerRes.json()
      
      setOrderBook(obData)
      setTrades(tradesData.trades || [])
      setKlines(klineData.candles || [])
      setTicker(tickerData)
    } catch (error) {
      console.error('Failed to fetch data:', error)
      // Fallback data
      setOrderBook({
        bids: [
          { price: 3500.00, amount: 10.5, total: 36750 },
          { price: 3499.50, amount: 25.0, total: 87487.50 },
          { price: 3499.00, amount: 15.0, total: 52485 },
        ],
        asks: [
          { price: 3500.50, amount: 15.0, total: 52507.50 },
          { price: 3501.00, amount: 30.0, total: 105030 },
          { price: 3501.50, amount: 20.0, total: 70030 },
        ]
      })
      setTrades([
        { id: 't1', price: 3500.00, amount: 1.5, time: Date.now() / 1000, side: 'buy' },
        { id: 't2', price: 3500.50, amount: 0.8, time: Date.now() / 1000 - 60, side: 'sell' },
      ])
      setKlines([
        { time: 1700000000, open: 3400, high: 3500, low: 3450, close: 3500, volume: 10000 },
        { time: 1700003600, open: 3500, high: 3520, low: 3480, close: 3510, volume: 12000 },
      ])
      setTicker({ price: 3500, change24h: 2.5, high24h: 3600, low24h: 3400, volume24h: 1000000 })
    } finally {
      setLoading(false)
    }
  }

  const formatNumber = (num: number, decimals = 2) => num.toLocaleString('en-US', { minimumFractionDigits: decimals, maximumFractionDigits: decimals })
  const formatTime = (timestamp: number) => new Date(timestamp * 1000).toLocaleTimeString()

  const isDark = theme === 'dark'

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900 text-white' : 'bg-gray-50 text-gray-900'}`}>
      {/* Header */}
      <header className={`${isDark ? 'bg-gray-800' : 'bg-white'} shadow-sm border-b ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
        <div className="max-w-full mx-auto px-4 py-3">
          <div className="flex justify-between items-center">
            <div className="flex items-center space-x-3">
              <span className="text-2xl">📊</span>
              <h1 className="text-xl font-bold">TigerWallet Terminal</h1>
            </div>
            <div className="flex items-center space-x-4">
              <select
                value={symbol}
                onChange={(e) => setSymbol(e.target.value)}
                className={`px-3 py-1.5 rounded-lg border ${
                  isDark ? 'bg-gray-700 border-gray-600' : 'bg-white border-gray-300'
                }`}
              >
                {SYMBOLS.map(s => <option key={s} value={s}>{s}</option>)}
              </select>
              <ThemeToggle />
            </div>
          </div>
        </div>
      </header>

      {/* Ticker Bar */}
      {ticker && (
        <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} border-b ${isDark ? 'border-gray-700' : 'border-gray-200'} px-4 py-2`}>
          <div className="flex justify-between items-center text-sm">
            <div className="flex items-center space-x-6">
              <span className="font-bold text-lg">{ticker.price.toFixed(2)}</span>
              <span className={ticker.change24h >= 0 ? 'text-green-500' : 'text-red-500'}>
                {ticker.change24h >= 0 ? '+' : ''}{ticker.change24h.toFixed(2)}%
              </span>
            </div>
            <div className="flex items-center space-x-6">
              <span>24h High: <span className="font-medium">{ticker.high24h.toFixed(2)}</span></span>
              <span>24h Low: <span className="font-medium">{ticker.low24h.toFixed(2)}</span></span>
              <span>24h Vol: <span className="font-medium">{(ticker.volume24h / 1000000).toFixed(2)}M</span></span>
            </div>
          </div>
        </div>
      )}

      <div className="flex h-[calc(100vh-120px)]">
        {/* Chart Area */}
        <div className="flex-1 p-4">
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg h-full p-4`}>
            <h3 className={`text-sm font-semibold mb-2 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Price Chart</h3>
            <div className="h-full flex items-center justify-center">
              {loading ? (
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
              ) : (
                <div className="w-full">
                  {/* Simple chart visualization */}
                  <div className="flex items-end space-x-1 h-64">
                    {klines.map((k, i) => {
                      const height = ((k.close - (ticker?.low24h || 3000)) / ((ticker?.high24h || 4000) - (ticker?.low24h || 3000))) * 100
                      return (
                        <div
                          key={i}
                          className={`flex-1 ${k.close >= k.open ? 'bg-green-500' : 'bg-red-500'}`}
                          style={{ height: `${Math.max(5, height)}%` }}
                          title={`O: ${k.open} H: ${k.high} L: ${k.low} C: ${k.close}`}
                        ></div>
                      )
                    })}
                  </div>
                  <div className="flex justify-between mt-2 text-xs text-gray-500">
                    <span>10:00</span>
                    <span>12:00</span>
                    <span>14:00</span>
                    <span>16:00</span>
                    <span>18:00</span>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Order Book */}
        <div className="w-72 p-4">
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg h-full p-4`}>
            <h3 className={`text-sm font-semibold mb-2 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Order Book</h3>
            
            {/* Asks */}
            <div className="mb-2">
              <div className={`flex text-xs ${isDark ? 'text-gray-500' : 'text-gray-400'} mb-1`}>
                <span className="flex-1">Price</span>
                <span className="flex-1 text-right">Amount</span>
                <span className="flex-1 text-right">Total</span>
              </div>
              {[...orderBook.asks].reverse().map((ask, i) => (
                <div key={`ask-${i}`} className="flex text-xs relative">
                  <div className="absolute right-0 top-0 bottom-0 bg-red-500/20" style={{ width: `${(ask.amount / 30) * 100}%` }}></div>
                  <span className="flex-1 text-red-500 relative z-10">{formatNumber(ask.price)}</span>
                  <span className="flex-1 text-right relative z-10">{formatNumber(ask.amount, 4)}</span>
                  <span className="flex-1 text-right relative z-10">{formatNumber(ask.total)}</span>
                </div>
              ))}
            </div>

            {/* Spread */}
            <div className={`text-center py-2 ${isDark ? 'bg-gray-700' : 'bg-gray-100'} rounded my-2`}>
              <span className="font-bold">{orderBook.asks[0]?.price.toFixed(2) || '0.00'}</span>
            </div>

            {/* Bids */}
            <div>
              {orderBook.bids.map((bid, i) => (
                <div key={`bid-${i}`} className="flex text-xs relative">
                  <div className="absolute right-0 top-0 bottom-0 bg-green-500/20" style={{ width: `${(bid.amount / 30) * 100}%` }}></div>
                  <span className="flex-1 text-green-500 relative z-10">{formatNumber(bid.price)}</span>
                  <span className="flex-1 text-right relative z-10">{formatNumber(bid.amount, 4)}</span>
                  <span className="flex-1 text-right relative z-10">{formatNumber(bid.total)}</span>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Recent Trades */}
        <div className="w-64 p-4">
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg h-full p-4`}>
            <h3 className={`text-sm font-semibold mb-2 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Recent Trades</h3>
            <div className={`flex text-xs ${isDark ? 'text-gray-500' : 'text-gray-400'} mb-2`}>
              <span className="flex-1">Price</span>
              <span className="flex-1 text-right">Amount</span>
              <span className="flex-1 text-right">Time</span>
            </div>
            <div className="overflow-y-auto h-64">
              {trades.map(trade => (
                <div key={trade.id} className="flex text-xs py-1">
                  <span className={`flex-1 ${trade.side === 'buy' ? 'text-green-500' : 'text-red-500'}`}>
                    {formatNumber(trade.price)}
                  </span>
                  <span className="flex-1 text-right">{formatNumber(trade.amount, 4)}</span>
                  <span className="flex-1 text-right">{formatTime(trade.time)}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
