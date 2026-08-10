// Perpetual Markets API Route — proxies to go/perpetual_service /api/v1/perpetual/pairs.
import { NextRequest, NextResponse } from 'next/server';
import { proxyGetFrom, PERPETUAL_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, PERPETUAL_SERVICE_URL, '/api/v1/perpetual/pairs');
}
