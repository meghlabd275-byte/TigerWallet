import { NextRequest } from 'next/server';
import { proxyGetFrom, INSURANCE_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, INSURANCE_SERVICE_URL, '/api/v1/insurance/positions');
}
