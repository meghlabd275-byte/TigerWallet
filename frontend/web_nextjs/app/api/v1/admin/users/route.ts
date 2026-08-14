import { NextRequest } from 'next/server';
import { proxyGet } from '../../_proxy';

// GET /api/v1/admin/users -> wallet_api (admin-role protected) — real user list
// from PostgreSQL (users table with per-user wallet/trade aggregates).
export async function GET(req: NextRequest) {
  return proxyGet(req, '/admin/users');
}
