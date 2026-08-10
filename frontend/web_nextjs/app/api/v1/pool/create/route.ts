import { NextRequest } from 'next/server';
import { proxyMutationFrom, POOL_SERVICE_URL } from '../../_proxy';

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, POOL_SERVICE_URL, '/api/v1/pool/create', 'POST');
}
