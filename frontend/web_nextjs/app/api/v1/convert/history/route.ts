import { NextRequest } from 'next/server';
import { serviceProxyGet, CONVERT_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return serviceProxyGet(req, CONVERT_SERVICE_URL, '/convert/history');
}
