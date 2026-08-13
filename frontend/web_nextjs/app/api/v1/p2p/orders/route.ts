import { NextRequest } from 'next/server';
import { proxyGetFrom, proxyMutationFrom, P2P_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, P2P_SERVICE_URL, '/api/v1/p2p/orders');
}

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, P2P_SERVICE_URL, '/api/v1/p2p/orders', 'POST');
}
