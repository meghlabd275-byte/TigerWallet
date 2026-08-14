import { NextRequest } from 'next/server';
import { proxyMutationFrom, FIAT_ONRAMP_URL } from '../../_proxy';

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, FIAT_ONRAMP_URL, '/api/v1/ramp/offramp-quote', 'POST');
}
