import { NextRequest } from 'next/server';
import { RED_PACKETS_SERVICE_URL, proxyMutationFrom } from '../../_proxy';

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, RED_PACKETS_SERVICE_URL, '/api/v1/red-packets/claim', 'POST');
}
