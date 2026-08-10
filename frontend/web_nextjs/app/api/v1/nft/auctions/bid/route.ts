import { NextRequest } from 'next/server';
import { proxyMutationFrom, NFT_SERVICE_URL } from '../../../_proxy';

// Place a bid on an active auction (auth required).
export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, NFT_SERVICE_URL, '/api/v1/nft/auctions/bid', 'POST');
}
