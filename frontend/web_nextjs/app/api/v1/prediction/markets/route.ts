import { NextRequest, NextResponse } from 'next/server';
import { proxyGetFrom, PREDICTION_SERVICE_URL } from '../../_proxy';

// List prediction markets (optional ?status=open|resolved). No fallback data:
// if the backend is unreachable the route returns a 502 error, never mock data.
export async function GET(req: NextRequest) {
  const status = new URL(req.url).searchParams.get('status') || '';
  const qs = status ? `?status=${encodeURIComponent(status)}` : '';
  return proxyGetFrom(req, PREDICTION_SERVICE_URL, `/api/v1/prediction/markets${qs}`);
}

