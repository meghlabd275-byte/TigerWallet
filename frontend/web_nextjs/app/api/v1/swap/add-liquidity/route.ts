import { NextRequest } from 'next/server';
import { swapProxyMutation } from '../../_proxy';

export async function POST(req: NextRequest) {
  return swapProxyMutation(req, '/add-liquidity', 'POST');
}
