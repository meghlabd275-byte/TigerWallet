import { NextRequest } from 'next/server';
import { proxyMutation } from '../_proxy';

// UserWallet auto-send: self-sign + MasterWallet policy auto-approval.
export async function POST(req: NextRequest) {
  const master = req.nextUrl.searchParams.get('master_wallet_id');
  const path = master ? `/auto-send?master_wallet_id=${master}` : '/auto-send';
  return proxyMutation(req, path, 'POST');
}
