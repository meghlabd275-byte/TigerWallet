import { NextRequest } from 'next/server';
import { proxyGet, proxyMutation } from '../../../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGet(req, '/admin/chains/validators');
}

export async function POST(req: NextRequest) {
  return proxyMutation(req, '/admin/chains/validators', 'POST');
}
