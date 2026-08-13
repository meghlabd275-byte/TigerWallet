import { NextRequest } from 'next/server';
import { proxyGetFrom, NFT_SERVICE_URL } from '../../../_proxy';

export async function GET(req: NextRequest) {
  const id = req.nextUrl.pathname.split('/').pop();
  return proxyGetFrom(req, NFT_SERVICE_URL, `/api/v1/nft/auctions/${id}`);
}
