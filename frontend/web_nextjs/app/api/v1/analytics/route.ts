import { NextRequest } from 'next/server';
import { proxyGetFrom, ANALYTICS_SERVICE_ALT_URL } from '../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, ANALYTICS_SERVICE_ALT_URL, '/api/v1/analytics/overview');
}
