import { NextRequest } from 'next/server';
import { proxyMutationFrom, BOT_API_SERVICE_URL } from '../../_proxy';

export async function DELETE(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutationFrom(req, BOT_API_SERVICE_URL, `/api/v1/bots/${params.id}`, 'DELETE');
}
