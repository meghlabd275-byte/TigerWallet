import { NextRequest } from 'next/server';
import { proxyGetFrom, BUG_BOUNTY_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, BUG_BOUNTY_SERVICE_URL, '/api/v1/stats');
}
