import { NextRequest, NextResponse } from 'next/server';

export async function GET(req: NextRequest) {
  // User widget preferences are stored client-side; return empty defaults
  return NextResponse.json({ widgets: [], theme: 'dark' });
}

export async function POST(req: NextRequest) {
  // Preferences are saved client-side; acknowledge the save
  return NextResponse.json({ success: true });
}
