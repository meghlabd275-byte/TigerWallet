import { NextRequest } from 'next/server';
import { proxyGetFrom, RWA_SERVICE_URL } from '../../_proxy';

// Get the caller's RWA portfolio with live valuations (auth required). No fallback.
export async function GET(req: NextRequest) {
  return proxyGetFrom(req, RWA_SERVICE_URL, '/api/v1/rwa/portfolio');
}
