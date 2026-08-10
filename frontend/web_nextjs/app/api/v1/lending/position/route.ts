import { NextRequest } from 'next/server';
import { serviceProxyGet, LENDING_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  const url = new URL(req.url);
  const user = url.searchParams.get('user_address') || '';
  const chain = url.searchParams.get('chain_id') || '1';
  return serviceProxyGet(req, LENDING_SERVICE_URL, '/lending/position?user_address=' + user + '&chain_id=' + chain);
}
