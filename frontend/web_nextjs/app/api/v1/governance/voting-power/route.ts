import { NextRequest, NextResponse } from 'next/server';
import { proxyGetFrom, GOVERNANCE_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  const user = new URL(req.url).searchParams.get('user') || '';
  return proxyGetFrom(req, GOVERNANCE_SERVICE_URL, `/api/v1/governance/voting-power/${encodeURIComponent(user)}`);
}
