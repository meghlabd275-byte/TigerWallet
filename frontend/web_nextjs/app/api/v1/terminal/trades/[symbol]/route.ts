// Terminal Trades API Route
import { NextRequest, NextResponse } from 'next/server'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

export async function GET(request: NextRequest, { params }: { params: { symbol: string } }) {
  const symbol = params.symbol.replace('-', '/')
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/terminal/trades/${symbol}`)
    const data = await response.json()
    return NextResponse.json(data)
  } catch {
    const now = Math.floor(Date.now() / 1000)
    return NextResponse.json({
      symbol,
      trades: [
        { id: 't1', price: 3500.00, amount: 1.5, time: now, side: 'buy' },
        { id: 't2', price: 3500.50, amount: 0.8, time: now - 60, side: 'sell' },
        { id: 't3', price: 3499.00, amount: 2.0, time: now - 120, side: 'buy' },
        { id: 't4', price: 3498.50, amount: 1.2, time: now - 180, side: 'sell' },
        { id: 't5', price: 3498.00, amount: 0.5, time: now - 240, side: 'buy' },
      ],
    })
  }
}
