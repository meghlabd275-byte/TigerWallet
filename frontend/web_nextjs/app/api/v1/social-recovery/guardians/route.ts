import { NextRequest, NextResponse } from 'next/server';
import { proxyGetFrom, proxyMutationFrom, SOCIAL_RECOVERY_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, SOCIAL_RECOVERY_SERVICE_URL, '/api/v1/social-recovery/wallets');
}

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, SOCIAL_RECOVERY_SERVICE_URL, '/api/v1/social-recovery/wallets', 'POST');
}
