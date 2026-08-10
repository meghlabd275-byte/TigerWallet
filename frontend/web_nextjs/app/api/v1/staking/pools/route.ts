import { NextRequest } from 'next/server';
import { serviceProxyGet, STAKING_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return serviceProxyGet(req, STAKING_SERVICE_URL, '/staking/pools');
}
