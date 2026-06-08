import React, { useState, useEffect, useCallback } from 'react';
import { TradingChart } from './components/TradingChart';
import { OrderBook } from './components/OrderBook';
import { OrderForm } from './components/OrderForm';
import { PositionsTable } from './components/PositionsTable';
import { TradesFeed } from './components/TradesFeed';
import { FundingInfo } from './components/FundingInfo';
import { AccountPanel } from './components/AccountPanel';
import { PerpetualsService } from './services/perpetuals';
import './styles.css';

interface AppProps {
  symbol?: string;
  userId?: string;
}

export const TradingTerminal: React.FC<AppProps> = ({ 
  symbol = 'BTC-USD',
  userId = '' 
}) => {
  const [selectedSymbol, setSelectedSymbol] = useState(symbol);
  const [currentPrice, setCurrentPrice] = useState<number>(50000);
  const [priceChange, setPriceChange] = useState<number>(0);
  const [priceChangePercent, setPriceChangePercent] = useState<number>(0);
  const [orderBook, setOrderBook] = useState<{ bids: [string, string][], asks: [string, string][] }>({
    bids: [],
    asks: []
  });
  const [positions, setPositions] = useState<Position[]>([]);
  const [orders, setOrders] = useState<Order[]>([]);
  const [account, setAccount] = useState<AccountInfo | null>(null);
  const [fundingRate, setFundingRate] = useState<string>('0.0001');
  const [nextFundingTime, setNextFundingTime] = useState<number>(0);
  const [wsConnected, setWsConnected] = useState(false);
  const [loading, setLoading] = useState(true);
  
  const perpetualsService = new PerpetualsService();

  // Load initial data
  useEffect(() => {
    const loadData = async () => {
      try {
        setLoading(true);
        
        // Fetch ticker
        const ticker = await perpetualsService.getTicker(selectedSymbol);
        setCurrentPrice(parseFloat(ticker.price));
        setPriceChange(parseFloat(ticker.priceChange));
        setPriceChangePercent(parseFloat(ticker.priceChangePercent));
        
        // Fetch order book
        const depth = await perpetualsService.getDepth(selectedSymbol, 20);
        setOrderBook(depth);
        
        // Fetch positions
        const userPositions = await perpetualsService.getPositions(userId);
        setPositions(userPositions);
        
        // Fetch funding rate
        const funding = await perpetualsService.getFundingRate(selectedSymbol);
        setFundingRate(funding.fundingRate);
        setNextFundingTime(funding.nextFundingTime);
        
        // Fetch account info
        const accountInfo = await perpetualsService.getAccountInfo(userId);
        setAccount(accountInfo);
        
      } catch (error) {
        console.error('Failed to load data:', error);
      } finally {
        setLoading(false);
      }
    };
    
    loadData();
  }, [selectedSymbol, userId]);

  // WebSocket connection
  useEffect(() => {
    const connect = async () => {
      try {
        await perpetualsService.connect();
        setWsConnected(true);
        
        // Subscribe to channels
        perpetualsService.subscribe(['ticker', 'depth', 'trades', 'positions']);
        
        // Listen for updates
        perpetualsService.on('ticker', (data: any) => {
          if (data.symbol === selectedSymbol) {
            setCurrentPrice(parseFloat(data.price));
            setPriceChange(parseFloat(data.priceChange));
            setPriceChangePercent(parseFloat(data.priceChangePercent));
          }
        });
        
        perpetualsService.on('depth', (data: any) => {
          if (data.symbol === selectedSymbol) {
            setOrderBook(data);
          }
        });
        
        perpetualsService.on('positions', (data: any) => {
          setPositions(data);
        });
        
      } catch (error) {
        console.error('WebSocket connection failed:', error);
      }
    };
    
    connect();
    
    return () => {
      perpetualsService.disconnect();
      setWsConnected(false);
    };
  }, [selectedSymbol]);

  const handleSymbolChange = useCallback((newSymbol: string) => {
    setSelectedSymbol(newSymbol);
  }, []);

  const handleOrderSubmit = useCallback(async (order: OrderRequest) => {
    try {
      const result = await perpetualsService.createOrder({
        ...order,
        symbol: selectedSymbol,
        userId
      });
      
      // Refresh positions
      const userPositions = await perpetualsService.getPositions(userId);
      setPositions(userPositions);
      
      return result;
    } catch (error) {
      console.error('Order failed:', error);
      throw error;
    }
  }, [selectedSymbol, userId]);

  const handleCancelOrder = useCallback(async (orderId: string) => {
    try {
      await perpetualsService.cancelOrder(orderId);
      setOrders(prev => prev.filter(o => o.id !== orderId));
    } catch (error) {
      console.error('Cancel failed:', error);
      throw error;
    }
  }, []);

  const handleClosePosition = useCallback(async (posSymbol: string, quantity?: string) => {
    try {
      await perpetualsService.closePosition({
        symbol: posSymbol,
        userId,
        quantity
      });
      
      // Refresh positions
      const userPositions = await perpetualsService.getPositions(userId);
      setPositions(userPositions);
    } catch (error) {
      console.error('Close position failed:', error);
      throw error;
    }
  }, [userId]);

  if (loading) {
    return (
      <div className="trading-terminal loading">
        <div className="loading-spinner">Loading...</div>
      </div>
    );
  }

  return (
    <div className="trading-terminal">
      <header className="terminal-header">
        <div className="symbol-selector">
          <select 
            value={selectedSymbol} 
            onChange={(e) => handleSymbolChange(e.target.value)}
          >
            <option value="BTC-USD">BTC-USD</option>
            <option value="ETH-USD">ETH-USD</option>
            <option value="SOL-USD">SOL-USD</option>
          </select>
        </div>
        
        <div className="price-display">
          <span className={`price ${priceChange >= 0 ? 'positive' : 'negative'}`}>
            ${currentPrice.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
          </span>
          <span className={`change ${priceChange >= 0 ? 'positive' : 'negative'}`}>
            {priceChange >= 0 ? '+' : ''}{priceChange} ({priceChangePercent}%)
          </span>
        </div>
        
        <div className="connection-status">
          <span className={`status-indicator ${wsConnected ? 'connected' : 'disconnected'}`} />
          {wsConnected ? 'Connected' : 'Disconnected'}
        </div>
      </header>

      <main className="terminal-main">
        <section className="chart-section">
          <TradingChart 
            symbol={selectedSymbol} 
            currentPrice={currentPrice}
          />
        </section>

        <section className="orderbook-section">
          <OrderBook 
            bids={orderBook.bids} 
            asks={orderBook.asks}
            currentPrice={currentPrice}
          />
        </section>

        <section className="orderform-section">
          <OrderForm 
            currentPrice={currentPrice}
            symbol={selectedSymbol}
            account={account}
            onSubmit={handleOrderSubmit}
          />
        </section>

        <section className="positions-section">
          <PositionsTable 
            positions={positions}
            onClose={handleClosePosition}
            onCancelOrder={handleCancelOrder}
          />
        </section>

        <section className="trades-section">
          <TradesFeed symbol={selectedSymbol} />
        </section>

        <section className="funding-section">
          <FundingInfo 
            fundingRate={fundingRate}
            nextFundingTime={nextFundingTime}
            symbol={selectedSymbol}
          />
        </section>

        <section className="account-section">
          <AccountPanel account={account} />
        </section>
      </main>
    </div>
  );
};

export default TradingTerminal;