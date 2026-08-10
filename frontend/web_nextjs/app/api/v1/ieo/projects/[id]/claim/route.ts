import { NextRequest } from 'next/server';
import { proxyMutationFrom, IEO_SERVICE_URL } from '../../../../_proxy';

export async function POST(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutationFrom(req, IEO_SERVICE_URL, '/api/v1/ieo/claim', 'POST');
}
