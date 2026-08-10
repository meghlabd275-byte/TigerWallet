import { NextRequest, NextResponse } from 'next/server';

export const BACKEND_URL = process.env.BACKEND_URL || process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8443';

// The swap/DEX service runs as its own process (go/swap_service) on a separate
// port (default 8005). It is distinct from the wallet_api signing backend.
export const SWAP_SERVICE_URL = process.env.SWAP_SERVICE_URL || process.env.NEXT_PUBLIC_SWAP_URL || 'http://localhost:8005';

// The staking service (go/staking_service) runs on port 8003.
export const STAKING_SERVICE_URL = process.env.STAKING_SERVICE_URL || process.env.NEXT_PUBLIC_STAKING_URL || 'http://localhost:8003';

// The NFT marketplace service (go/nft_service) runs on port 8004.
export const NFT_SERVICE_URL = process.env.NFT_SERVICE_URL || process.env.NEXT_PUBLIC_NFT_URL || 'http://localhost:8004';

// The listing service (go/listing_service) runs on port 8097.
export const LISTING_SERVICE_URL = process.env.LISTING_SERVICE_URL || process.env.NEXT_PUBLIC_LISTING_URL || 'http://localhost:8097';

// The lending service (go/lending_service) runs on port 8009.
export const LENDING_SERVICE_URL = process.env.LENDING_SERVICE_URL || process.env.NEXT_PUBLIC_LENDING_URL || 'http://localhost:8009';

// The copy-trading service (go/copy_trading_service) runs on port 8006.
export const COPY_TRADING_SERVICE_URL = process.env.COPY_TRADING_SERVICE_URL || process.env.NEXT_PUBLIC_COPY_TRADING_URL || 'http://localhost:8006';

// --- Remaining microservices (go/<name>) ---
// Each runs as its own process on the documented default port. Ports are
// overridable via env so co-located deployments can resolve collisions.
export const ADMIN_SERVICE_URL = process.env.ADMIN_SERVICE_URL || 'http://localhost:8002';
export const ANALYTICS_SERVICE_URL = process.env.ANALYTICS_SERVICE_URL || 'http://localhost:8088';
export const ANALYTICS_SERVICE_ALT_URL = process.env.ANALYTICS_SERVICE_ALT_URL || 'http://localhost:8010';
export const API_GATEWAY_URL = process.env.API_GATEWAY_URL || 'http://localhost:8000';
export const APPROVAL_MANAGER_URL = process.env.APPROVAL_MANAGER_URL || 'http://localhost:8449';
export const BLOCKCHAIN_RPC_URL = process.env.BLOCKCHAIN_RPC_URL || 'http://localhost:8090';
export const BRIDGE_SERVICE_URL = process.env.BRIDGE_SERVICE_URL || 'http://localhost:8007';
export const BRIDGE_AGGREGATOR_URL = process.env.BRIDGE_AGGREGATOR_URL || 'http://localhost:8447';
export const CARD_SERVICE_URL = process.env.CARD_SERVICE_URL || 'http://localhost:8457';
export const BUG_BOUNTY_SERVICE_URL = process.env.BUG_BOUNTY_SERVICE_URL || 'http://localhost:8080';
export const CEX_CONNECTOR_URL = process.env.CEX_CONNECTOR_URL || 'http://localhost:8090';
export const CLOUD_BACKUP_SERVICE_URL = process.env.CLOUD_BACKUP_SERVICE_URL || 'http://localhost:8452';
export const ENS_SERVICE_URL = process.env.ENS_SERVICE_URL || 'http://localhost:8453';
export const ENTERPRISE_SERVICE_URL = process.env.ENTERPRISE_SERVICE_URL || 'http://localhost:8443';
export const FIAT_SERVICE_URL = process.env.FIAT_SERVICE_URL || 'http://localhost:8008';
export const FIAT_ONRAMP_URL = process.env.FIAT_ONRAMP_URL || 'http://localhost:8451';
export const FIAT_OFFRAMP_URL = process.env.FIAT_OFFRAMP_URL || 'http://localhost:8452';
export const GAS_ORACLE_URL = process.env.GAS_ORACLE_URL || 'http://localhost:8445';
export const GOVERNANCE_SERVICE_URL = process.env.GOVERNANCE_SERVICE_URL || 'http://localhost:8454';
export const PREDICTION_SERVICE_URL = process.env.PREDICTION_SERVICE_URL || 'http://localhost:8455';
export const RWA_SERVICE_URL = process.env.RWA_SERVICE_URL || 'http://localhost:8456';
export const LEADERBOARD_SERVICE_URL = process.env.LEADERBOARD_SERVICE_URL || 'http://localhost:8458';
export const INSURANCE_SERVICE_URL = process.env.INSURANCE_SERVICE_URL || 'http://localhost:8459';
export const IEO_SERVICE_URL = process.env.IEO_SERVICE_URL || 'http://localhost:8460';
export const BOTS_SERVICE_URL = process.env.BOTS_SERVICE_URL || 'http://localhost:8461';
export const TWAP_SERVICE_URL = process.env.TWAP_SERVICE_URL || 'http://localhost:8462';
export const POOL_SERVICE_URL = process.env.POOL_SERVICE_URL || 'http://localhost:8463';
export const GRAPHQL_SERVICE_URL = process.env.GRAPHQL_SERVICE_URL || 'http://localhost:9003';
export const LAUNCHPAD_SERVICE_URL = process.env.LAUNCHPAD_SERVICE_URL || 'http://localhost:8012';
export const LIQUID_STAKING_URL = process.env.LIQUID_STAKING_URL || 'http://localhost:8448';
export const LIQUIDITY_URL = process.env.LIQUIDITY_URL || 'http://localhost:8090';
export const LOG_AGGREGATION_SERVICE_URL = process.env.LOG_AGGREGATION_SERVICE_URL || 'http://localhost:9200';
export const MATCHING_ENGINE_URL = process.env.MATCHING_ENGINE_URL || 'http://localhost:8092';
export const MONITORING_SERVICE_URL = process.env.MONITORING_SERVICE_URL || 'http://localhost:9090';
export const MULTISIG_SERVICE_URL = process.env.MULTISIG_SERVICE_URL || 'http://localhost:8450';
export const NFT_PRICES_URL = process.env.NFT_PRICES_URL || 'http://localhost:8089';
export const NOTIFICATION_SERVICE_URL = process.env.NOTIFICATION_SERVICE_URL || 'http://localhost:8011';
export const ORACLE_SERVICE_URL = process.env.ORACLE_SERVICE_URL || 'http://localhost:8093';
export const OTP_SERVICE_URL = process.env.OTP_SERVICE_URL || 'http://localhost:8104';
export const PAYMENT_SERVICE_URL = process.env.PAYMENT_SERVICE_URL || 'http://localhost:8096';
export const PERPETUAL_SERVICE_URL = process.env.PERPETUAL_SERVICE_URL || 'http://localhost:8009';
export const PORTFOLIO_TRACKER_URL = process.env.PORTFOLIO_TRACKER_URL || 'http://localhost:8081';
export const PROTECTION_FUND_SERVICE_URL = process.env.PROTECTION_FUND_SERVICE_URL || 'http://localhost:8081';
export const PUSH_NOTIFICATIONS_URL = process.env.PUSH_NOTIFICATIONS_URL || 'http://localhost:8085';
export const RATE_LIMITER_SERVICE_URL = process.env.RATE_LIMITER_SERVICE_URL || 'http://localhost:9004';
export const RBAC_ADMIN_SERVICE_URL = process.env.RBAC_ADMIN_SERVICE_URL || 'http://localhost:8081';
export const REAL_TIME_CHARTS_URL = process.env.REAL_TIME_CHARTS_URL || 'http://localhost:8080';
export const RPC_NODE_MANAGER_URL = process.env.RPC_NODE_MANAGER_URL || 'http://localhost:8087';
export const RPC_SERVICE_URL = process.env.RPC_SERVICE_URL || 'http://localhost:8080';
export const SCHEDULER_URL = process.env.SCHEDULER_URL || 'http://localhost:8095';
export const SIGNATURE_SERVICE_URL = process.env.SIGNATURE_SERVICE_URL || 'http://localhost:8444';
export const SOCIAL_RECOVERY_SERVICE_URL = process.env.SOCIAL_RECOVERY_SERVICE_URL || 'http://localhost:8451';
export const SUPER_ADMIN_SERVICE_URL = process.env.SUPER_ADMIN_SERVICE_URL || 'http://localhost:8080';
export const TAX_REPORTS_URL = process.env.TAX_REPORTS_URL || 'http://localhost:8082';
export const TWO_FACTOR_AUTH_URL = process.env.TWO_FACTOR_AUTH_URL || 'http://localhost:8446';
export const WALLETCONNECT_URL = process.env.WALLETCONNECT_URL || 'http://localhost:8443';
export const WEBHOOK_SERVICE_URL = process.env.WEBHOOK_SERVICE_URL || 'http://localhost:9002';
export const WEBSOCKET_SERVICE_URL = process.env.WEBSOCKET_SERVICE_URL || 'http://localhost:8095';
export const WHITE_LABEL_SERVICE_URL = process.env.WHITE_LABEL_SERVICE_URL || 'http://localhost:8085';

// Fetch a path from a specific service base URL (for services that do NOT run
// on the wallet_api). Falls back gracefully with a 502 on connection error.
export async function proxyGetFrom(req: NextRequest, baseUrl: string, path: string): Promise<NextResponse> {
  const url = new URL(req.url);
  const search = url.searchParams.toString();
  try {
    const res = await fetch(`${baseUrl}${path}${search ? `?${search}` : ''}`, {
      headers: authHeaders(req),
      cache: 'no-store',
    });
    const data = await res.json();
    return NextResponse.json(data, { status: res.status });
  } catch {
    return NextResponse.json(
      { success: false, error: `Failed to fetch ${path} from backend` },
      { status: 502 }
    );
  }
}

export async function proxyMutationFrom(
  req: NextRequest,
  baseUrl: string,
  path: string,
  method: 'POST' | 'PUT' | 'DELETE'
): Promise<NextResponse> {
  try {
    let body: string | undefined;
    if (method !== 'DELETE') {
      body = await req.text();
    }
    const res = await fetch(`${baseUrl}${path}`, {
      method,
      headers: authHeaders(req),
      body,
      cache: 'no-store',
    });
    const data = await res.json().catch(() => ({}));
    return NextResponse.json(data, { status: res.status });
  } catch {
    return NextResponse.json(
      { success: false, error: `Failed to mutate ${path} on backend` },
      { status: 502 }
    );
  }
}

export function authHeaders(req: NextRequest): Record<string, string> {
  const auth = req.headers.get('authorization');
  return auth ? { Authorization: auth, 'Content-Type': 'application/json' } : { 'Content-Type': 'application/json' };
}

export async function proxyGet(req: NextRequest, path: string): Promise<NextResponse> {
  const url = new URL(req.url);
  const search = url.searchParams.toString();
  try {
    const res = await fetch(`${BACKEND_URL}/api/v1${path}${search ? `?${search}` : ''}`, {
      headers: authHeaders(req),
      cache: 'no-store',
    });
    const data = await res.json();
    return NextResponse.json(data, { status: res.status });
  } catch (error) {
    return NextResponse.json(
      { success: false, error: `Failed to fetch ${path} from backend` },
      { status: 502 }
    );
  }
}

export async function proxyMutation(
  req: NextRequest,
  path: string,
  method: 'POST' | 'PUT' | 'DELETE'
): Promise<NextResponse> {
  try {
    let body: string | undefined;
    if (method !== 'DELETE') {
      body = await req.text();
    }
    const res = await fetch(`${BACKEND_URL}/api/v1${path}`, {
      method,
      headers: authHeaders(req),
      body,
      cache: 'no-store',
    });
    const data = await res.json().catch(() => ({}));
    return NextResponse.json(data, { status: res.status });
  } catch (error) {
    return NextResponse.json(
      { success: false, error: `Failed to ${method.toLowerCase()} ${path} on backend` },
      { status: 502 }
    );
  }
}

// --- Swap service proxy (targets go/swap_service on SWAP_SERVICE_URL) ---

export async function serviceProxyGet(req: NextRequest, baseUrl: string, path: string): Promise<NextResponse> {
  const url = new URL(req.url);
  const search = url.searchParams.toString();
  try {
    const res = await fetch(`${baseUrl}/api/v1${path}${search ? `?${search}` : ''}`, {
      headers: authHeaders(req),
      cache: 'no-store',
    });
    const data = await res.json();
    return NextResponse.json(data, { status: res.status });
  } catch (error) {
    return NextResponse.json(
      { success: false, error: `Failed to fetch ${path} from backend` },
      { status: 502 }
    );
  }
}

export async function serviceProxyMutation(
  req: NextRequest,
  baseUrl: string,
  path: string,
  method: 'POST' | 'PUT' | 'DELETE'
): Promise<NextResponse> {
  try {
    let body: string | undefined;
    if (method !== 'DELETE') {
      body = await req.text();
    }
    const res = await fetch(`${baseUrl}/api/v1${path}`, {
      method,
      headers: authHeaders(req),
      body,
      cache: 'no-store',
    });
    const data = await res.json().catch(() => ({}));
    return NextResponse.json(data, { status: res.status });
  } catch (error) {
    return NextResponse.json(
      { success: false, error: `Failed to ${method.toLowerCase()} ${path} on backend` },
      { status: 502 }
    );
  }
}

export async function swapProxyGet(req: NextRequest, path: string): Promise<NextResponse> {
  return serviceProxyGet(req, SWAP_SERVICE_URL, path);
}

export async function swapProxyMutation(
  req: NextRequest,
  path: string,
  method: 'POST' | 'PUT' | 'DELETE'
): Promise<NextResponse> {
  return serviceProxyMutation(req, SWAP_SERVICE_URL, path, method);
}
