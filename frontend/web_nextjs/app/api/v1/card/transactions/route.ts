import { NextRequest } from 'next/server';
import { proxyGetFrom, proxyMutationFrom, CARD_SERVICE_URL } from '../../_proxy';

// List the caller's card transactions (auth required). No fake merchant data.
export async function GET(req: NextRequest) {
  return proxyGetFrom(req, CARD_SERVICE_URL, '/api/v1/card/transactions');
}

// Create (authorize) a new card transaction (auth required). Debits the real
// available balance; rejects if it would exceed available credit.
export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, CARD_SERVICE_URL, '/api/v1/card/transactions', 'POST');
}
