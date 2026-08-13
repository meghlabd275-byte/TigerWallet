import { NextRequest } from 'next/server';
import { proxyMutationFrom, PREDICTION_SERVICE_URL } from '../../../../_proxy';

export async function POST(req: NextRequest) {
  const id = req.nextUrl.pathname.split('/').slice(-3, -2)[0];
  return proxyMutationFrom(req, PREDICTION_SERVICE_URL, `/api/v1/prediction/markets/${id}/bet`, 'POST');
}
