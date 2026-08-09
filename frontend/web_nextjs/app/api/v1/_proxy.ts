import { NextRequest, NextResponse } from 'next/server';

export const BACKEND_URL = process.env.BACKEND_URL || process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8443';

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
