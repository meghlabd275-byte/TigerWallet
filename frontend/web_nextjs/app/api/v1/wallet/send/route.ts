import { NextRequest } from 'next/server';
import { proxyMutation } from '../../_proxy';

export async function POST(req: NextRequest) {
  return proxyMutation(req, '/send', 'POST');
}
