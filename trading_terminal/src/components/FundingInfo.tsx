import React from 'react';

interface FundingInfoProps {
  fundingRate: string;
  nextFundingTime: number;
  symbol: string;
}

export const FundingInfo: React.FC<FundingInfoProps> = ({ fundingRate, nextFundingTime, symbol }) => {
  const rate = parseFloat(fundingRate);
  const nextFunding = new Date(nextFundingTime * 1000);
  const hoursUntil = Math.max(0, Math.floor((nextFunding.getTime() - Date.now()) / (1000 * 60 * 60)));
  const minutesUntil = Math.floor(((nextFunding.getTime() - Date.now()) % (1000 * 60 * 60)) / (1000 * 60));

  return (
    <div className="funding-info">
      <h3>Funding</h3>
      <div className="funding-rate">
        <span className="label">Funding Rate</span>
        <span className={`value ${rate >= 0 ? 'positive' : 'negative'}`}>
          {rate >= 0 ? '+' : ''}{(rate * 100).toFixed(4)}%
        </span>
      </div>
      <div className="next-funding">
        <span className="label">Next Funding</span>
        <span className="value">
          {hoursUntil}h {minutesUntil}m
        </span>
      </div>
      <div className="countdown">
        <div className="countdown-bar" style={{ width: `${(1 - (hoursUntil / 8)) * 100}%` }} />
      </div>
    </div>
  );
};