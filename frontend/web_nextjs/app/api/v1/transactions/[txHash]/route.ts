import { NextRequest } from 'next/server';
import { proxyGet } from '../../_proxy';

// GET /api/v1/transactions/:txHash?chain_id=N -> wallet_api
export async function GET(
  req: NextRequest,
  { params }: { params: { txHash: string } }
) {
  return proxyGet(req, `/transactions/${encodeURIComponent(params.txHash)}`);
}
