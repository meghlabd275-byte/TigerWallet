import { NextRequest, NextResponse } from 'next/server';
import { proxyGet, proxyMutation } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGet(req, '/fiat/orders');
}

export async function POST(req: NextRequest) {
  return proxyMutation(req, '/fiat/orders', 'POST');
}
