import { NextRequest, NextResponse } from 'next/server';
import { proxyGetFrom, proxyMutationFrom, COPY_TRADING_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, COPY_TRADING_SERVICE_URL, '/api/v1/copytrading/traders');
}

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, COPY_TRADING_SERVICE_URL, '/api/v1/copytrading/follow', 'POST');
}
