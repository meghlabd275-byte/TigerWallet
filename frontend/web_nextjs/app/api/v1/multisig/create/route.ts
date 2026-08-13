import { NextRequest } from 'next/server';
import { MULTISIG_SERVICE_URL, proxyMutationFrom } from '../../_proxy';

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, MULTISIG_SERVICE_URL, '/api/v1/multisig/wallets', 'POST');
}
