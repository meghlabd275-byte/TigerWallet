import { NextRequest } from 'next/server';
import { proxyGetFrom, proxyMutationFrom, PREDICTION_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, PREDICTION_SERVICE_URL, '/api/v1/prediction/markets');
}

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, PREDICTION_SERVICE_URL, '/api/v1/prediction/markets', 'POST');
}
