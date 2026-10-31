import { NextRequest } from 'next/server';
import { proxyMutationFrom, PERPETUAL_SERVICE_URL, encodePathSegment } from '../../../_proxy';

export async function POST(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutationFrom(req, PERPETUAL_SERVICE_URL, `/api/v1/perpetual/position/${encodePathSegment(params.id)}/close`, 'POST');
}
