// P2P Trading Routes - REST API
import { Router, Request, Response } from 'express';
import { authenticate, rateLimiter, requireKyc, auditLog } from '../middleware/auth.js';
import p2pService from '../services/p2p_service.js';
import { body, query as validateQuery, param } from 'express-validator';

const router = Router();

// Get P2P advertisements
router.get('/adverts',
  rateLimiter({ maxRequests: 30 }),
  validateQuery('token').optional().isString(),
  validateQuery('side').optional().isIn(['BUY', 'SELL']),
  validateQuery('fiatCurrency').optional().isString(),
  validateQuery('paymentMethod').optional().isString(),
  validateQuery('page').optional().isInt({ min: 1 }),
  validateQuery('limit').optional().isInt({ min: 1, max: 100 }),
  async (req: Request, res: Response) => {
    try {
      const filters = {
        token: req.query.token as string,
        side: req.query.side as string,
        fiatCurrency: req.query.fiatCurrency as string,
        paymentMethod: req.query.paymentMethod as string,
        page: parseInt(req.query.page as string) || 1,
        limit: parseInt(req.query.limit as string) || 20
      };
      
      const adverts = await p2pService.getAdverts(filters);
      res.json({ success: true, data: adverts });
    } catch (error: any) {
      res.status(500).json({ error: error.message });
    }
  }
);

// Create P2P order
router.post('/orders',
  authenticate,
  rateLimiter({ maxRequests: 10 }),
  requireKyc('BASIC'),
  auditLog('CREATE_P2P_ORDER', 'order'),
  [
    body('advertId').isUUID().withMessage('Valid advert ID required'),
    body('amount').isFloat({ min: 0.01 }).withMessage('Amount must be greater than 0')
  ],
  async (req: Request, res: Response) => {
    try {
      const { advertId, amount } = req.body;
      
      const result = await p2pService.createOrder({
        advertId,
        buyerId: req.user!.id,
        amount
      });
      
      res.json({ success: true, data: result });
    } catch (error: any) {
      res.status(400).json({ error: error.message });
    }
  }
);

// Get user's orders
router.get('/orders',
  authenticate,
  rateLimiter({ maxRequests: 20 }),
  validateQuery('status').optional().isIn(['PENDING', 'PAID', 'COMPLETED', 'CANCELLED', 'DISPUTED']),
  async (req: Request, res: Response) => {
    try {
      const orders = await p2pService.getUserOrders(req.user!.id, req.query.status as string);
      res.json({ success: true, data: orders });
    } catch (error: any) {
      res.status(500).json({ error: error.message });
    }
  }
);

// Get order details
router.get('/orders/:orderId',
  authenticate,
  param('orderId').isUUID(),
  async (req: Request, res: Response) => {
    try {
      const orders = await p2pService.getUserOrders(req.user!.id);
      const order = orders.find((o: any) => o.id === req.params.orderId);
      
      if (!order) {
        return res.status(404).json({ error: 'Order not found' });
      }
      
      res.json({ success: true, data: order });
    } catch (error: any) {
      res.status(500).json({ error: error.message });
    }
  }
);

// Mark order as paid
router.post('/orders/:orderId/pay',
  authenticate,
  rateLimiter({ maxRequests: 5 }),
  requireKyc('BASIC'),
  auditLog('ORDER_PAID', 'order'),
  [
    param('orderId').isUUID(),
    body('paymentProof').isString().notEmpty()
  ],
  async (req: Request, res: Response) => {
    try {
      const result = await p2pService.markAsPaid(
        req.params.orderId,
        req.user!.id,
        req.body.paymentProof
      );
      
      res.json({ success: true, data: result });
    } catch (error: any) {
      res.status(400).json({ error: error.message });
    }
  }
);

// Confirm payment (seller releases crypto)
router.post('/orders/:orderId/confirm',
  authenticate,
  rateLimiter({ maxRequests: 5 }),
  requireKyc('BASIC'),
  auditLog('ORDER_CONFIRMED', 'order'),
  param('orderId').isUUID(),
  async (req: Request, res: Response) => {
    try {
      const result = await p2pService.confirmPayment(req.params.orderId, req.user!.id);
      res.json({ success: true, data: result });
    } catch (error: any) {
      res.status(400).json({ error: error.message });
    }
  }
);

// Cancel order
router.post('/orders/:orderId/cancel',
  authenticate,
  rateLimiter({ maxRequests: 5 }),
  requireKyc('BASIC'),
  auditLog('ORDER_CANCELLED', 'order'),
  [
    param('orderId').isUUID(),
    body('reason').isString().notEmpty()
  ],
  async (req: Request, res: Response) => {
    try {
      const result = await p2pService.cancelOrder(
        req.params.orderId,
        req.user!.id,
        req.body.reason
      );
      
      res.json({ success: true, data: result });
    } catch (error: any) {
      res.status(400).json({ error: error.message });
    }
  }
);

// Open dispute
router.post('/orders/:orderId/dispute',
  authenticate,
  rateLimiter({ maxRequests: 3 }),
  requireKyc('INTERMEDIATE'),
  auditLog('DISPUTE_OPENED', 'order'),
  [
    param('orderId').isUUID(),
    body('reason').isString().isLength({ min: 20 })
  ],
  async (req: Request, res: Response) => {
    try {
      const result = await p2pService.openDispute(
        req.params.orderId,
        req.user!.id,
        req.body.reason
      );
      
      res.json({ success: true, data: result });
    } catch (error: any) {
      res.status(400).json({ error: error.message });
    }
  }
);

// Get payment methods
router.get('/payment-methods',
  async (req: Request, res: Response) => {
    res.json({
      success: true,
      data: [
        { id: 'bank_transfer', name: 'Bank Transfer', type: 'bank' },
        { id: 'paypal', name: 'PayPal', type: 'ewallet' },
        { id: 'alipay', name: 'AliPay', type: 'ewallet' },
        { id: 'wechat_pay', name: 'WeChat Pay', type: 'ewallet' },
        { id: 'upi', name: 'UPI', type: 'bank' },
        { id: 'gift_card', name: 'Gift Card', type: 'card' }
      ]
    });
  }
);

// Get supported fiat currencies
router.get('/fiat-currencies',
  async (req: Request, res: Response) => {
    res.json({
      success: true,
      data: [
        { code: 'USD', name: 'US Dollar', symbol: '$' },
        { code: 'EUR', name: 'Euro', symbol: '€' },
        { code: 'GBP', name: 'British Pound', symbol: '£' },
        { code: 'CNY', name: 'Chinese Yuan', symbol: '¥' },
        { code: 'INR', name: 'Indian Rupee', symbol: '₹' },
        { code: 'KRW', name: 'Korean Won', symbol: '₩' },
        { code: 'JPY', name: 'Japanese Yen', symbol: '¥' },
        { code: 'BRL', name: 'Brazilian Real', symbol: 'R$' }
      ]
    });
  }
);

export default router;
