import { NextRequest } from 'next/server';
import { proxyGetFrom, proxyMutationFrom, IEO_SERVICE_URL } from '../../../_proxy';

export async function GET(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyGetFrom(req, IEO_SERVICE_URL, `/api/v1/ieo/rounds/${params.id}`);
}

export async function POST(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutationFrom(req, IEO_SERVICE_URL, `/api/v1/ieo/rounds/${params.id}/participate`, 'POST');
}
