import { NextRequest } from 'next/server';
import { swapProxyGet } from '../../../../_proxy';

export async function GET(req: NextRequest) {
  const userId = req.nextUrl.pathname.split('/').pop();
  return swapProxyGet(req, `/users/${userId}/swaps`);
}
