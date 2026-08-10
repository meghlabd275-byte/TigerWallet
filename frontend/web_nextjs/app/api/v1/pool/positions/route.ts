import { NextRequest } from 'next/server';
import { proxyGetFrom, POOL_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, POOL_SERVICE_URL, '/api/v1/pool/positions');
}
