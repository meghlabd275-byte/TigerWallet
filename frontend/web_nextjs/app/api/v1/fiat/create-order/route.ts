import { NextRequest } from 'next/server';
import { proxyMutationFrom, FIAT_SERVICE_URL } from '../../_proxy';

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, FIAT_SERVICE_URL, '/api/v1/fiat/orders', 'POST');
}
