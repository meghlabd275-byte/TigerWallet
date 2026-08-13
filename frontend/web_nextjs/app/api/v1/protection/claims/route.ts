import { NextRequest } from 'next/server';
import { proxyGetFrom, proxyMutationFrom, PROTECTION_FUND_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, PROTECTION_FUND_SERVICE_URL, '/api/v1/claims');
}

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, PROTECTION_FUND_SERVICE_URL, '/api/v1/claims', 'POST');
}
