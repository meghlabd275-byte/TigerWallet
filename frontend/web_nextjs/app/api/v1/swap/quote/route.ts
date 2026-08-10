import { NextRequest } from 'next/server';
import { swapProxyGet } from '../../_proxy';

export async function GET(req: NextRequest) {
  return swapProxyGet(req, '/swap/quote');
}
