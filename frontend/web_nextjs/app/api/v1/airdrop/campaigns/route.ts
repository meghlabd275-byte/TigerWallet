import { NextRequest } from 'next/server';
import { AIRDROP_SERVICE_URL, proxyGetFrom } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, AIRDROP_SERVICE_URL, '/api/v1/airdrop/campaigns');
}
