import { NextRequest } from 'next/server';
import { proxyGetFrom, RWA_SERVICE_URL } from '../../_proxy';

// List tokenized real-world assets with live USD prices. No mock fallback.
export async function GET(req: NextRequest) {
  return proxyGetFrom(req, RWA_SERVICE_URL, '/api/v1/rwa/assets');
}
