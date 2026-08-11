import React, { useEffect, useRef, useState } from 'react';

interface TradingChartProps {
  symbol: string;
  currentPrice: number;
}

export const TradingChart: React.FC<TradingChartProps> = ({ symbol, currentPrice }) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [chartData, setChartData] = useState<{ time: number; price: number }[]>([]);

  useEffect(() => {
    // Fetch real OHLC/price history from the backend price feed
    // (GET /api/v1/price?symbol=...). On failure, show an empty chart rather
    // than fabricated Math.random() data.
    let cancelled = false;
    const fetchData = async () => {
      try {
        const res = await fetch(`http://localhost:8443/api/v1/price?symbol=${encodeURIComponent(symbol)}`);
        if (!res.ok) {
          if (!cancelled) setChartData([]);
          return;
        }
        const json = await res.json();
        const history: Array<{ time: number; price: number }> =
          Array.isArray(json.history)
            ? json.history.map((p: { time: number; price: number }) => ({ time: p.time, price: Number(p.price) }))
            : [];
        if (!cancelled) setChartData(history);
      } catch {
        if (!cancelled) setChartData([]);
      }
    };
    fetchData();
    return () => { cancelled = true; };
  }, [symbol, currentPrice]);
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas || chartData.length === 0) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const width = canvas.width;
    const height = canvas.height;
    
    // Clear canvas
    ctx.fillStyle = '#1a1a2e';
    ctx.fillRect(0, 0, width, height);
    
    // Calculate price range
    const prices = chartData.map(d => d.price);
    const minPrice = Math.min(...prices);
    const maxPrice = Math.max(...prices);
    const priceRange = maxPrice - minPrice || 1;
    
    // Draw grid
    ctx.strokeStyle = '#2a2a4e';
    ctx.lineWidth = 1;
    
    for (let i = 0; i <= 5; i++) {
      const y = (i / 5) * height;
      ctx.beginPath();
      ctx.moveTo(0, y);
      ctx.lineTo(width, y);
      ctx.stroke();
    }
    
    // Draw price line
    const isUp = chartData[chartData.length - 1].price >= chartData[0].price;
    ctx.strokeStyle = isUp ? '#00d4aa' : '#ff4757';
    ctx.lineWidth = 2;
    ctx.beginPath();
    
    chartData.forEach((point, i) => {
      const x = (i / (chartData.length - 1)) * width;
      const y = height - ((point.price - minPrice) / priceRange) * height;
      
      if (i === 0) {
        ctx.moveTo(x, y);
      } else {
        ctx.lineTo(x, y);
      }
    });
    
    ctx.stroke();
    
    // Fill area under the line
    ctx.lineTo(width, height);
    ctx.lineTo(0, height);
    ctx.closePath();
    
    const gradient = ctx.createLinearGradient(0, 0, 0, height);
    if (isUp) {
      gradient.addColorStop(0, 'rgba(0, 212, 170, 0.3)');
      gradient.addColorStop(1, 'rgba(0, 212, 170, 0)');
    } else {
      gradient.addColorStop(0, 'rgba(255, 71, 87, 0.3)');
      gradient.addColorStop(1, 'rgba(255, 71, 87, 0)');
    }
    
    ctx.fillStyle = gradient;
    ctx.fill();
    
    // Draw current price line
    const currentY = height - ((currentPrice - minPrice) / priceRange) * height;
    ctx.strokeStyle = isUp ? '#00d4aa' : '#ff4757';
    ctx.setLineDash([5, 5]);
    ctx.beginPath();
    ctx.moveTo(0, currentY);
    ctx.lineTo(width, currentY);
    ctx.stroke();
    ctx.setLineDash([]);
    
    // Draw current price label
    ctx.fillStyle = isUp ? '#00d4aa' : '#ff4757';
    ctx.font = 'bold 14px monospace';
    ctx.fillText(`$${currentPrice.toLocaleString()}`, 10, currentY - 10);
    
  }, [chartData, currentPrice]);

  return (
    <div className="trading-chart">
      <canvas 
        ref={canvasRef}
        width={800}
        height={400}
      />
    </div>
  );
};