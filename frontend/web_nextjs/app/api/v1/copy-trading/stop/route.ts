import { NextRequest, NextResponse } from 'next/server';
import { COPY_TRADING_SERVICE_URL, authHeaders } from '../../_proxy';

// Stop a single copier/position (when `positionId` is supplied) by calling the
// backend's per-id stop endpoint, or stop all of the caller's copiers when no
// id is given. The previous implementation always hit /stop-all, which ignored
// the request body and stopped every position even for a single-position stop.
export async function POST(req: NextRequest) {
  let body: string | undefined;
  let parsed: { positionId?: string; traderId?: string } = {};
  try {
    body = await req.text();
    parsed = body ? JSON.parse(body) : {};
  } catch {
    // Non-JSON or empty body: fall through to stop-all.
  }

  if (parsed.positionId) {
    try {
      const res = await fetch(
        `${COPY_TRADING_SERVICE_URL}/api/v1/copytrading/copiers/${encodeURIComponent(parsed.positionId)}/stop`,
        { method: 'POST', headers: authHeaders(req), body, cache: 'no-store' },
      );
      const data = await res.json().catch(() => ({}));
      return NextResponse.json(data, { status: res.status });
    } catch {
      return NextResponse.json({ success: false, error: 'Failed to stop position on backend' }, { status: 502 });
    }
  }

  // No specific position: stop all of the caller's copiers (also covers the
  // trader-unfollow case, which the backend models as stopping the copier rows).
  try {
    const res = await fetch(`${COPY_TRADING_SERVICE_URL}/api/v1/copytrading/stop-all`, {
      method: 'POST',
      headers: authHeaders(req),
      body: body ?? '{}',
      cache: 'no-store',
    });
    const data = await res.json().catch(() => ({}));
    return NextResponse.json(data, { status: res.status });
  } catch {
    return NextResponse.json({ success: false, error: 'Failed to stop-all on backend' }, { status: 502 });
  }
}
