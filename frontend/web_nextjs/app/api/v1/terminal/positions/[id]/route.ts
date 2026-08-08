import { NextRequest } from 'next/server';
import { proxyMutation } from '../../../../../_proxy';

export async function POST(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutation(req, `/terminal/positions/${params.id}/close`, 'POST');
}
