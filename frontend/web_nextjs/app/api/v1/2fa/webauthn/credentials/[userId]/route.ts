import { NextRequest } from 'next/server';
import { proxyGetFrom, TWO_FACTOR_AUTH_URL } from '../../../../_proxy';

export async function GET(req: NextRequest) {
  const userId = req.nextUrl.pathname.split('/').pop();
  return proxyGetFrom(req, TWO_FACTOR_AUTH_URL, `/api/v1/2fa/webauthn/credentials/${userId}`);
}
