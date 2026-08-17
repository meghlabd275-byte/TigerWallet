import { NextRequest } from 'next/server';
import { proxyMutation } from '../../_proxy';

// UserWallet no-registration: provisions an anonymous guest account.
export async function POST(req: NextRequest) {
  return proxyMutation(req, '/auth/guest', 'POST');
}
