import { NextRequest, NextResponse } from 'next/server';
import { proxyMutation } from '../../../../_proxy';

export async function POST(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutation(req, `/copy-trading/positions/${params.id}/stop`, 'POST');
}
