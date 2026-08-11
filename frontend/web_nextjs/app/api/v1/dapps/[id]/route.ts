import { NextRequest } from 'next/server';
import { proxyGet } from '../../_proxy';

export async function GET(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyGet(req, `/dapps/${params.id}`);
}
