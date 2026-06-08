import React from 'react';
import { Position } from '../services/perpetuals';

interface PositionsTableProps {
  positions: Position[];
  onClose: (symbol: string, quantity?: string) => Promise<void>;
  onCancelOrder: (orderId: string) => Promise<void>;
}

export const PositionsTable: React.FC<PositionsTableProps> = ({ positions, onClose, onCancelOrder }) => {
  if (positions.length === 0) {
    return (
      <div className="positions-empty">
        <p>No open positions</p>
      </div>
    );
  }

  return (
    <div className="positions-table">
      <table>
        <thead>
          <tr>
            <th>Symbol</th>
            <th>Side</th>
            <th>Quantity</th>
            <th>Entry Price</th>
            <th>Mark Price</th>
            <th>Leverage</th>
            <th>Unrealized P&L</th>
            <th>ROE</th>
            <th>Liq. Price</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {positions.map((pos) => {
            const pnl = parseFloat(pos.unrealizedPnl);
            const roe = parseFloat(pos.roe);
            
            return (
              <tr key={pos.id} className={pnl >= 0 ? 'profit' : 'loss'}>
                <td>{pos.symbol}</td>
                <td className={pos.side.toLowerCase()}>{pos.side}</td>
                <td>{parseFloat(pos.quantity).toFixed(4)}</td>
                <td>${parseFloat(pos.entryPrice).toFixed(2)}</td>
                <td>${parseFloat(pos.markPrice).toFixed(2)}</td>
                <td>{pos.leverage}x</td>
                <td className={pnl >= 0 ? 'profit' : 'loss'}>
                  ${pnl.toFixed(2)}
                </td>
                <td className={roe >= 0 ? 'profit' : 'loss'}>
                  {roe.toFixed(2)}%
                </td>
                <td>${parseFloat(pos.liquidationPrice).toFixed(2)}</td>
                <td>
                  <button
                    className="close-btn"
                    onClick={() => onClose(pos.symbol)}
                  >
                    Close
                  </button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};