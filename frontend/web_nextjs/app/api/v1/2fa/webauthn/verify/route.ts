import { NextRequest } from 'next/server';
import { proxyMutationFrom, TWO_FACTOR_AUTH_URL } from '../../../_proxy';

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, TWO_FACTOR_AUTH_URL, '/api/v1/2fa/webauthn/verify', 'POST');
}
