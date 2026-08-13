import { NextRequest } from 'next/server';
import { proxyMutationFrom, STAKING_SERVICE_URL } from '../../_proxy';

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, STAKING_SERVICE_URL, '/api/v1/staking/convert', 'POST');
}
