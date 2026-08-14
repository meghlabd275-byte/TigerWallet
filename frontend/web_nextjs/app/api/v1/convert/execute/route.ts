import { NextRequest } from 'next/server';
import { serviceProxyMutation, CONVERT_SERVICE_URL } from '../../_proxy';

export async function POST(req: NextRequest) {
  return serviceProxyMutation(req, CONVERT_SERVICE_URL, '/convert/execute', 'POST');
}
