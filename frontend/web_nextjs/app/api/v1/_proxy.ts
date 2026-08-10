import { NextRequest, NextResponse } from 'next/server';

export const BACKEND_URL = process.env.BACKEND_URL || process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8443';

// The swap/DEX service runs as its own process (go/swap_service) on a separate
// port (default 8005). It is distinct from the wallet_api signing backend.
export const SWAP_SERVICE_URL = process.env.SWAP_SERVICE_URL || process.env.NEXT_PUBLIC_SWAP_URL || 'http://localhost:8005';

// The staking service (go/staking_service) runs on port 8003.
export const STAKING_SERVICE_URL = process.env.STAKING_SERVICE_URL || process.env.NEXT_PUBLIC_STAKING_URL || 'http://localhost:8003';

// The NFT marketplace service (go/nft_service) runs on port 8008.
export const NFT_SERVICE_URL = process.env.NFT_SERVICE_URL || process.env.NEXT_PUBLIC_NFT_URL || 'http://localhost:8008';

// The listing service (go/listing_service) runs on port 8097.
export const LISTING_SERVICE_URL = process.env.LISTING_SERVICE_URL || process.env.NEXT_PUBLIC_LISTING_URL || 'http://localhost:8097';

// The lending service (go/lending_service) runs on port 8009.
export const LENDING_SERVICE_URL = process.env.LENDING_SERVICE_URL || process.env.NEXT_PUBLIC_LENDING_URL || 'http://localhost:8009';

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
