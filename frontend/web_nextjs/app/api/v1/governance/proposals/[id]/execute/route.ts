import { NextRequest, NextResponse } from 'next/server';
import { proxyMutationFrom, GOVERNANCE_SERVICE_URL } from '../../../../_proxy';

export async function POST(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutationFrom(req, GOVERNANCE_SERVICE_URL, `/api/v1/governance/proposals/${encodeURIComponent(params.id)}/execute`, 'POST');
}
