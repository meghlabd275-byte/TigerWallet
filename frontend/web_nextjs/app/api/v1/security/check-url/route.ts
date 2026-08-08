import { NextRequest, NextResponse } from 'next/server';
import { proxyGet } from '../../_proxy';

/**
 * GET /api/v1/security/check-url?url=...
 *
 * Proxies to the TigerWallet backend security service, which classifies the
 * dApp URL against the MetaMask eth-phishing-detect blocklist and returns:
 *   { success, data: { url, host, classification, risk_level, reasons, cached, checked_at } }
 */
export async function GET(req: NextRequest) {
  const url = new URL(req.url);
  const target = url.searchParams.get('url');
  if (!target) {
    return NextResponse.json(
      { success: false, error: 'url query parameter required' },
      { status: 400 }
    );
  }
  return proxyGet(req, '/security/check-url');
}
