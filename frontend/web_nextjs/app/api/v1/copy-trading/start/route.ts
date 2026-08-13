import { NextRequest } from 'next/server';
import { COPY_TRADING_SERVICE_URL, proxyMutationFrom } from '../../_proxy';

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, COPY_TRADING_SERVICE_URL, '/api/v1/copytrading/follow', 'POST');
}
