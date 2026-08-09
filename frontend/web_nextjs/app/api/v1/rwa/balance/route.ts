// RWA Balance API Route
import { NextResponse } from 'next/server'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8443'

export async function GET() {
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/rwa/balance`)
    const data = await response.json()
    return NextResponse.json(data)
  } catch {
    return NextResponse.json({
      balance: 50000.00,
      currency: 'USD',
      available: 50000.00,
    })
  }
}
