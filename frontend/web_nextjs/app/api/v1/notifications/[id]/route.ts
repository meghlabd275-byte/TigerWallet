import { NextRequest } from 'next/server';
import { proxyMutationFrom, NOTIFICATION_SERVICE_URL } from '../../_proxy';

export async function DELETE(req: NextRequest) {
  const id = req.nextUrl.pathname.split('/').pop();
  return proxyMutationFrom(req, NOTIFICATION_SERVICE_URL, `/api/v1/notifications/${id}`, 'DELETE');
}
