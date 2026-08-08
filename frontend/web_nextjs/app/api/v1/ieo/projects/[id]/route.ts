import { NextRequest, NextResponse } from 'next/server';
import { proxyGet, proxyMutation } from '../../../_proxy';

export async function GET(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyGet(req, `/ieo/projects/${params.id}`);
}

export async function POST(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutation(req, `/ieo/projects/${params.id}/participate`, 'POST');
}
