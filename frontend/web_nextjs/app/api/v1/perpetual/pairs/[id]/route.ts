import { NextRequest } from 'next/server';
import { proxyGetFrom, PERPETUAL_SERVICE_URL } from '../../../_proxy';

export async function GET(req: NextRequest) {
  const id = req.nextUrl.pathname.split('/').pop();
  return proxyGetFrom(req, PERPETUAL_SERVICE_URL, `/api/v1/perpetual/pairs/${id}`);
}
