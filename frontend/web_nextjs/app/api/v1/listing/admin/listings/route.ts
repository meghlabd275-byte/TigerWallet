import { NextRequest } from 'next/server';
import { serviceProxyGet, LISTING_SERVICE_URL } from '../../../_proxy';

export async function GET(req: NextRequest) {
  return serviceProxyGet(req, LISTING_SERVICE_URL, '/listing/admin/listings');
}
