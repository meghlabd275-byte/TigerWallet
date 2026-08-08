import { NextRequest, NextResponse } from 'next/server';
import { proxyGet } from '../../_proxy';

/**
 * GET /api/v1/security/check-address?address=...&chain=ethereum
 *
 * Proxies to the TigerWallet backend security service, which checks the
 * address against the MetaMask malicious-address blocklist and performs an
 * eth_getCode lookup to determine whether it is a contract. Returns:
 *   { success, data: { address, chain, classification, risk_level, reasons,
 *                       is_contract, code_size, cached, checked_at } }
 */
export async function GET(req: NextRequest) {
  const url = new URL(req.url);
  const address = url.searchParams.get('address');
  if (!address) {
    return NextResponse.json(
      { success: false, error: 'address query parameter required' },
      { status: 400 }
    );
  }
  return proxyGet(req, '/security/check-address');
}
