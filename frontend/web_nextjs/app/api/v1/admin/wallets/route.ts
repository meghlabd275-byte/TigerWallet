import { NextRequest } from 'next/server';
import { proxyGet, proxyMutation } from '../../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGet(req, '/admin/wallets');
}

export async function POST(req: NextRequest) {
  return proxyMutation(req, '/admin/wallets', 'POST');
}
