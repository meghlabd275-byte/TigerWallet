import { NextRequest } from 'next/server';
import { proxyMutation } from '../../_proxy';

// Mnemonic import is served by the wallet-api backend's POST /api/v1/wallets
// endpoint (create-with-mnemonic): include a `mnemonic` to import, omit it to
// generate a fresh wallet. This is the canonical path and is NOT the same as
// /wallets/import-encrypted-seed (Google Drive encrypted-blob restore) or
// /keystore/import (Web3 Secret Storage V3 keystore), which have their own
// dedicated routes under /api/v1/wallets and /api/v1/keystore.
export async function POST(req: NextRequest) {
  return proxyMutation(req, '/wallets', 'POST');
}
