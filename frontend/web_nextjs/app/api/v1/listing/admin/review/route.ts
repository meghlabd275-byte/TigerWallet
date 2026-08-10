import { NextRequest, NextResponse } from 'next/server';
import { LISTING_SERVICE_URL, authHeaders } from '../../../_proxy';

export async function PUT(req: NextRequest) {
  const url = new URL(req.url);
  const requestId = url.searchParams.get('request_id') || '';
  const backend = new URL(LISTING_SERVICE_URL);
  backend.pathname = '/api/v1/listing/admin/listings/' + requestId + '/review';
  try {
    const body = await req.text();
    const r = await fetch(backend.toString(), {
      method: 'PUT',
      headers: authHeaders(req),
      body: body || undefined,
    });
    const text = await r.text();
    return NextResponse.json(text ? JSON.parse(text) : {}, { status: r.status });
  } catch (e) {
    return NextResponse.json({ error: 'listing service unavailable' }, { status: 502 });
  }
}
