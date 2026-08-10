import { NextRequest } from 'next/server';
import { proxyMutationFrom, NFT_SERVICE_URL } from '../../../_proxy';

// Cancel an active fixed-price NFT listing. Only the listing owner may cancel.
export async function DELETE(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutationFrom(
    req,
    NFT_SERVICE_URL,
    `/api/v1/nft/listings/${encodeURIComponent(params.id)}`,
    'DELETE'
  );
}
