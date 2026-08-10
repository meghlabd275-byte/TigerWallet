import { NextRequest } from 'next/server';
import { proxyGetFrom, IEO_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, IEO_SERVICE_URL, '/api/v1/ieo/rounds');
}
