// Terminal Ticker API Route
import { NextRequest, NextResponse } from 'next/server'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

export async function GET(request: NextRequest, { params }: { params: { symbol: string } }) {
  const symbol = params.symbol.replace('-', '/')
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/terminal/ticker/${symbol}`)
    const data = await response.json()
    return NextResponse.json(data)
  } catch {
    const basePrice = symbol.includes('BTC') ? 65000 : symbol.includes('ETH') ? 3500 : 150
    return NextResponse.json({
      symbol,
      price: basePrice,
      change24h: (Math.random() - 0.5) * 10,
      high24h: basePrice * 1.03,
      low24h: basePrice * 0.97,
      volume24h: 1000000 + Math.random() * 1000000,
    })
  }
}
