import { NextRequest, NextResponse } from 'next/server';
import { proxyGet } from '../../../_proxy';

export async function GET(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyGet(req, `/launchpad/projects/${params.id}`);
}
