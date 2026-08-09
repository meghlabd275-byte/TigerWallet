// Prediction Markets API Route
import { NextResponse } from 'next/server'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8443'

export async function GET() {
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/prediction/markets`)
    const data = await response.json()
    return NextResponse.json(data)
  } catch {
    return NextResponse.json({
      markets: [
        { id: 'pm1', question: 'Will ETH reach $5000 by Dec 2026?', yes_price: 0.45, no_price: 0.55, volume: 500000, end_time: 1767225600 },
        { id: 'pm2', question: 'Will BTC reach $100k by June 2026?', yes_price: 0.60, no_price: 0.40, volume: 1200000, end_time: 1759300800 },
      ],
    })
  }
}
