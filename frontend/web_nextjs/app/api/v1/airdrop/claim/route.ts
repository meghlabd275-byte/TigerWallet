import { NextRequest } from 'next/server';
import { AIRDROP_SERVICE_URL, proxyMutationFrom } from '../../_proxy';

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, AIRDROP_SERVICE_URL, '/api/v1/airdrop/claim', 'POST');
}
