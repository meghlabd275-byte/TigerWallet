import { NextRequest } from 'next/server';
import { proxyMutationFrom, PERPETUAL_SERVICE_URL } from '../../../../_proxy';

export async function POST(req: NextRequest) {
  const id = req.nextUrl.pathname.split('/').slice(-3, -2)[0];
  return proxyMutationFrom(req, PERPETUAL_SERVICE_URL, `/api/v1/perpetual/position/${id}/close`, 'POST');
}
