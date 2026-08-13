import { NextRequest } from 'next/server';
import { EARN_SERVICE_URL, proxyMutationFrom } from '../../_proxy';

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, EARN_SERVICE_URL, '/api/v1/earn/deposit', 'POST');
}
