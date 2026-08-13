import { NextRequest } from 'next/server';
import { swapProxyGet } from '../../../_proxy';

export async function GET(req: NextRequest) {
  const id = req.nextUrl.pathname.split('/').pop();
  return swapProxyGet(req, `/swap/pairs/${id}`);
}
