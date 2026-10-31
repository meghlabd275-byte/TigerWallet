import { NextRequest } from 'next/server';
import { proxyMutation, encodePathSegment } from '../../../../_proxy';

// PUT /api/v1/admin/users/:id/role -> wallet_api (admin-role protected)
export async function PUT(
  req: NextRequest,
  { params }: { params: { id: string } }
) {
  return proxyMutation(req, `/admin/users/${encodePathSegment(params.id)}/role`, 'PUT');
}
