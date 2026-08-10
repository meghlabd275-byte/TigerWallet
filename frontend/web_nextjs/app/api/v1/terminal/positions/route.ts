import { NextRequest } from 'next/server';
import { proxyGetFrom, PERPETUAL_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  // Perpetual positions are scoped by user_id (query param).
  const userId = new URL(req.url).searchParams.get('user_id') || '';
  return proxyGetFrom(req, PERPETUAL_SERVICE_URL, `/api/v1/perpetual/users/${encodeURIComponent(userId)}/positions`);
}
