import { NextRequest, NextResponse } from 'next/server';
import { EARN_SERVICE_URL, proxyGetFrom } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, EARN_SERVICE_URL, '/api/v1/earn/deposits');
}
