import { NextRequest, NextResponse } from 'next/server';
import { proxyMutation } from '../../../_proxy';

export async function DELETE(req: NextRequest, { params }: { params: { address: string } }) {
  return proxyMutation(req, `/social-recovery/guardians/${params.address}`, 'DELETE');
}
