import { NextRequest } from 'next/server';
import { proxyGetFrom, LAUNCHPAD_SERVICE_URL } from '../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, LAUNCHPAD_SERVICE_URL, '/api/v1/launchpad/projects');
}
