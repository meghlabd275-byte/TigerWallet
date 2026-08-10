import { NextRequest } from 'next/server';
import { proxyGetFrom, proxyMutationFrom, INSURANCE_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, INSURANCE_SERVICE_URL, '/api/v1/insurance/claims');
}

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, INSURANCE_SERVICE_URL, '/api/v1/insurance/claims', 'POST');
}
