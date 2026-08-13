import { NextRequest } from 'next/server';
import { proxyGetFrom, NFT_SERVICE_URL } from '../../../../_proxy';

export async function GET(req: NextRequest) {
  const id = req.nextUrl.pathname.split('/').slice(-3, -2)[0];
  return proxyGetFrom(req, NFT_SERVICE_URL, `/api/v1/nft/collections/${id}/nfts`);
}
