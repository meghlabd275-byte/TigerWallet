import { NextRequest } from 'next/server';
import { proxyGetFrom, BRIDGE_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  // Bridge transfer status — expects a tx id query param.
  const id = new URL(req.url).searchParams.get('id') || '';
  return proxyGetFrom(req, BRIDGE_SERVICE_URL, `/api/v1/bridge/tx/${encodeURIComponent(id)}`);
}
