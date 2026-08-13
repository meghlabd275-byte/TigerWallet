import { NextRequest } from 'next/server';
import { proxyGetFrom, GIFT_CARD_SERVICE_URL } from '../../_proxy';

export async function GET(req: NextRequest) {
  const search = new URL(req.url).searchParams.toString();
  return proxyGetFrom(req, GIFT_CARD_SERVICE_URL, `/api/v1/gift-cards/list${search ? `?${search}` : ''}`);
}
