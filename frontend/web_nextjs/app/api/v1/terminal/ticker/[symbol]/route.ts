// Terminal Ticker API Route
import { NextRequest, NextResponse } from 'next/server';
import { proxyGet, encodePathSegment } from '../../../_proxy';

export async function GET(req: NextRequest, { params }: { params: { symbol: string } }) {
  const rawSymbol = params.symbol;
  if (!/^[A-Za-z0-9]{1,20}(-[A-Za-z0-9]{1,20})?$/.test(rawSymbol)) {
    return NextResponse.json({ success: false, error: 'invalid symbol' }, { status: 400 });
  }
  const symbol = encodePathSegment(rawSymbol.replace('-', '/'));
  return proxyGet(req, `/terminal/ticker/${symbol}`);
}
