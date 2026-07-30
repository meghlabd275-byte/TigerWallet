// Terminal Orderbook API Route
import { NextRequest, NextResponse } from 'next/server'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

export async function GET(request: NextRequest, { params }: { params: { symbol: string } }) {
  const symbol = params.symbol.replace('-', '/')
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/terminal/orderbook/${symbol}`)
    const data = await response.json()
    return NextResponse.json(data)
  } catch {
    return NextResponse.json({
      symbol,
      bids: [
        [3500.00, 10.5, 36750],
        [3499.50, 25.0, 87487.50],
        [3499.00, 15.0, 52485],
      ],
      asks: [
        [3500.50, 15.0, 52507.50],
        [3501.00, 30.0, 105030],
        [3501.50, 20.0, 70030],
      ],
    })
  }
}
