import { NextRequest } from 'next/server';
import { proxyGetFrom, PREDICTION_SERVICE_URL } from '../../../../_proxy';

export async function GET(req: NextRequest) {
  const id = req.nextUrl.pathname.split('/').slice(-3, -2)[0];
  return proxyGetFrom(req, PREDICTION_SERVICE_URL, `/api/v1/prediction/markets/${id}/bets`);
}
