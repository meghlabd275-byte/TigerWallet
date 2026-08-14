import { NextRequest } from 'next/server';
import { proxyGetFrom, BOT_API_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, BOT_API_SERVICE_URL, '/api/v1/bots/me');
}
