import { NextRequest } from 'next/server';
import { proxyMutationFrom, MATCHING_ENGINE_URL } from '../../../_proxy';

export async function DELETE(req: NextRequest, { params }: { params: { id: string } }) {
  // The matching engine cancels by order id via POST /cancel with {order_id}.
  return proxyMutationFrom(req, MATCHING_ENGINE_URL, '/cancel', 'POST');
}
