import { NextRequest } from 'next/server';
import { PERPETUAL_SERVICE_URL, proxyMutationFrom } from '../../_proxy';

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, PERPETUAL_SERVICE_URL, '/api/v1/perpetual/position', 'POST');
}
