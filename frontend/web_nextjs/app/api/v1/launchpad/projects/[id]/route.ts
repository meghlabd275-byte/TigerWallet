import { NextRequest, NextResponse } from 'next/server';
import { proxyGetFrom, LAUNCHPAD_SERVICE_URL } from '../../../_proxy';

export async function GET(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyGetFrom(req, LAUNCHPAD_SERVICE_URL, `/api/v1/launchpad/projects/${params.id}`);
}
