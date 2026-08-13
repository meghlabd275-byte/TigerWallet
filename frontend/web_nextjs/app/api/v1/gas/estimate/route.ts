import { NextRequest } from 'next/server';
import { proxyGet, BACKEND_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGet(req, '/gas');
}
