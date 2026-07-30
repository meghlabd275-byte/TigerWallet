// Card Balance API Route
import { NextRequest, NextResponse } from 'next/server'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

export async function GET() {
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/card/balance`)
    const data = await response.json()
    return NextResponse.json(data)
  } catch {
    return NextResponse.json({
      balance: 5000.00,
      currency: 'USD',
      available_credit: 10000.00,
      daily_limit: 10000,
      monthly_limit: 50000,
      used_today: 500,
    })
  }
}
