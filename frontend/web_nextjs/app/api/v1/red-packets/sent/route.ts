import { NextRequest, NextResponse } from 'next/server';
import { RED_PACKETS_SERVICE_URL, proxyGetFrom } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, RED_PACKETS_SERVICE_URL, '/api/v1/red-packets/sent');
}
