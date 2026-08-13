import { NextRequest } from 'next/server';
import { proxyGetFrom, PROTECTION_FUND_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, PROTECTION_FUND_SERVICE_URL, '/api/v1/funds');
}
