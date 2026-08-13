import { NextRequest } from 'next/server';
import { proxyMutationFrom, COPY_TRADING_SERVICE_URL } from '../../../../_proxy';

export async function POST(req: NextRequest) {
  const id = req.nextUrl.pathname.split('/').slice(-3, -2)[0];
  return proxyMutationFrom(req, COPY_TRADING_SERVICE_URL, `/api/v1/copytrading/copiers/${id}/stop`, 'POST');
}
