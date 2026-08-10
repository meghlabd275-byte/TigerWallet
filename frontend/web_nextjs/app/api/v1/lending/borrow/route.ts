import { NextRequest } from 'next/server';
import { serviceProxyMutation, LENDING_SERVICE_URL } from '../../_proxy';

export async function POST(req: NextRequest) {
  return serviceProxyMutation(req, LENDING_SERVICE_URL, '/lending/borrow', 'POST');
}
