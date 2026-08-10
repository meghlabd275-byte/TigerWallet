import { NextRequest } from 'next/server';
import { proxyGetFrom, LISTING_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  // The KYC status endpoint expects a user_id query param.
  const userId = new URL(req.url).searchParams.get('user_id') || '';
  return proxyGetFrom(req, LISTING_SERVICE_URL, `/api/v1/kyc/status/${encodeURIComponent(userId)}`);
}
