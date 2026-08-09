import { NextRequest } from 'next/server';
import { proxyGet, proxyMutation } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGet(req, '/terminal/orders');
}

export async function POST(req: NextRequest) {
  return proxyMutation(req, '/terminal/orders', 'POST');
}
