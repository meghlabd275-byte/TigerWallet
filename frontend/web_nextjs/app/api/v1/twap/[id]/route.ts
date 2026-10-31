import { NextRequest } from 'next/server';
import { proxyMutationFrom, TWAP_SERVICE_URL, encodePathSegment } from '../../_proxy';

export async function PUT(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutationFrom(req, TWAP_SERVICE_URL, `/api/v1/twap/${encodePathSegment(params.id)}/cancel`, 'PUT');
}

export async function DELETE(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutationFrom(req, TWAP_SERVICE_URL, `/api/v1/twap/${encodePathSegment(params.id)}/cancel`, 'DELETE');
}
