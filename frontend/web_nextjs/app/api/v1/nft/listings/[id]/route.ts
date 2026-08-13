import { NextRequest } from 'next/server';
import { proxyMutationFrom, NFT_SERVICE_URL } from '../../../_proxy';

export async function DELETE(req: NextRequest) {
  const id = req.nextUrl.pathname.split('/').pop();
  return proxyMutationFrom(req, NFT_SERVICE_URL, `/api/v1/nft/listings/${id}`, 'DELETE');
}
