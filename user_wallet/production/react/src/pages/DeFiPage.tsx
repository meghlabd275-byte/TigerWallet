/**
 * DeFi Page - single hub with tabbed sections.
 *
 * Tabs: Lending, Copy Trading, DAO, Perpetual, Margin, Prediction,
 * Launchpool, Token Sales. Each tab fetches live data from the canonical
 * backend via WalletService and exposes a simple action form. No mock data.
 *
 * Method names match WalletService exactly:
 *   getLendingMarkets, lendingSupply
 *   getCopyTraders, followTrader
 *   getDaoProposals, voteDaoProposal
 *   getPerpetualPositions, createPerpetualPosition, closePerpetualPosition
 *   getMarginPositions, createMarginPosition, closeMarginPosition
 *   getPredictionMarkets, placePredictionBet
 *   getLaunchpool, launchpoolStake
 *   getTokenSales, participateTokenSale
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { useWallet } from '../contexts/WalletContext';
import { WalletService } from '../services/WalletService';
import LoadingSpinner from '../components/LoadingSpinner';

type TabId =
  | 'lending'
  | 'copy'
  | 'dao'
  | 'perpetual'
  | 'margin'
  | 'prediction'
  | 'launchpool'
  | 'sales';

const TABS: { id: TabId; label: string }[] = [
  { id: 'lending', label: 'Lending' },
  { id: 'copy', label: 'Copy Trading' },
  { id: 'dao', label: 'DAO' },
  { id: 'perpetual', label: 'Perpetual' },
  { id: 'margin', label: 'Margin' },
  { id: 'prediction', label: 'Prediction' },
  { id: 'launchpool', label: 'Launchpool' },
  { id: 'sales', label: 'Token Sales' },
];

type Row = Record<string, unknown>;

function asRows(data: unknown): Row[] {
  if (Array.isArray(data)) return data as Row[];
  if (data && typeof data === 'object') {
    const obj = data as Record<string, unknown>;
    for (const k of ['markets', 'traders', 'proposals', 'positions', 'pools', 'sales', 'items', 'data']) {
      if (Array.isArray(obj[k])) return obj[k] as Row[];
    }
  }
  return [];
}

function str(v: unknown, fallback = ''): string {
  return v === undefined || v === null ? fallback : String(v);
}

function DeFiPage() {
  const { theme } = useTheme();
  const { activeWallet } = useWallet();
  const [walletService] = useState(() => new WalletService());

  const [activeTab, setActiveTab] = useState<TabId>('lending');
  const [rows, setRows] = useState<Row[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Shared action-form state (reused per tab)
  const [actionA, setActionA] = useState('');
  const [actionB, setActionB] = useState('');
  const [actionC, setActionC] = useState('');
  const [password, setPassword] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [actionMessage, setActionMessage] = useState<string | null>(null);

  const chainId = activeWallet?.chain?.chainId ?? 1;

  const resetForm = () => {
    setActionA('');
    setActionB('');
    setActionC('');
    setPassword('');
    setActionMessage(null);
  };

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      let data: unknown;
      switch (activeTab) {
        case 'lending': data = await walletService.getLendingMarkets(); break;
        case 'copy': data = await walletService.getCopyTraders(); break;
        case 'dao': data = await walletService.getDaoProposals(); break;
        case 'perpetual': data = await walletService.getPerpetualPositions(); break;
        case 'margin': data = await walletService.getMarginPositions(); break;
        case 'prediction': data = await walletService.getPredictionMarkets(); break;
        case 'launchpool': data = await walletService.getLaunchpool(); break;
        case 'sales': data = await walletService.getTokenSales(); break;
      }
      setRows(asRows(data));
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to load data');
      setRows([]);
    } finally {
      setLoading(false);
    }
  }, [walletService, activeTab]);

  useEffect(() => {
    resetForm();
    load();
  }, [load]);

  const fail = (msg: string) => {
    setError(msg);
    setSubmitting(false);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setActionMessage(null);

    const walletId = activeWallet?.id ?? '';
    if ((activeTab === 'lending' || activeTab === 'launchpool') && !walletId) {
      return fail('No active wallet selected');
    }
    if ((activeTab === 'lending' || activeTab === 'launchpool') && !password) {
      return fail('Wallet password is required');
    }

    setSubmitting(true);
    try {
      let result: unknown;
      switch (activeTab) {
        case 'lending':
          if (!actionA.trim() || !actionB.trim()) return fail('Asset and amount are required');
          result = await walletService.lendingSupply({
            walletId, password, asset: actionA.trim(), amount: actionB.trim(), chainId,
          });
          break;
        case 'copy':
          if (!actionA.trim() || !actionB.trim()) return fail('Trader ID and allocation are required');
          result = await walletService.followTrader({
            traderId: actionA.trim(), allocation: actionB.trim(),
          });
          break;
        case 'dao':
          if (!actionA.trim()) return fail('Proposal ID is required');
          result = await walletService.voteDaoProposal({
            proposalId: actionA.trim(), support: actionB === 'for',
          });
          break;
        case 'perpetual':
          if (!actionA.trim() || !actionB.trim() || !actionC.trim()) return fail('Pair, side and size are required');
          result = await walletService.createPerpetualPosition({
            pair: actionA.trim(), side: actionB.trim(), size: actionC.trim(),
            leverage: 1, chainId,
          });
          break;
        case 'margin':
          if (!actionA.trim() || !actionB.trim() || !actionC.trim()) return fail('Pair, side and size are required');
          result = await walletService.createMarginPosition({
            pair: actionA.trim(), side: actionB.trim(), size: actionC.trim(),
            leverage: 1, chainId,
          });
          break;
        case 'prediction':
          if (!actionA.trim() || !actionB.trim() || !actionC.trim()) return fail('Market ID, side and amount are required');
          result = await walletService.placePredictionBet({
            marketId: actionA.trim(), side: actionB.trim(), amount: actionC.trim(),
          });
          break;
        case 'launchpool':
          if (!actionB.trim()) return fail('Amount is required');
          result = await walletService.launchpoolStake({
            walletId, password, amount: actionB.trim(),
          });
          break;
        case 'sales':
          if (!actionA.trim() || !actionB.trim()) return fail('Sale ID and amount are required');
          result = await walletService.participateTokenSale({
            saleId: actionA.trim(), amount: actionB.trim(),
          });
          break;
      }
      setActionMessage(typeof result === 'string' ? result : 'Action submitted.');
      resetForm();
      await load();
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Action failed');
    } finally {
      setSubmitting(false);
    }
  };

  const handleClosePosition = async (id: string, kind: 'perpetual' | 'margin') => {
    setError(null);
    setSubmitting(true);
    try {
      if (kind === 'perpetual') await walletService.closePerpetualPosition(id);
      else await walletService.closeMarginPosition(id);
      await load();
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to close position');
    } finally {
      setSubmitting(false);
    }
  };

  const cardClass = `card mb-6 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`;
  const inputClass = `input w-full ${theme === 'dark' ? 'bg-slate-900 border-slate-700' : 'bg-white'}`;
  const rowCard = `card ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`;

  // Primary display field for a row, per tab
  const rowTitle = (r: Row): string => {
    switch (activeTab) {
      case 'lending': return str(r.asset ?? r.symbol, 'Asset');
      case 'copy': return str(r.name ?? r.trader_id ?? r.address, 'Trader');
      case 'dao': return str(r.title ?? r.id, 'Proposal');
      case 'perpetual':
      case 'margin': return str(r.pair ?? r.id, 'Position');
      case 'prediction': return str(r.question ?? r.title ?? r.id, 'Market');
      case 'launchpool': return str(r.name ?? r.token ?? r.id, 'Pool');
      case 'sales': return str(r.name ?? r.token ?? r.id, 'Sale');
    }
  };

  const rowSubtitle = (r: Row): string => {
    switch (activeTab) {
      case 'lending': return `Supply APY: ${str(r.supply_apy ?? r.apy, '—')} • Borrow APY: ${str(r.borrow_apy, '—')}`;
      case 'copy': return `ROI: ${str(r.roi ?? r.apy, '—')} • Followers: ${str(r.followers, '—')}`;
      case 'dao': return `For: ${str(r.for ?? r.yes, '0')} • Against: ${str(r.against ?? r.no, '0')}`;
      case 'perpetual':
      case 'margin': return `${str(r.side, '—')} • Size: ${str(r.size, '—')} • Leverage: ${str(r.leverage, '—')}x`;
      case 'prediction': return `Yes: ${str(r.yes_price ?? r.yes, '—')} • No: ${str(r.no_price ?? r.no, '—')}`;
      case 'launchpool': return `APY: ${str(r.apy, '—')} • Total staked: ${str(r.total_staked, '—')}`;
      case 'sales': return `Price: ${str(r.price, '—')} • Raised: ${str(r.raised, '—')}`;
    }
  };

  const rowId = (r: Row, i: number): string =>
    str(r.id ?? r.proposal_id ?? r.market_id ?? r.sale_id ?? r.position_id ?? r.trader_id ?? i);

  const renderActionForm = () => {
    const labelA: Record<TabId, string> = {
      lending: 'Asset', copy: 'Trader ID', dao: 'Proposal ID',
      perpetual: 'Pair', margin: 'Pair', prediction: 'Market ID',
      launchpool: 'Sale ID', sales: 'Sale ID',
    };
    const labelB: Record<TabId, string> = {
      lending: 'Amount', copy: 'Allocation', dao: 'Vote',
      perpetual: 'Side', margin: 'Side', prediction: 'Side',
      launchpool: 'Amount', sales: 'Amount',
    };
    const labelC: Record<TabId, string> = {
      lending: '', copy: '', dao: '',
      perpetual: 'Size', margin: 'Size', prediction: 'Amount',
      launchpool: '', sales: '',
    };
    const needsC = labelC[activeTab] !== '';

    return (
      <form onSubmit={handleSubmit} className={cardClass}>
        <h3 className="font-semibold mb-4">New {TABS.find((t) => t.id === activeTab)?.label} Action</h3>
        <div className="grid grid-cols-2 gap-4 mb-4">
          <div>
            <label className="label">{labelA[activeTab]}</label>
            <input type="text" value={actionA} onChange={(e) => setActionA(e.target.value)} className={inputClass} required />
          </div>
          <div>
            <label className="label">{labelB[activeTab]}</label>
            {activeTab === 'dao' ? (
              <select value={actionB} onChange={(e) => setActionB(e.target.value)} className={inputClass}>
                <option value="for">For</option>
                <option value="against">Against</option>
              </select>
            ) : (
              <input type="text" value={actionB} onChange={(e) => setActionB(e.target.value)} className={inputClass} required />
            )}
          </div>
        </div>
        {needsC && (
          <div className="mb-4">
            <label className="label">{labelC[activeTab]}</label>
            <input type="text" value={actionC} onChange={(e) => setActionC(e.target.value)} className={inputClass} required />
          </div>
        )}
        {(activeTab === 'lending' || activeTab === 'launchpool') && (
          <div className="mb-6">
            <label className="label">Wallet Password</label>
            <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} className={inputClass} required />
          </div>
        )}
        {(activeTab === 'perpetual' || activeTab === 'margin') && !needsC && <div className="mb-6" />}
        <button type="submit" disabled={submitting} className="btn btn-primary w-full">
          {submitting ? 'Submitting...' : 'Submit'}
        </button>
      </form>
    );
  };

  return (
    <div className="p-6 max-w-3xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">DeFi Hub</h1>

      {/* Tabs */}
      <div className="flex gap-2 mb-6 overflow-x-auto">
        {TABS.map((t) => (
          <button
            key={t.id}
            onClick={() => setActiveTab(t.id)}
            className={`px-4 py-2 rounded-lg text-sm whitespace-nowrap ${
              activeTab === t.id ? 'bg-amber-500 text-black' : theme === 'dark' ? 'bg-slate-800' : 'bg-gray-200'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {error && (
        <div className={`card mb-6 ${theme === 'dark' ? 'bg-red-900/30' : 'bg-red-50'}`}>
          <p className="text-sm text-red-500">{error}</p>
        </div>
      )}

      {actionMessage && (
        <div className={`card mb-6 ${theme === 'dark' ? 'bg-green-900/30' : 'bg-green-50'}`}>
          <p className="text-sm text-green-500">{actionMessage}</p>
        </div>
      )}

      {renderActionForm()}

      <h3 className="font-semibold mb-3">{TABS.find((t) => t.id === activeTab)?.label}</h3>
      {loading ? (
        <LoadingSpinner label="Loading..." />
      ) : rows.length === 0 ? (
        <div className={`card text-center py-12 opacity-60 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
          No {TABS.find((t) => t.id === activeTab)?.label.toLowerCase()} data available.
        </div>
      ) : (
        <div className="space-y-3">
          {rows.map((r, i) => {
            const id = rowId(r, i);
            return (
              <div key={id} className={rowCard}>
                <div className="flex justify-between items-start gap-3">
                  <div className="min-w-0">
                    <div className="font-semibold">{rowTitle(r)}</div>
                    <p className="text-xs opacity-60 mt-1">{rowSubtitle(r)}</p>
                  </div>
                  {(activeTab === 'perpetual' || activeTab === 'margin') && (
                    <button
                      onClick={() => handleClosePosition(id, activeTab)}
                      disabled={submitting}
                      className="btn btn-secondary text-sm whitespace-nowrap"
                    >
                      Close
                    </button>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

export default DeFiPage;
