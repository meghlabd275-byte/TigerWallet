import { NextRequest } from 'next/server';
import { proxyGetFrom, PORTFOLIO_TRACKER_URL } from '../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, PORTFOLIO_TRACKER_URL, '/api/portfolio');
}
