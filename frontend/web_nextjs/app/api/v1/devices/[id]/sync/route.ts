import { NextRequest, NextResponse } from 'next/server';
import { proxyMutation, encodePathSegment } from '../../../_proxy';

export async function POST(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutation(req, `/devices/${encodePathSegment(params.id)}/sync`, 'POST');
}
