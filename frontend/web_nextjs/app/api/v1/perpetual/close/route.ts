import { NextRequest } from 'next/server';
import { PERPETUAL_SERVICE_URL, proxyMutationFrom } from '../../_proxy';

// Reads positionId from the body, then forwards to /perpetual/positions/:id/close
export async function POST(req: NextRequest) {
  const body = await req.json();
  const positionId = body.positionId;
  if (!positionId) {
    return Response.json({ error: 'positionId is required' }, { status: 400 });
  }
  const { positionId: _, ...rest } = body;
  return proxyMutationFrom(
    new NextRequest(req, { body: JSON.stringify(rest) }),
    PERPETUAL_SERVICE_URL,
    `/api/v1/perpetual/position/${positionId}/close`,
    'POST'
  );
}
