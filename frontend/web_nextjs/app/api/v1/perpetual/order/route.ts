import { NextRequest } from 'next/server';
import { proxyMutationFrom, PERPETUAL_SERVICE_URL } from '../../_proxy';

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, PERPETUAL_SERVICE_URL, '/api/v1/perpetual/order', 'POST');
}
