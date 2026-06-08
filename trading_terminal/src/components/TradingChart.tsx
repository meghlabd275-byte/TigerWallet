import React, { useEffect, useRef, useState } from 'react';

interface TradingChartProps {
  symbol: string;
  currentPrice: number;
}

export const TradingChart: React.FC<TradingChartProps> = ({ symbol, currentPrice }) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [chartData, setChartData] = useState<{ time: number; price: number }[]>([]);

  useEffect(() => {
    // Generate sample chart data
    const data: { time: number; price: number }[] = [];
    let price = currentPrice * 0.95;
    const now = Date.now();
    
    for (let i = 100; i >= 0; i--) {
      data.push({
        time: now - i * 60000,
        price: price + (Math.random() - 0.5) * currentPrice * 0.02
      });
    }
    
    setChartData(data);
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