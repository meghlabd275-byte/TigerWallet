import { NextRequest } from 'next/server';
import { proxyGetFrom, ENS_SERVICE_URL } from '../../_proxy';

// Resolve an ENS name to an address. Pass ?name=foo.eth
export async function GET(req: NextRequest) {
  const name = new URL(req.url).searchParams.get('name') || '';
  return proxyGetFrom(req, ENS_SERVICE_URL, `/api/v1/ens/resolve/${encodeURIComponent(name)}`);
}
