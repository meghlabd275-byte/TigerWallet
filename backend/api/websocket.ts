// WebSocket Handler - Real-time updates
import { Server as HTTPServer } from 'http';
import { Server as SocketServer } from 'socket.io';
import jwt from 'jsonwebtoken';
import { config } from '../config/index.js';

interface AuthenticatedSocket extends SocketIO.Socket {
  userId?: string;
}

const userSockets = new Map<string, Set<string>>();
let io: SocketServer;

export const initWebSocket = (httpServer: HTTPServer): SocketServer => {
  io = new SocketServer(httpServer, {
    cors: { origin: config.security.corsOrigins, credentials: true },
    pingTimeout: 60000, pingInterval: 25000
  });

  io.use((socket: AuthenticatedSocket, next) => {
    const token = socket.handshake.auth.token || socket.handshake.query.token;
    if (!token) return next(new Error('Authentication required'));
    try {
      const decoded = jwt.verify(token as string, config.jwt.secret) as { userId: string };
      socket.userId = decoded.userId;
      next();
    } catch { next(new Error('Invalid token')); }
  });

  io.on('connection', (socket: AuthenticatedSocket) => {
    if (socket.userId) {
      if (!userSockets.has(socket.userId)) userSockets.set(socket.userId, new Set());
      userSockets.get(socket.userId)!.add(socket.id);
      socket.join(`user:${socket.userId}`);
    }
    socket.on('subscribe:market', (s: string) => socket.join(`market:${s}`));
    socket.on('unsubscribe:market', (s: string) => socket.leave(`market:${s}`));
    socket.on('subscribe:order', (o: string) => socket.join(`order:${o}`));
    socket.on('disconnect', () => {
      if (socket.userId) {
        const sockets = userSockets.get(socket.userId);
        if (sockets) { sockets.delete(socket.id); if (sockets.size === 0) userSockets.delete(socket.userId); }
      }
    });
  });
  return io;
};

export const broadcastToUser = (userId: string, event: string, data: any) => {
  if (io) io.to(`user:${userId}`).emit(event, data);
};

export const broadcastToMarket = (symbol: string, event: string, data: any) => {
  if (io) io.to(`market:${symbol}`).emit(event, data);
};

export const broadcastPriceUpdate = (prices: Record<string, number>) => {
  if (io) io.emit('prices:update', prices);
};

export default { initWebSocket, broadcastToUser, broadcastToMarket, broadcastPriceUpdate };
