import { NextRequest } from 'next/server';
import { proxyGetFrom, proxyMutationFrom, PERPETUAL_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, PERPETUAL_SERVICE_URL, '/api/v1/perpetual/pairs');
}

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, PERPETUAL_SERVICE_URL, '/api/v1/perpetual/pairs', 'POST');
}
