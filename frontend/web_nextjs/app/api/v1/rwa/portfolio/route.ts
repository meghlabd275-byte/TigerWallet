// RWA Portfolio API Route
import { NextResponse } from 'next/server'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

export async function GET() {
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/rwa/portfolio`)
    const data = await response.json()
    return NextResponse.json(data)
  } catch {
    return NextResponse.json({
      holdings: [],
      total_value: 0,
    })
  }
}
