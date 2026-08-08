import { NextRequest } from 'next/server';
import { proxyMutation } from '../../../../_proxy';

export async function PUT(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutation(req, `/admin/fees/${params.id}`, 'PUT');
}

export async function DELETE(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutation(req, `/admin/fees/${params.id}`, 'DELETE');
}
