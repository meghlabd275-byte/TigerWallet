import { NextRequest } from 'next/server';
import { proxyMutationFrom, SOCIAL_RECOVERY_SERVICE_URL } from '../../../_proxy';

export async function POST(req: NextRequest, { params }: { params: { id: string } }) {
  return proxyMutationFrom(req, SOCIAL_RECOVERY_SERVICE_URL, `/api/v1/social-recovery/recoveries/${params.id}/confirm`, 'POST');
}
