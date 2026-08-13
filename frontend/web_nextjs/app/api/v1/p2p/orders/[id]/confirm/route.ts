import { NextRequest } from 'next/server';
import { proxyMutationFrom, P2P_SERVICE_URL } from '../../../../_proxy';

export async function POST(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutationFrom(req, P2P_SERVICE_URL, `/api/v1/p2p/orders/${params.id}/confirm`, 'POST');
}
