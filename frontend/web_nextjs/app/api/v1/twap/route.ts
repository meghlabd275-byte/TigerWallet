import { NextRequest } from 'next/server';
import { proxyGet, proxyMutation } from '../_proxy';

export async function GET(req: NextRequest) {
  return proxyGet(req, '/twap');
}

export async function POST(req: NextRequest) {
  return proxyMutation(req, '/twap', 'POST');
}
