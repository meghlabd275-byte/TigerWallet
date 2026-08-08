import { NextRequest, NextResponse } from 'next/server';
import { proxyGet, proxyMutation } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGet(req, '/copy-trading/traders');
}

export async function POST(req: NextRequest) {
  return proxyMutation(req, '/copy-trading/start', 'POST');
}
