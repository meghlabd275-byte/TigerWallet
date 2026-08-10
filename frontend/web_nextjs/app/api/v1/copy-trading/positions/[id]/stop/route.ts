import { NextRequest } from 'next/server';
import { proxyMutationFrom, COPY_TRADING_SERVICE_URL } from '../../../../_proxy';

export async function POST(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutationFrom(req, COPY_TRADING_SERVICE_URL, `/api/v1/copytrading/copiers/${params.id}/stop`, 'POST');
}
