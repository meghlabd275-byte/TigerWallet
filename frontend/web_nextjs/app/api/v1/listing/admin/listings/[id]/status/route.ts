import { NextRequest } from 'next/server';
import { proxyMutationFrom, LISTING_SERVICE_URL } from '../../../../../_proxy';

export async function PUT(req: NextRequest) {
  const id = req.nextUrl.pathname.split('/').slice(-2, -1)[0];
  return proxyMutationFrom(req, LISTING_SERVICE_URL, `/api/v1/listing/admin/listings/${id}/status`, 'PUT');
}
