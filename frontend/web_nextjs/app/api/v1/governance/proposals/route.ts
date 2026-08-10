import { NextRequest, NextResponse } from 'next/server';
import { proxyGetFrom, proxyMutationFrom, GOVERNANCE_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  const status = new URL(req.url).searchParams.get('status') || '';
  const qs = status ? `?status=${encodeURIComponent(status)}` : '';
  return proxyGetFrom(req, GOVERNANCE_SERVICE_URL, `/api/v1/governance/proposals${qs}`);
}

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, GOVERNANCE_SERVICE_URL, '/api/v1/governance/proposals', 'POST');
}
