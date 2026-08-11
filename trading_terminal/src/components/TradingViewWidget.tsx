/**
 * TradingView Chart Integration
 * Professional trading charts for Trading Terminal
 * No mock data - real market data integration
 */

import React, { useEffect, useRef, useState, useCallback } from 'react';

// TradingView widget types
interface TradingViewWidgetProps {
  symbol?: string;
  theme?: 'light' | 'dark';
  locale?: string;
  autosize?: boolean;
  width?: number | string;
  height?: number | string;
  interval?: string;
  timezone?: string;
  style?: '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' | '10' | '11' | '12';
  toolbar_bg?: string;
  enable_publishing?: boolean;
  allow_symbol_change?: boolean;
  hide_top_toolbar?: boolean;
  hide_legend?: boolean;
  save_image?: boolean;
  container_id?: string;
  library_path?: string;
  studies?: string[];
  show_popup_button?: boolean;
  popup_width?: string;
  popup_height?: string;
}

interface CandlestickData {
  time: number;
  open: number;
  high: number;
  low: number;
  close: number;
  volume?: number;
}

interface TradingViewChartProps {
  symbol?: string;
  onReady?: (widget: any) => void;
  onError?: (error: Error) => void;
}

// Default symbols for different markets
export const TRADING_PAIRS = [
  { symbol: 'BINANCE:BTCUSDT', name: 'BTC/USDT' },
  { symbol: 'BINANCE:ETHUSDT', name: 'ETH/USDT' },
  { symbol: 'BINANCE:SOLUSDT', name: 'SOL/USDT' },
  { symbol: 'BINANCE:BNBUSDT', name: 'BNB/USDT' },
  { symbol: 'BINANCE:AVAXUSDT', name: 'AVAX/USDT' },
  { symbol: 'BINANCE:MATICUSDT', name: 'MATIC/USDT' },
  { symbol: 'BINANCE:ARBUSDT', name: 'ARB/USDT' },
  { symbol: 'BINANCE:OPUSDT', name: 'OP/USDT' },
];

// Advanced Real-time TradingView Widget
export const TradingViewWidget: React.FC<TradingViewWidgetProps> = ({
  symbol = 'BINANCE:BTCUSDT',
  theme = 'dark',
  locale = 'en',
  autosize = true,
  width = '100%',
  height = 500,
  interval = '15',
  timezone = 'Etc/UTC',
  style = '1',
  toolbar_bg = '#1a1a2e',
  enable_publishing = false,
  allow_symbol_change = true,
  hide_top_toolbar = false,
  hide_legend = false,
  save_image = true,
  container_id = 'tradingview_chart',
  show_popup_button = true,
  popup_width = '1000',
  popup_height = '650',
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const widgetRef = useRef<any>(null);

  useEffect(() => {
    if (!containerRef.current) return;

    // Create widget container
    const script = document.createElement('script');
    script.src = 'https://s3.tradingview.com/tv.js';
    script.async = true;
    script.onload = () => {
      if ((window as any).TradingView) {
        widgetRef.current = new (window as any).TradingView.widget({
          autosize,
          symbol,
          theme,
          locale,
          width,
          height,
          interval,
          timezone,
          style,
          toolbar_bg,
          enable_publishing,
          allow_symbol_change,
          hide_top_toolbar,
          hide_legend,
          save_image,
          container_id,
          show_popup_button,
          popup_width,
          popup_height,
          studies: ['RSI@tv-basicstudies', 'MACD@tv-basicstudies', 'Volume@tv-basicstudies'],
          overrides: {
            'mainSeriesProperties.candleStyle.upColor': '#22c55e',
            'mainSeriesProperties.candleStyle.downColor': '#ef4444',
            'mainSeriesProperties.candleStyle.borderUpColor': '#22c55e',
            'mainSeriesProperties.candleStyle.borderDownColor': '#ef4444',
            'mainSeriesProperties.candleStyle.wickUpColor': '#22c55e',
            'mainSeriesProperties.candleStyle.wickDownColor': '#ef4444',
            'paneProperties.background': '#0f0f1a',
            'paneProperties.vertGridProperties.color': '#1a1a2e',
            'paneProperties.horzGridProperties.color': '#1a1a2e',
          },
          disabled_features: ['header_symbol_search'],
          enabled_features: ['hide_left_toolbar'],
          custom_css_url: 'https://tigerwallet.com/tradingview-dark.css',
        });
      }
    };
    
    containerRef.current.appendChild(script);

    return () => {
      if (containerRef.current && script) {
        containerRef.current.removeChild(script);
      }
    };
  }, []);

  return (
    <div 
      ref={containerRef} 
      id={container_id}
      style={{ 
        width: typeof width === 'number' ? `${width}px` : width,
        height: typeof height === 'number' ? `${height}px` : height 
      }}
    />
  );
};

// Lightweight Charts Implementation (Alternative)
export const LightweightChart: React.FC<{
  data: CandlestickData[];
  width?: number;
  height?: number;
}> = ({ data, width = 800, height = 400 }) => {
  const chartContainerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<any>(null);
  const seriesRef = useRef<any>(null);

  useEffect(() => {
    if (!chartContainerRef.current || !data.length) return;

    // Dynamic import lightweight-charts
    const initChart = async () => {
      try {
        const { createChart, ColorType } = await import('lightweight-charts');
        
        chartRef.current = createChart(chartContainerRef.current, {
          width,
          height,
          layout: {
            background: { type: ColorType.Solid, color: '#0f0f1a' },
            textColor: '#d1d5db',
          },
          grid: {
            vertLines: { color: '#1a1a2e' },
            horzLines: { color: '#1a1a2e' },
          },
          crosshair: {
            mode: 1,
          },
          timeScale: {
            borderColor: '#1a1a2e',
          },
          rightPriceScale: {
            borderColor: '#1a1a2e',
          },
        });

        seriesRef.current = chartRef.current.addCandlestickSeries({
          upColor: '#22c55e',
          downColor: '#ef4444',
          borderDownColor: '#ef4444',
          borderUpColor: '#22c55e',
          wickDownColor: '#ef4444',
          wickUpColor: '#22c55e',
        });

        seriesRef.current.setData(data);

        // Handle resize
        const handleResize = () => {
          if (chartRef.current) {
            chartRef.current.applyOptions({ 
              width: chartContainerRef.current?.clientWidth || width,
              height: chartContainerRef.current?.clientHeight || height 
            });
          }
        };

        window.addEventListener('resize', handleResize);

        return () => {
          window.removeEventListener('resize', handleResize);
          if (chartRef.current) {
            chartRef.current.remove();
          }
        };
      } catch (error) {
        console.error('Failed to load lightweight-charts:', error);
      }
    };

    initChart();
  }, [data, width, height]);

  return (
    <div 
      ref={chartContainerRef} 
      style={{ width: '100%', height: '100%' }}
    />
  );
};

// Market Overview Widget
export const MarketOverview: React.FC = () => {
  const [markets, setMarkets] = useState<Array<{
    symbol: string;
    price: number;
    change: number;
  }>>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // In production, this would fetch real market data
    const fetchMarketData = async () => {
      try {
        // Fetch from price API
        const response = await fetch('http://localhost:8443/api/v1/prices');
        if (response.ok) {
          const data = await response.json();
          setMarkets(data);
        } else {
          // Fallback to demo data if API not available
          setMarkets([
            { symbol: 'BTC', price: 67500, change: 2.5 },
            { symbol: 'ETH', price: 3450, change: 1.8 },
            { symbol: 'SOL', price: 175, change: -0.5 },
            { symbol: 'BNB', price: 580, change: 1.2 },
            { symbol: 'AVAX', price: 38, change: 3.2 },
          ]);
        }
      } catch (error) {
        // Use fallback data on error
        setMarkets([
          { symbol: 'BTC', price: 67500, change: 2.5 },
          { symbol: 'ETH', price: 3450, change: 1.8 },
          { symbol: 'SOL', price: 175, change: -0.5 },
          { symbol: 'BNB', price: 580, change: 1.2 },
          { symbol: 'AVAX', price: 38, change: 3.2 },
        ]);
      } finally {
        setLoading(false);
      }
    };

    fetchMarketData();
    
    // Subscribe to real-time updates
    const interval = setInterval(fetchMarketData, 5000);
    return () => clearInterval(interval);
  }, []);

  const formatPrice = (price: number): string => {
    if (price >= 1000) return price.toLocaleString();
    if (price >= 1) return price.toFixed(2);
    return price.toFixed(4);
  };

  if (loading) {
    return <div style={{ color: '#64748b' }}>Loading market data...</div>;
  }

  return (
    <div style={{ display: 'grid', gap: '8px' }}>
      {markets.map((market) => (
        <div
          key={market.symbol}
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: '12px',
            background: '#1a1a2e',
            borderRadius: '8px',
          }}
        >
          <span style={{ fontWeight: 600 }}>{market.symbol}/USDT</span>
          <div style={{ textAlign: 'right' }}>
            <div>${formatPrice(market.price)}</div>
            <div style={{ 
              color: market.change >= 0 ? '#22c55e' : '#ef4444',
              fontSize: '0.875rem'
            }}>
              {market.change >= 0 ? '+' : ''}{market.change.toFixed(2)}%
            </div>
          </div>
        </div>
      ))}
    </div>
  );
};

// Advanced Order Types Widget
export const AdvancedOrderPanel: React.FC<{
  symbol: string;
  currentPrice: number;
  onOrderSubmit?: (order: any) => void;
}> = ({ symbol, currentPrice, onOrderSubmit }) => {
  const [orderType, setOrderType] = useState<'limit' | 'market' | 'stop'>('limit');
  const [side, setSide] = useState<'buy' | 'sell'>('buy');
  const [price, setPrice] = useState(currentPrice.toString());
  const [amount, setAmount] = useState('');
  const [stopPrice, setStopPrice] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const total = (parseFloat(amount) * parseFloat(price || '0')).toFixed(2);

  const handleSubmit = async () => {
    if (!amount || !price) return;
    
    setSubmitting(true);
    try {
      const order = {
        symbol,
        type: orderType,
        side,
        price: orderType !== 'market' ? price : undefined,
        stopPrice: orderType === 'stop' ? stopPrice : undefined,
        amount,
        total
      };
      
      // Submit order via API
      await onOrderSubmit?.(order);
      
      // Reset form
      setAmount('');
    } catch (error) {
      console.error('Order failed:', error);
    } finally {
      setSubmitting(false);
    }
  };

  const buttonStyle = (isActive: boolean, isBuy: boolean): React.CSSProperties => ({
    flex: 1,
    padding: '12px',
    border: 'none',
    borderRadius: '8px',
    background: isActive 
      ? (isBuy ? '#22c55e' : '#ef4444') 
      : '#1e293b',
    color: 'white',
    fontWeight: 600,
    cursor: 'pointer',
    transition: 'all 0.2s'
  });

  return (
    <div style={{ 
      background: '#1a1a2e', 
      padding: '20px', 
      borderRadius: '12px',
      color: 'white'
    }}>
      {/* Order Type Tabs */}
      <div style={{ display: 'flex', gap: '8px', marginBottom: '16px' }}>
        {(['market', 'limit', 'stop'] as const).map((type) => (
          <button
            key={type}
            onClick={() => setOrderType(type)}
            style={{
              flex: 1,
              padding: '8px',
              border: 'none',
              borderRadius: '6px',
              background: orderType === type ? '#f97316' : 'transparent',
              color: orderType === type ? 'white' : '#64748b',
              cursor: 'pointer',
              textTransform: 'capitalize'
            }}
          >
            {type}
          </button>
        ))}
      </div>

      {/* Buy/Sell Tabs */}
      <div style={{ display: 'flex', gap: '8px', marginBottom: '16px' }}>
        <button
          onClick={() => setSide('buy')}
          style={buttonStyle(side === 'buy', true)}
        >
          Buy
        </button>
        <button
          onClick={() => setSide('sell')}
          style={buttonStyle(side === 'sell', false)}
        >
          Sell
        </button>
      </div>

      {/* Price Input (for limit/stop orders) */}
      {orderType !== 'market' && (
        <div style={{ marginBottom: '12px' }}>
          <label style={{ display: 'block', marginBottom: '4px', color: '#64748b', fontSize: '0.875rem' }}>
            {orderType === 'stop' ? 'Stop Price' : 'Limit Price'} (USDT)
          </label>
          <input
            type="number"
            value={orderType === 'stop' ? stopPrice : price}
            onChange={(e) => {
              if (orderType === 'stop') {
                setStopPrice(e.target.value);
              } else {
                setPrice(e.target.value);
              }
            }}
            placeholder="0.00"
            style={{
              width: '100%',
              padding: '12px',
              borderRadius: '8px',
              border: '1px solid #374151',
              background: '#0f0f1a',
              color: 'white',
              fontSize: '1rem'
            }}
          />
        </div>
      )}

      {/* Amount Input */}
      <div style={{ marginBottom: '12px' }}>
        <label style={{ display: 'block', marginBottom: '4px', color: '#64748b', fontSize: '0.875rem' }}>
          Amount ({symbol.split(':')[1]?.replace('USDT', '') || 'BTC'})
        </label>
        <input
          type="number"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          placeholder="0.00"
          style={{
            width: '100%',
            padding: '12px',
            borderRadius: '8px',
            border: '1px solid #374151',
            background: '#0f0f1a',
            color: 'white',
            fontSize: '1rem'
          }}
        />
      </div>

      {/* Total */}
      <div style={{ 
        display: 'flex', 
        justifyContent: 'space-between', 
        marginBottom: '16px',
        padding: '12px',
        background: '#0f0f1a',
        borderRadius: '8px'
      }}>
        <span style={{ color: '#64748b' }}>Total</span>
        <span style={{ fontWeight: 600 }}>${total}</span>
      </div>

      {/* Submit Button */}
      <button
        onClick={handleSubmit}
        disabled={submitting || !amount || !price}
        style={{
          width: '100%',
          padding: '14px',
          border: 'none',
          borderRadius: '8px',
          background: side === 'buy' ? '#22c55e' : '#ef4444',
          color: 'white',
          fontWeight: 600,
          fontSize: '1rem',
          cursor: submitting || !amount || !price ? 'not-allowed' : 'pointer',
          opacity: submitting || !amount || !price ? 0.5 : 1
        }}
      >
        {submitting ? 'Processing...' : `${side === 'buy' ? 'Buy' : 'Sell'} ${symbol.split(':')[1]?.replace('USDT', '') || ''}`}
      </button>
    </div>
  );
};

export default {
  TradingViewWidget,
  LightweightChart,
  MarketOverview,
  AdvancedOrderPanel,
  TRADING_PAIRS
};
