import { NextRequest } from 'next/server';
import { proxyMutationFrom, BOT_API_SERVICE_URL } from '../../../_proxy';

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, BOT_API_SERVICE_URL, '/api/v1/auth/login', 'POST');
}
