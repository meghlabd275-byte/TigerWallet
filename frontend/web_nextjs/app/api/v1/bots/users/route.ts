import { NextRequest } from 'next/server';
import { proxyGetFrom, proxyMutationFrom, BOT_API_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, BOT_API_SERVICE_URL, '/api/v1/bots/users');
}

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, BOT_API_SERVICE_URL, '/api/v1/bots/users', 'POST');
}
