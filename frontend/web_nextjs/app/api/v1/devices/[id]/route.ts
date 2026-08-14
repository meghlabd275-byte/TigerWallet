import { NextRequest, NextResponse } from 'next/server';
import { proxyMutation } from '../../_proxy';

export async function DELETE(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutation(req, `/devices/${params.id}`, 'DELETE');
}
