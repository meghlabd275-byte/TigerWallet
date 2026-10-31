// Terminal Trades API Route — proxies to the matching engine /trades endpoint.
import { NextRequest, NextResponse } from 'next/server';
import { proxyGetFrom, MATCHING_ENGINE_URL, encodePathSegment } from '../../../_proxy';

export async function GET(req: NextRequest, { params }: { params: { symbol: string } }) {
  const rawSymbol = params.symbol;
  if (!/^[A-Za-z0-9]{1,20}(-[A-Za-z0-9]{1,20})?$/.test(rawSymbol)) {
    return NextResponse.json({ success: false, error: 'invalid symbol' }, { status: 400 });
  }
  const symbol = encodePathSegment(rawSymbol.replace('-', '/'));
  const url = new URL(req.url);
  const search = url.searchParams.toString();
  return proxyGetFrom(req, MATCHING_ENGINE_URL, `/trades${search ? `?${search}` : ''}&symbol=${encodeURIComponent(symbol)}`);
}
