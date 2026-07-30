// Terminal Kline API Route
import { NextRequest, NextResponse } from 'next/server'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

export async function GET(request: NextRequest, { params }: { params: { symbol: string } }) {
  const symbol = params.symbol.replace('-', '/')
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/terminal/kline/${symbol}`)
    const data = await response.json()
    return NextResponse.json(data)
  } catch {
    const now = Math.floor(Date.now() / 1000)
    const candles = []
    for (let i = 0; i < 50; i++) {
      const time = now - (50 - i) * 300
      const open = 3400 + Math.random() * 100
      const close = open + (Math.random() - 0.5) * 20
      const high = Math.max(open, close) + Math.random() * 10
      const low = Math.min(open, close) - Math.random() * 10
      const volume = 1000 + Math.random() * 5000
      candles.push([time, open, high, low, close, volume])
    }
    return NextResponse.json({ symbol, candles })
  }
}
