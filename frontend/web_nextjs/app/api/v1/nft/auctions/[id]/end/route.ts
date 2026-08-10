import { NextRequest } from 'next/server';
import { proxyMutationFrom, NFT_SERVICE_URL } from '../../../../_proxy';

// Settle/end an auction (auth required).
export async function POST(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutationFrom(
    req,
    NFT_SERVICE_URL,
    `/api/v1/nft/auctions/${encodeURIComponent(params.id)}/end`,
    'POST'
  );
}
