// TigerWallet Backend Server
// Production Express server with real database

import express from 'express';
import { createServer } from 'http';
import { config } from './config/index.js';
import { initWebSocket } from './api/websocket.js';
import { securityHeaders, rateLimiter, sanitizeInput } from './api/middleware/auth.js';
import p2pRoutes from './api/controllers/p2p_controller.js';

const app = express();
const httpServer = createServer(app);

initWebSocket(httpServer);

app.use(express.json({ limit: '10mb' }));
app.use(express.urlencoded({ extended: true }));
app.use(sanitizeInput);
app.use(securityHeaders);
app.use(rateLimiter({ maxRequests: 100 }));

app.get('/health', (req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString(), version: '1.0.0' });
});

app.use('/api/v1/p2p', p2pRoutes);

app.use((err: any, req: express.Request, res: express.Response, next: express.NextFunction) => {
  console.error('Error:', err);
  res.status(err.status || 500).json({ error: err.message || 'Internal server error' });
});

app.use((req, res) => { res.status(404).json({ error: 'Not found' }); });

const PORT = config.port;
httpServer.listen(PORT, () => {
  console.log(`TigerWallet API running on port ${PORT}`);
  console.log(`Environment: ${config.nodeEnv}`);
});

export default app;
