import { NextRequest, NextResponse } from 'next/server';
import { proxyMutationFrom, LAUNCHPAD_SERVICE_URL, encodePathSegment } from '../../../../_proxy';

export async function POST(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutationFrom(req, LAUNCHPAD_SERVICE_URL, `/api/v1/launchpad/allocations/${encodePathSegment(params.id)}/claim`, 'POST');
}
