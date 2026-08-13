import { NextRequest } from 'next/server';
import { swapProxyGet } from '../../../_proxy';

export async function GET(req: NextRequest) {
  const symbol = req.nextUrl.pathname.split('/').pop();
  return swapProxyGet(req, `/swap/tokens/${symbol}`);
}
