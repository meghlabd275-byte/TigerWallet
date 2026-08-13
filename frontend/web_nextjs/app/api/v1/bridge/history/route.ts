import { NextRequest } from 'next/server';
import { proxyGetFrom, BRIDGE_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, BRIDGE_SERVICE_URL, '/api/v1/bridge/history');
}
