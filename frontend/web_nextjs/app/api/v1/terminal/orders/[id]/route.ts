import { NextRequest } from 'next/server';
import { proxyMutation } from '../../../../../_proxy';

export async function DELETE(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutation(req, `/terminal/orders/${params.id}`, 'DELETE');
}
