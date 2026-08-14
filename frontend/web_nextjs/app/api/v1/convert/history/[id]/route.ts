import { NextRequest } from 'next/server';
import { serviceProxyGet, serviceProxyMutation, CONVERT_SERVICE_URL } from '../../../_proxy';

export async function GET(req: NextRequest) {
  const id = req.nextUrl.pathname.split('/').pop();
  return serviceProxyGet(req, CONVERT_SERVICE_URL, `/convert/history/${id}`);
}

export async function PATCH(req: NextRequest) {
  const id = req.nextUrl.pathname.split('/').pop();
  return serviceProxyMutation(req, CONVERT_SERVICE_URL, `/convert/history/${id}`, 'PATCH');
}
