import { NextRequest } from 'next/server';
import { proxyGetFrom, NFT_SERVICE_URL } from '../../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, NFT_SERVICE_URL, '/api/v1/nft/auctions/active');
}
