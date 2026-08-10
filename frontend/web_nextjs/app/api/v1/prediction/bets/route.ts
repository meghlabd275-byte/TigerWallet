import { NextRequest, NextResponse } from 'next/server';
import { PREDICTION_SERVICE_URL } from '../../_proxy';

// List the caller's prediction positions. GET expects ?user=<id>.
export async function GET(req: NextRequest) {
  const user = new URL(req.url).searchParams.get('user') || '';
  if (!user) {
    return NextResponse.json({ error: 'user query param required' }, { status: 400 });
  }
  try {
    const res = await fetch(
      `${PREDICTION_SERVICE_URL}/api/v1/prediction/positions/${encodeURIComponent(user)}`,
      { cache: 'no-store' }
    );
    const data = await res.json();
    return NextResponse.json(data, { status: res.status });
  } catch {
    return NextResponse.json({ error: 'Failed to fetch positions from backend' }, { status: 502 });
  }
}

// Place a bet. Body must include { market_id, side, amount }. The route reads
// the body, extracts market_id, and forwards to the backend market bet
// endpoint. No fake bet_id fallback.
export async function POST(req: NextRequest) {
  let body: { market_id?: string; side?: number; amount?: string };
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ error: 'invalid JSON body' }, { status: 400 });
  }
  if (!body.market_id) {
    return NextResponse.json({ error: 'market_id required' }, { status: 400 });
  }
  try {
    const res = await fetch(
      `${PREDICTION_SERVICE_URL}/api/v1/prediction/markets/${encodeURIComponent(body.market_id)}/bet`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authForward(req) },
        body: JSON.stringify({ side: body.side, amount: body.amount }),
        cache: 'no-store',
      }
    );
    const data = await res.json();
    return NextResponse.json(data, { status: res.status });
  } catch {
    return NextResponse.json({ error: 'Failed to place bet on backend' }, { status: 502 });
  }
}

function authForward(req: NextRequest): Record<string, string> {
  const h = req.headers.get('authorization');
  return h ? { Authorization: h } : {};
}
