import { NextRequest } from 'next/server';
import { proxyMutationFrom, COPY_TRADING_SERVICE_URL } from '../../_proxy';

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, COPY_TRADING_SERVICE_URL, '/api/v1/copytrading/stop-all', 'POST');
}
