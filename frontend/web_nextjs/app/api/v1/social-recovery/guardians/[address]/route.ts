import { NextRequest } from 'next/server';
import { proxyMutationFrom, SOCIAL_RECOVERY_SERVICE_URL } from '../../../_proxy';

export async function DELETE(req: NextRequest, { params }: { params: { address: string } }) {
  // The guardian is removed from a wallet; the wallet id is read from the
  // request body (the frontend sends { wallet_id }).
  const body = await req.text().catch(() => '{}');
  let walletId = '';
  try { walletId = JSON.parse(body || '{}').wallet_id || ''; } catch {}
  if (!walletId) {
    return Response.json({ success: false, error: 'wallet_id required in body' }, { status: 400 });
  }
  return proxyMutationFrom(req, SOCIAL_RECOVERY_SERVICE_URL, `/api/v1/social-recovery/wallets/${walletId}/guardians/${params.address}`, 'DELETE');
}
