import { NextRequest } from 'next/server';
import { proxyGetFrom, CARD_SERVICE_URL } from '../../_proxy';

// Get the caller's card balance (auth required). No fabricated balance.
export async function GET(req: NextRequest) {
  return proxyGetFrom(req, CARD_SERVICE_URL, '/api/v1/card/balance');
}
