import { NextRequest } from 'next/server';
import { proxyMutationFrom, BOTS_SERVICE_URL } from '../../../_proxy';

export async function DELETE(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutationFrom(req, BOTS_SERVICE_URL, `/api/v1/bots/${params.id}`, 'DELETE');
}
