import { NextRequest } from 'next/server';
import { serviceProxyGet, NFT_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return serviceProxyGet(req, NFT_SERVICE_URL, '/nft/listings');
}
