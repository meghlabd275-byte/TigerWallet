// Card Transactions API Route
import { NextRequest, NextResponse } from 'next/server'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

export async function GET() {
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/card/transactions`)
    const data = await response.json()
    return NextResponse.json(data)
  } catch {
    return NextResponse.json({
      transactions: [
        { id: 'tx1', merchant: 'Amazon', amount: -50.00, currency: 'USD', timestamp: Math.floor(Date.now() / 1000) },
        { id: 'tx2', merchant: 'Apple Store', amount: -999.00, currency: 'USD', timestamp: Math.floor(Date.now() / 1000) - 86400 },
      ],
    })
  }
}
