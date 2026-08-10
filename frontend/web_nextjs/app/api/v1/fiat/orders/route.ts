import { NextRequest } from 'next/server';
import { proxyGetFrom, proxyMutationFrom, FIAT_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, FIAT_SERVICE_URL, '/api/v1/fiat/orders');
}

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, FIAT_SERVICE_URL, '/api/v1/fiat/orders', 'POST');
}
