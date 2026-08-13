import { NextRequest } from 'next/server';
import { proxyGetFrom, COPY_TRADING_SERVICE_URL } from '../../../_proxy';

export async function GET(req: NextRequest) {
  const id = req.nextUrl.pathname.split('/').pop();
  return proxyGetFrom(req, COPY_TRADING_SERVICE_URL, `/api/v1/copytrading/traders/${id}`);
}
