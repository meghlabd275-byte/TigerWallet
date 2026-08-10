import { NextRequest } from 'next/server';
import { proxyGetFrom, proxyMutationFrom, RWA_SERVICE_URL } from '../../_proxy';

// List the caller's RWA orders (auth required). No empty-list fallback.
export async function GET(req: NextRequest) {
  return proxyGetFrom(req, RWA_SERVICE_URL, '/api/v1/rwa/orders');
}

// Place a buy/sell order for a tokenized RWA (auth required). No fake order_id.
export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, RWA_SERVICE_URL, '/api/v1/rwa/orders', 'POST');
}
