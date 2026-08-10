import { NextRequest } from 'next/server';
import { serviceProxyMutation, LISTING_SERVICE_URL } from '../../_proxy';

export async function POST(req: NextRequest) {
  return serviceProxyMutation(req, LISTING_SERVICE_URL, '/listing/apply', 'POST');
}
