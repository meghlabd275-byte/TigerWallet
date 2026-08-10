import { NextRequest } from 'next/server';
import { proxyGetFrom, LEADERBOARD_SERVICE_URL } from '../_proxy';

export async function GET(req: NextRequest) {
  return proxyGetFrom(req, LEADERBOARD_SERVICE_URL, '/api/v1/leaderboard');
}
