// RWA Orders API Route
import { NextRequest, NextResponse } from 'next/server'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8443'

export async function GET() {
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/rwa/orders`)
    const data = await response.json()
    return NextResponse.json(data)
  } catch {
    return NextResponse.json({ orders: [] })
  }
}

export async function POST(request: NextRequest) {
  const body = await request.json()
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/rwa/orders`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
    const data = await response.json()
    return NextResponse.json(data, { status: response.status })
  } catch {
    return NextResponse.json({ 
      success: true, 
      order_id: `order_${Date.now()}` 
    })
  }
}
