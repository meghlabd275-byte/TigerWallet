import { NextRequest } from 'next/server';
import {
  proxyGetFrom,
  proxyMutationFrom,
  FIAT_ONRAMP_URL,
} from '../../../../../_proxy';

// GET /api/v1/ramp/admin/providers/:id/key -> { providerId, configured }
export async function GET(
  req: NextRequest,
  { params }: { params: { id: string } }
) {
  return proxyGetFrom(req, FIAT_ONRAMP_URL, `/api/v1/ramp/admin/providers/${params.id}/key`);
}

// POST /api/v1/ramp/admin/providers/:id/key { apiKey } -> set the key
export async function POST(
  req: NextRequest,
  { params }: { params: { id: string } }
) {
  return proxyMutationFrom(req, FIAT_ONRAMP_URL, `/api/v1/ramp/admin/providers/${params.id}/key`, 'POST');
}

// DELETE /api/v1/ramp/admin/providers/:id/key -> clear the key (fall back to env)
export async function DELETE(
  req: NextRequest,
  { params }: { params: { id: string } }
) {
  return proxyMutationFrom(req, FIAT_ONRAMP_URL, `/api/v1/ramp/admin/providers/${params.id}/key`, 'DELETE');
}
