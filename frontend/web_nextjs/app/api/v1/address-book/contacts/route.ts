import { NextRequest, NextResponse } from 'next/server';
import { proxyGet, proxyMutation } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGet(req, '/address-book/contacts');
}

export async function POST(req: NextRequest) {
  return proxyMutation(req, '/address-book/contacts', 'POST');
}
