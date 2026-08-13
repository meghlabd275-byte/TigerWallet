import { NextRequest } from 'next/server';
import { proxyGetFrom, FIAT_ONRAMP_URL } from '../../../_proxy';

export async function GET(req: NextRequest) {
  const userId = req.nextUrl.pathname.split('/').pop();
  return proxyGetFrom(req, FIAT_ONRAMP_URL, `/api/v1/ramp/orders/${userId}`);
}
