// Perpetual Markets API Route
import { NextRequest, NextResponse } from 'next/server'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

export async function GET() {
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/perpetual/markets`)
    const data = await response.json()
    return NextResponse.json(data)
  } catch {
    return NextResponse.json({
      markets: [
        { symbol: 'ETH/USDT', name: 'Ethereum', price: 3500.00, change24h: 2.5, volume24h: 1250000000, max_leverage: 100, funding_rate: 0.01 },
        { symbol: 'BTC/USDT', name: 'Bitcoin', price: 65000.00, change24h: 1.8, volume24h: 2500000000, max_leverage: 100, funding_rate: 0.01 },
        { symbol: 'SOL/USDT', name: 'Solana', price: 150.00, change24h: 5.2, volume24h: 850000000, max_leverage: 50, funding_rate: 0.02 },
      ],
    })
  }
}
