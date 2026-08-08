// Terminal Orderbook API Route
import { NextRequest } from 'next/server';
import { proxyGet } from '../../../_proxy';

export async function GET(req: NextRequest, { params }: { params: { symbol: string } }) {
  const symbol = params.symbol.replace('-', '/');
  return proxyGet(req, `/terminal/orderbook/${symbol}`);
}
