import { NextRequest } from 'next/server';
import { proxyMutation } from '../../_proxy';

// The canonical handler is POST /api/v1/gas/estimate and binds a JSON body
// ({from,to,value,data,chain_id}). Proxying GET here sent no body, so every
// call to this route failed before reaching the backend. Forward the POST.
export async function POST(req: NextRequest) {
  return proxyMutation(req, '/gas/estimate', 'POST');
}
