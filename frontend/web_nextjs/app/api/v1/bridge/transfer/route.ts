import { NextRequest } from 'next/server';
import { proxyMutationFrom, BRIDGE_SERVICE_URL } from '../../_proxy';

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, BRIDGE_SERVICE_URL, '/api/v1/bridge/transfer', 'POST');
}
