import { NextRequest } from 'next/server';
import { proxyGetFrom, BOTS_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, BOTS_SERVICE_URL, '/api/v1/bots');
}
