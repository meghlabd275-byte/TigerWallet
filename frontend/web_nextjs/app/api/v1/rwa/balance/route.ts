import { NextRequest } from 'next/server';
import { proxyGetFrom, RWA_SERVICE_URL } from '../../_proxy';

// Get the caller's RWA balance (auth required). No mock balance fallback.
export async function GET(req: NextRequest) {
  return proxyGetFrom(req, RWA_SERVICE_URL, '/api/v1/rwa/balance');
}
