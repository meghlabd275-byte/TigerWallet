import { NextRequest } from 'next/server';
import { MULTISIG_SERVICE_URL, proxyMutationFrom } from '../../_proxy';

// Reads transactionId from the body, then forwards to /multisig/transactions/:id/sign
export async function POST(req: NextRequest) {
  const body = await req.json();
  const txId = body.transactionId;
  if (!txId) {
    return Response.json({ error: 'transactionId is required' }, { status: 400 });
  }
  const { transactionId: _, ...rest } = body;
  return proxyMutationFrom(
    new NextRequest(req, { body: JSON.stringify(rest) }),
    MULTISIG_SERVICE_URL,
    `/api/v1/multisig/transactions/${txId}/sign`,
    'POST'
  );
}
