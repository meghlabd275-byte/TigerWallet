import { NextRequest } from 'next/server';
import { proxyGetFrom, NFT_SERVICE_URL } from '../../../_proxy';

// Get the current state of a single auction (public, no auth).
export async function GET(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyGetFrom(req, NFT_SERVICE_URL, `/api/v1/nft/auctions/${encodeURIComponent(params.id)}`);
}
