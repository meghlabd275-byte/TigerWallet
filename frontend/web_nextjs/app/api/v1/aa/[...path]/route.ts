import { NextRequest } from 'next/server';
import { proxyGetFrom, proxyMutationFrom, AA_SERVICE_URL } from '../../_proxy';

// Catch-all proxy for the ERC-4337 bundler (account_abstraction/go on :8081).
// Forwards /api/v1/aa/<path...> -> :8081/v1/<path...> for GET/POST.
export async function GET(req: NextRequest, { params }: { params: { path: string[] } }) {
  const tail = params.path.join('/');
  const search = new URL(req.url).searchParams.toString();
  return proxyGetFrom(req, AA_SERVICE_URL, `/v1/${tail}${search ? `?${search}` : ''}`);
}

export async function POST(req: NextRequest, { params }: { params: { path: string[] } }) {
  const tail = params.path.join('/');
  return proxyMutationFrom(req, AA_SERVICE_URL, `/v1/${tail}`, 'POST');
}

export async function PUT(req: NextRequest, { params }: { params: { path: string[] } }) {
  const tail = params.path.join('/');
  return proxyMutationFrom(req, AA_SERVICE_URL, `/v1/${tail}`, 'PUT');
}

export async function DELETE(req: NextRequest, { params }: { params: { path: string[] } }) {
  const tail = params.path.join('/');
  return proxyMutationFrom(req, AA_SERVICE_URL, `/v1/${tail}`, 'DELETE');
}
