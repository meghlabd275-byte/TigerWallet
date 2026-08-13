import { NextRequest } from 'next/server';
import { proxyMutationFrom, NOTIFICATION_SERVICE_URL } from '../../../_proxy';

export async function PUT(req: NextRequest) {
  const id = req.nextUrl.pathname.split('/').slice(-2, -1)[0];
  return proxyMutationFrom(req, NOTIFICATION_SERVICE_URL, `/api/v1/notifications/${id}/read`, 'PUT');
}
