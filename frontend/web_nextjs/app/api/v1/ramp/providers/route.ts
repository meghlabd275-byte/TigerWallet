import { NextRequest } from 'next/server';
import { proxyGetFrom, FIAT_ONRAMP_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, FIAT_ONRAMP_URL, '/api/v1/ramp/providers');
}
