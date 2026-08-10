import { NextRequest } from 'next/server';
import { proxyGetFrom, proxyMutationFrom, TWAP_SERVICE_URL } from '../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, TWAP_SERVICE_URL, '/api/v1/twap');
}

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, TWAP_SERVICE_URL, '/api/v1/twap', 'POST');
}
