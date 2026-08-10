import { NextRequest } from 'next/server';
import { serviceProxyGet, STAKING_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  const url = new URL(req.url);
  const userId = url.searchParams.get('user_id') || '';
  return serviceProxyGet(req, STAKING_SERVICE_URL, '/api/v1/staking/users/' + userId + '/positions');
}
