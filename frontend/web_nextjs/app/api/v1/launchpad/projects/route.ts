import { NextRequest, NextResponse } from 'next/server';
import { proxyGetFrom, proxyMutationFrom, LAUNCHPAD_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, LAUNCHPAD_SERVICE_URL, '/api/v1/launchpad/projects');
}

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, LAUNCHPAD_SERVICE_URL, '/api/v1/launchpad/projects', 'POST');
}
