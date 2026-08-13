import { NextRequest, NextResponse } from 'next/server';
import { proxyGetFrom, proxyMutationFrom, BUG_BOUNTY_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, BUG_BOUNTY_SERVICE_URL, '/api/v1/reports');
}

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, BUG_BOUNTY_SERVICE_URL, '/api/v1/reports', 'POST');
}
