/**
 * Fee Management
 */

import React, { useState } from 'react';

function FeeManagement() {
  const [activeTab, setActiveTab] = useState('trading');

  const fees = {
    trading: { maker: '0.1%', taker: '0.2%', volumeDiscount: true },
    withdrawal: { eth: '0.005 ETH', btc: '0.0005 BTC', usdt: '1 USDT' },
    deposit: { enabled: false, fee: '0' },
    swap: { fee: '0.3%', router: 'Uniswap V3' },
    bridge: { fee: '0.5%', enabled: true },
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Fee Management</h1>

      <div className="flex gap-2 mb-6">
        {['Trading', 'Withdrawal', 'Deposit', 'Swap', 'Bridge'].map(tab => (
          <button key={tab} onClick={() => setActiveTab(tab.toLowerCase())} className={`px-4 py-2 rounded-lg ${activeTab === tab.toLowerCase() ? 'bg-amber-500 text-black' : 'bg-slate-800'}`}>
            {tab}
          </button>
        ))}
      </div>

      {activeTab === 'trading' && (
        <div className="space-y-6">
          <div className="bg-slate-800 p-6 rounded-lg">
            <h3 className="font-semibold mb-4">Trading Fees</h3>
            <div className="grid grid-cols-2 gap-4 max-w-md">
              <div>
                <label className="label">Maker Fee (%)</label>
                <input type="number" className="input" defaultValue={fees.trading.maker.replace('%', '')} step="0.01" />
              </div>
              <div>
                <label className="label">Taker Fee (%)</label>
                <input type="number" className="input" defaultValue={fees.trading.taker.replace('%', '')} step="0.01" />
              </div>
            </div>
            <div className="flex items-center gap-2 mt-4">
              <input type="checkbox" defaultChecked id="volumeDiscount" />
              <label htmlFor="volumeDiscount">Enable Volume Discounts</label>
            </div>
            <button className="btn btn-primary mt-4">Save</button>
          </div>
        </div>
      )}

      {activeTab === 'withdrawal' && (
        <div className="space-y-6">
          <div className="bg-slate-800 p-6 rounded-lg">
            <h3 className="font-semibold mb-4">Withdrawal Fees</h3>
            <div className="space-y-4 max-w-md">
              <div>
                <label className="label">Ethereum (ETH)</label>
                <input type="text" className="input" defaultValue={fees.withdrawal.eth} />
              </div>
              <div>
                <label className="label">Bitcoin (BTC)</label>
                <input type="text" className="input" defaultValue={fees.withdrawal.btc} />
              </div>
              <div>
                <label className="label">Tether (USDT)</label>
                <input type="text" className="input" defaultValue={fees.withdrawal.usdt} />
              </div>
            </div>
            <button className="btn btn-primary mt-4">Save</button>
          </div>
        </div>
      )}

      {activeTab === 'deposit' && (
        <div className="space-y-6">
          <div className="bg-slate-800 p-6 rounded-lg">
            <h3 className="font-semibold mb-4">Deposit Fees</h3>
            <div className="flex items-center gap-2 mb-4">
              <input type="checkbox" defaultChecked={fees.deposit.enabled} id="depositFee" />
              <label htmlFor="depositFee">Enable Deposit Fee</label>
            </div>
            <div className="max-w-md">
              <label className="label">Fee Amount (%)</label>
              <input type="number" className="input" defaultValue={fees.deposit.fee} step="0.01" />
            </div>
            <button className="btn btn-primary mt-4">Save</button>
          </div>
        </div>
      )}

      {activeTab === 'swap' && (
        <div className="space-y-6">
          <div className="bg-slate-800 p-6 rounded-lg">
            <h3 className="font-semibold mb-4">Swap Fees</h3>
            <div className="space-y-4 max-w-md">
              <div>
                <label className="label">Swap Fee (%)</label>
                <input type="number" className="input" defaultValue={fees.swap.fee.replace('%', '')} step="0.01" />
              </div>
              <div>
                <label className="label">Router</label>
                <select className="input" defaultValue={fees.swap.router}>
                  <option>Uniswap V3</option>
                  <option>Curve</option>
                  <option>Balancer</option>
                  <option>1inch</option>
                </select>
              </div>
            </div>
            <button className="btn btn-primary mt-4">Save</button>
          </div>
        </div>
      )}

      {activeTab === 'bridge' && (
        <div className="space-y-6">
          <div className="bg-slate-800 p-6 rounded-lg">
            <h3 className="font-semibold mb-4">Bridge Fees</h3>
            <div className="flex items-center gap-2 mb-4">
              <input type="checkbox" defaultChecked={fees.bridge.enabled} id="bridgeFee" />
              <label htmlFor="bridgeFee">Enable Bridge</label>
            </div>
            <div className="max-w-md">
              <label className="label">Bridge Fee (%)</label>
              <input type="number" className="input" defaultValue={fees.bridge.fee.replace('%', '')} step="0.01" />
            </div>
            <button className="btn btn-primary mt-4">Save</button>
          </div>
        </div>
      )}
    </div>
  );
}

export default FeeManagement;
