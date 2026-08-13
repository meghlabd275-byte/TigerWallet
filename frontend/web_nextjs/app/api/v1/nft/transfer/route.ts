import { NextRequest } from 'next/server';
import { proxyMutation } from '../../_proxy';

// NFT transfer (ERC-721 safeTransferFrom) is a signing operation performed by
// the canonical wallet_api backend (go/wallet_api/nft_transfer.go) -- it builds
// the on-chain calldata, signs with the wallet's derived key, and broadcasts.
export async function POST(req: NextRequest) {
  return proxyMutation(req, '/nft/transfer', 'POST');
}
