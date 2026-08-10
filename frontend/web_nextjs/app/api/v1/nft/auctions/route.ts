import { NextRequest } from 'next/server';
import { proxyMutationFrom, proxyGetFrom, NFT_SERVICE_URL } from '../../_proxy';

// Create an English-style auction for an NFT (auth required).
export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, NFT_SERVICE_URL, '/api/v1/nft/auctions', 'POST');
}

// List currently-active auctions (optional ?collection_id=...).
export async function GET(req: NextRequest) {
  const url = new URL(req.url);
  const qs = url.searchParams.toString();
  return proxyGetFrom(req, NFT_SERVICE_URL, `/api/v1/nft/auctions/active${qs ? `?${qs}` : ''}`);
}
