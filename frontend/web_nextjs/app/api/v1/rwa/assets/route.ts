// RWA Assets API Route
import { NextResponse } from 'next/server'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8443'

export async function GET() {
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/rwa/assets`)
    const data = await response.json()
    return NextResponse.json(data)
  } catch {
    return NextResponse.json({
      assets: [
        { id: 'rwa1', name: 'Tesla Inc', symbol: 'TSLA', type: 'STOCK', price: 250.00, change24h: 1.5, volume24h: 50000000 },
        { id: 'rwa2', name: 'Apple Inc', symbol: 'AAPL', type: 'STOCK', price: 180.00, change24h: 0.8, volume24h: 80000000 },
        { id: 'rwa3', name: 'Gold', symbol: 'XAU', type: 'COMMODITY', price: 2000.00, change24h: 0.2, volume24h: 200000000 },
        { id: 'rwa4', name: 'SPY ETF', symbol: 'SPY', type: 'ETF', price: 450.00, change24h: 0.5, volume24h: 100000000 },
      ],
    })
  }
}
