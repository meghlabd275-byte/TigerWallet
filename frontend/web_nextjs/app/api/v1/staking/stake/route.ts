import { NextRequest } from 'next/server';
import { serviceProxyMutation, STAKING_SERVICE_URL } from '../../_proxy';

export async function POST(req: NextRequest) {
  return serviceProxyMutation(req, STAKING_SERVICE_URL, '/staking/stake', 'POST');
}
