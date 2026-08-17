import { NextRequest } from 'next/server';
import { proxyMutation } from '../../_proxy';

// POST /api/v1/wallets/import-encrypted-seed -> wallet_api
// Restores a wallet from an encrypted seed blob (e.g. Google Drive restore).
export async function POST(req: NextRequest) {
  return proxyMutation(req, '/wallets/import-encrypted-seed', 'POST');
}
