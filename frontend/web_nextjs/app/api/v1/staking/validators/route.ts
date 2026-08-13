import { NextRequest } from 'next/server';
import { proxyGetFrom, STAKING_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, STAKING_SERVICE_URL, '/api/v1/staking/validators');
}
