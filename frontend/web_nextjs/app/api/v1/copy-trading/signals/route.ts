import { NextRequest, NextResponse } from 'next/server';
import { proxyGet } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGet(req, '/copy-trading/signals');
}
