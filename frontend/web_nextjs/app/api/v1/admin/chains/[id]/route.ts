import { NextRequest } from 'next/server';
import { proxyMutation, encodePathSegment } from '../../../_proxy';

export async function PUT(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutation(req, `/admin/chains/${encodePathSegment(params.id)}`, 'PUT');
}

export async function DELETE(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutation(req, `/admin/chains/${encodePathSegment(params.id)}`, 'DELETE');
}
