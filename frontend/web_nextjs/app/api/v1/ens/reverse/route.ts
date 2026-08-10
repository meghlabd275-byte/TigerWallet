import { NextRequest } from 'next/server';
import { proxyGetFrom, ENS_SERVICE_URL } from '../../_proxy';

// Reverse-resolve an address to an ENS name. Pass ?address=0x...
export async function GET(req: NextRequest) {
  const address = new URL(req.url).searchParams.get('address') || '';
  return proxyGetFrom(req, ENS_SERVICE_URL, `/api/v1/ens/reverse/${encodeURIComponent(address)}`);
}
