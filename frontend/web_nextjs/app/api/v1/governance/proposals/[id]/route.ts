import { NextRequest, NextResponse } from 'next/server';
import { proxyGetFrom, GOVERNANCE_SERVICE_URL } from '../../../_proxy';

export async function GET(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyGetFrom(req, GOVERNANCE_SERVICE_URL, `/api/v1/governance/proposals/${encodeURIComponent(params.id)}`);
}
