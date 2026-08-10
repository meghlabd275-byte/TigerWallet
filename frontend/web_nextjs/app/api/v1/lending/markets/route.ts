import { NextRequest } from 'next/server';
import { serviceProxyGet, LENDING_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return serviceProxyGet(req, LENDING_SERVICE_URL, '/lending/markets');
}
