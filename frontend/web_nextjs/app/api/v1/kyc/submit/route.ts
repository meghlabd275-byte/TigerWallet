import { NextRequest } from 'next/server';
import { proxyMutationFrom, LISTING_SERVICE_URL } from '../../_proxy';

// Submit a KYC document for verification. Forwards to the listing_service
// KYC document-upload endpoint (real AML/KYC pipeline).
export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, LISTING_SERVICE_URL, '/api/v1/kyc/document', 'POST');
}
