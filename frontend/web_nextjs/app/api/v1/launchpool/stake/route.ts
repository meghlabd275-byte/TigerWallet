import { NextRequest } from 'next/server';
import { proxyMutationFrom, LAUNCHPAD_SERVICE_URL } from '../../_proxy';

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, LAUNCHPAD_SERVICE_URL, '/api/v1/launchpad/participate', 'POST');
}
