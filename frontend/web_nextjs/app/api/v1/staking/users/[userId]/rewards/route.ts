import { NextRequest } from 'next/server';
import { proxyGetFrom, STAKING_SERVICE_URL } from '../../../../_proxy';

export async function GET(req: NextRequest) {
  const userId = req.nextUrl.pathname.split('/').pop();
  return proxyGetFrom(req, STAKING_SERVICE_URL, `/api/v1/staking/users/${userId}/rewards`);
}
