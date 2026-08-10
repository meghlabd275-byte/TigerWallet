import { NextRequest } from 'next/server';
import { proxyGetFrom, proxyMutationFrom, PERPETUAL_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  // List a user's perpetual orders via their positions endpoint.
  const userId = new URL(req.url).searchParams.get('user_id') || '';
  return proxyGetFrom(req, PERPETUAL_SERVICE_URL, `/api/v1/perpetual/users/${encodeURIComponent(userId)}/positions`);
}

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, PERPETUAL_SERVICE_URL, '/api/v1/perpetual/order', 'POST');
}
