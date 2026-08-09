import { NextRequest } from 'next/server';
import { proxyGet, proxyMutation } from '../_proxy';

export async function GET(req: NextRequest) {
  return proxyGet(req, '/wallets');
}

export async function POST(req: NextRequest) {
  return proxyMutation(req, '/wallets', 'POST');
}
