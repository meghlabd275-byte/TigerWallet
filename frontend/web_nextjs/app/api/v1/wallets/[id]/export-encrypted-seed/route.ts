import { NextRequest } from 'next/server';
import { proxyMutation } from '../../../_proxy';

// POST /api/v1/wallets/:id/export-encrypted-seed -> wallet_api
// Returns the encrypted seed blob (password-verified) for Google Drive backup.
export async function POST(
  req: NextRequest,
  { params }: { params: { id: string } }
) {
  return proxyMutation(req, `/wallets/${encodeURIComponent(params.id)}/export-encrypted-seed`, 'POST');
}
