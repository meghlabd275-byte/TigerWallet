import { NextRequest } from 'next/server';
import { proxyMutationFrom, GIFT_CARD_SERVICE_URL } from '../../_proxy';

export async function POST(req: NextRequest) {
  return proxyMutationFrom(req, GIFT_CARD_SERVICE_URL, '/api/v1/gift-cards/buy', 'POST');
}
