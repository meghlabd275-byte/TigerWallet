import { NextRequest } from 'next/server';
import { COUPON_SERVICE_URL, proxyMutationFrom } from '../../_proxy';

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, COUPON_SERVICE_URL, '/api/v1/coupon/validate', 'POST');
}
