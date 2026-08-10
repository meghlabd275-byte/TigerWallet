// Terminal Trades API Route — proxies to the matching engine /trades endpoint.
import { NextRequest } from 'next/server';
import { proxyGetFrom, MATCHING_ENGINE_URL } from '../../../_proxy';

export async function GET(req: NextRequest, { params }: { params: { symbol: string } }) {
  const symbol = params.symbol.replace('-', '/');
  const url = new URL(req.url);
  const search = url.searchParams.toString();
  return proxyGetFrom(req, MATCHING_ENGINE_URL, `/trades${search ? `?${search}` : ''}&symbol=${encodeURIComponent(symbol)}`);
}
