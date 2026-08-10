import { NextRequest } from 'next/server';
import { serviceProxyMutation, NFT_SERVICE_URL } from '../../_proxy';

export async function POST(req: NextRequest) {
  return serviceProxyMutation(req, NFT_SERVICE_URL, '/nft/buy', 'POST');
}
