'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Typography, Card, CardContent, Button, TextField,
  CircularProgress, Alert, IconButton,
  Dialog, DialogTitle, DialogContent, List, ListItemButton,
  ListItemText, Avatar
} from '@mui/material';
import {
  SwapHoriz, Settings, ArrowDropDown,
  OpenInNew, Shield, Speed, CompareArrows
} from '@mui/icons-material';
import { useTheme } from '../components/ThemeProvider';

// Same-origin API base: the Next.js app proxies /api/v1/* to the backend
// services (see app/api/v1/_proxy.ts). In the browser this resolves to the
// current host, so no CORS and no hardcoded port.
const API_BASE = process.env.NEXT_PUBLIC_API_URL || '';

interface SwapToken {
  address: string;
  symbol: string;
  name: string;
  decimals: number;
  chainId: number;
  isNative?: boolean;
  isStable?: boolean;
  priceUsd?: number;
  logoUri?: string;
}

interface SwapQuote {
  inputToken: string;
  outputToken: string;
  inputAmount: number;
  outputAmount: number;
  minimumOut: number;
  priceImpact: number;
  gasEstimate: number;
  gasFeeUsd: number;
  exchangeRate: number;
  expiresAt: number;
  route: { dex: string; fee: number; amountIn: number; amountOut: number }[];
}

const CHAIN_CONFIG: Record<number, { name: string; explorer: string }> = {
  1: { name: 'Ethereum', explorer: 'https://etherscan.io' },
  56: { name: 'BNB Chain', explorer: 'https://bscscan.com' },
  42161: { name: 'Arbitrum', explorer: 'https://arbiscan.io' },
  137: { name: 'Polygon', explorer: 'https://polygonscan.com' },
  10: { name: 'Optimism', explorer: 'https://optimistic.etherscan.io' },
  8453: { name: 'Base', explorer: 'https://basescan.org' },
};

export default function SwapPage() {
  const { isDark } = useTheme();
  const [chainId, setChainId] = useState(1);
  const [tokenIn, setTokenIn] = useState<SwapToken | null>(null);
  const [tokenOut, setTokenOut] = useState<SwapToken | null>(null);
  const [amountIn, setAmountIn] = useState('');
  const [amountOut, setAmountOut] = useState('');
  const [tokens, setTokens] = useState<SwapToken[]>([]);
  const [quote, setQuote] = useState<SwapQuote | null>(null);
  const [loadingQuote, setLoadingQuote] = useState(false);
  const [loadingTx, setLoadingTx] = useState(false);
  const [slippage, setSlippage] = useState(0.5);
  const [showTokenSelector, setShowTokenSelector] = useState<'in' | 'out' | null>(null);
  const [showSettings, setShowSettings] = useState(false);
  const [walletAddress, setWalletAddress] = useState('');
  const [txHash, setTxHash] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    const savedWallet = localStorage.getItem('tigerwallet_address');
    if (savedWallet) setWalletAddress(savedWallet);
  }, []);

  const setDefaultTokens = () => {
    const defaultTokens: SwapToken[] = [
      { address: '0x0000000000000000000000000000000000000000', symbol: 'ETH', name: 'Ethereum', decimals: 18, chainId: 1, isNative: true, priceUsd: 3500, logoUri: 'https://assets.coingecko.com/coins/images/279/small/ethereum.png' },
      { address: '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48', symbol: 'USDC', name: 'USD Coin', decimals: 6, chainId: 1, isStable: true, priceUsd: 1, logoUri: 'https://assets.coingecko.com/coins/images/6319/small/usdc.png' },
      { address: '0xdAC17F958D2ee523a2206206994597C13D831ec7', symbol: 'USDT', name: 'Tether USD', decimals: 6, chainId: 1, isStable: true, priceUsd: 1, logoUri: 'https://assets.coingecko.com/coins/images/325/small/Tether.png' },
      { address: '0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599', symbol: 'WBTC', name: 'Wrapped Bitcoin', decimals: 8, chainId: 1, priceUsd: 65000, logoUri: 'https://assets.coingecko.com/coins/images/7598/small/wrapped_bitcoin_wbtc.png' },
    ];
    setTokens(defaultTokens);
    if (!tokenIn) setTokenIn(defaultTokens[0]);
    if (!tokenOut) setTokenOut(defaultTokens[1]);
  };

  const fetchTokens = useCallback(async () => {
    try {
      // Backend (go/swap_service) filters tokens by `chain` (string name).
      const response = await fetch(`${API_BASE}/api/v1/swap/tokens?chain_id=${chainId}`);
      if (response.ok) {
        const data = await response.json();
        if (data.tokens && data.tokens.length > 0) {
          // Map backend Token fields to the frontend SwapToken interface.
          const mapped: SwapToken[] = data.tokens.map((t: any) => ({
            address: t.contract || '',
            symbol: t.symbol,
            name: t.name,
            decimals: t.decimals,
            chainId: chainId,
            isNative: t.symbol === 'ETH' || t.symbol === 'WETH',
            isStable: ['USDT', 'USDC', 'DAI'].includes(t.symbol),
            priceUsd: parseFloat(t.price_usd) || 0,
            logoUri: undefined,
          }));
          setTokens(mapped);
          if (!tokenIn) setTokenIn(mapped[0]);
          if (!tokenOut && mapped.length > 1) setTokenOut(mapped[1]);
        } else {
          setDefaultTokens();
        }
      } else {
        setDefaultTokens();
      }
    } catch (err) {
      setDefaultTokens();
    }
  }, [chainId, tokenIn, tokenOut]);

  useEffect(() => {
    fetchTokens();
  }, [fetchTokens]);

  const fetchQuote = useCallback(async () => {
    if (!tokenIn || !tokenOut || !amountIn || parseFloat(amountIn) <= 0) {
      setQuote(null);
      setAmountOut('');
      return;
    }

    setLoadingQuote(true);
    try {
      const response = await fetch(
        `${API_BASE}/api/v1/swap/quote?token_in=${tokenIn.symbol}&token_out=${tokenOut.symbol}&amount=${amountIn}&chain_id=${chainId}`
      );
      if (response.ok) {
        const data = await response.json();
        // Map backend quote (to_amount, min_received, price_impact, gas_estimate)
        // to the frontend SwapQuote interface.
        const outputAmount = parseFloat(data.to_amount) || 0;
        const inputAmount = parseFloat(amountIn) || 0;
        const minReceived = parseFloat(data.min_received) || outputAmount;
        setQuote({
          inputToken: tokenIn.symbol,
          outputToken: tokenOut.symbol,
          inputAmount,
          outputAmount,
          minimumOut: minReceived,
          priceImpact: parseFloat(data.price_impact) || 0,
          gasEstimate: parseFloat(data.gas_estimate) || 0,
          gasFeeUsd: 0,
          exchangeRate: inputAmount > 0 ? outputAmount / inputAmount : 0,
          expiresAt: Date.now() + 30000,
          route: [{ dex: data.dex || 'DEX', fee: 3000, amountIn: inputAmount, amountOut: outputAmount }],
        });
        setAmountOut(outputAmount.toFixed(6));
      } else {
        // Backend quote failed: show no quote rather than fabricate one.
        setQuote(null);
        setAmountOut('');
        setError('Unable to fetch quote from swap service');
      }
    } catch (err) {
      setQuote(null);
      setAmountOut('');
      setError('Unable to reach swap service');
    } finally {
      setLoadingQuote(false);
    }
  }, [tokenIn, tokenOut, amountIn, chainId]);

  useEffect(() => {
    fetchQuote();
  }, [fetchQuote]);

  const handleSwapTokens = () => {
    const temp = tokenIn;
    setTokenIn(tokenOut);
    setTokenOut(temp);
    setAmountIn(amountOut);
    setAmountOut('');
    setQuote(null);
  };

  const handleSwap = async () => {
    if (!walletAddress || !quote) {
      setError('Please connect wallet first');
      return;
    }
    // ---- Client-side input validation (defense in depth; server validates
    // slippage/amount too) ----
    const slippageNum = Number(slippage);
    if (!Number.isFinite(slippageNum) || slippageNum <= 0) {
      setError('Slippage tolerance must be greater than 0%');
      return;
    }
    if (slippageNum > 50) {
      setError('Slippage tolerance above 50% is unsafe. Use a smaller value.');
      return;
    }
    const inAmount = parseFloat(amountIn);
    if (!Number.isFinite(inAmount) || inAmount <= 0) {
      setError('Enter a valid positive amount to swap');
      return;
    }
    if (inAmount > 1e9) {
      // Sanity cap to reject fat-finger / overflow inputs before they reach the
      // backend. The backend re-validates, but failing early gives a clearer
      // message and avoids building a huge unsigned-integer amount.
      setError('Amount is unreasonably large; please check your input');
      return;
    }

    setLoadingTx(true);
    setError(null);
    setSuccess(null);

    try {
      const authToken = localStorage.getItem('tigerwallet_token');
      const headers: HeadersInit = { 'Content-Type': 'application/json' };
      if (authToken) headers['Authorization'] = `Bearer ${authToken}`;

      const chainName = CHAIN_CONFIG[chainId]?.name?.toLowerCase() || 'ethereum';
      const response = await fetch(`${API_BASE}/api/v1/swap/execute`, {
        method: 'POST',
        headers,
        body: JSON.stringify({
          // Match go/swap_service SwapRequest field names.
          user_id: walletAddress,
          from_token: tokenIn?.symbol,
          to_token: tokenOut?.symbol,
          from_amount: amountIn,
          min_output: String(quote.minimumOut),
          slippage: String(slippage),
          chain: chainName,
        }),
      });

      const data = await response.json();
      if (data.success) {
        // The swap service computes the quote/route and returns the on-chain
        // action the client must submit through the signing backend
        // (wallet_api /api/v1/send). It does NOT fabricate a tx hash.
        if (data.tx_hash) {
          setSuccess(`Swap broadcast! TX: ${data.tx_hash}`);
          setTxHash(data.tx_hash);
        } else if (data.action_required) {
          setSuccess(
            `Quote ready (swap ${data.swap_id}). ${data.action_required.description} ` +
            `Next: submit via ${data.action_required.endpoint}.`
          );
          setTxHash('');
        } else {
          setSuccess(`Swap quote ready (swap ${data.swap_id}). Status: ${data.status}`);
        }
        setAmountIn('');
        setAmountOut('');
        setQuote(null);
      } else {
        setError(data.error || 'Swap failed');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Swap failed');
    } finally {
      setLoadingTx(false);
    }
  };

  return (
    <div className={`min-h-screen p-6 ${isDark ? 'bg-slate-900 text-slate-50' : 'bg-slate-50 text-slate-900'}`}>
      <header className="mb-8">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <span className="text-4xl">🔄</span>
            <h1 className="text-2xl font-bold">Swap</h1>
          </div>
          <div className="flex gap-2">
            <IconButton onClick={() => setShowSettings(true)} className={isDark ? 'text-gray-400' : 'text-gray-500'}>
              <Settings />
            </IconButton>
          </div>
        </div>
      </header>

      <div className="max-w-lg mx-auto">
        <div className="mb-4">
          <select
            value={chainId}
            onChange={(e) => setChainId(parseInt(e.target.value))}
            className={`w-full p-3 rounded-lg border ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}
          >
            {Object.entries(CHAIN_CONFIG).map(([id, config]) => (
              <option key={id} value={id}>{config.name}</option>
            ))}
          </select>
        </div>

        <Card className="mb-4">
          <CardContent sx={{ p: 3 }}>
            <div className={`rounded-xl p-4 mb-2 ${isDark ? 'bg-slate-800' : 'bg-slate-100'}`}>
              <div className="flex justify-between mb-2">
                <span className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>You Pay</span>
                <span className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Balance: --</span>
              </div>
              <div className="flex gap-2">
                <input
                  type="number"
                  value={amountIn}
                  onChange={(e) => setAmountIn(e.target.value)}
                  placeholder="0.00"
                  className={`flex-1 bg-transparent text-2xl font-semibold outline-none ${isDark ? 'text-white' : 'text-gray-900'}`}
                />
                <button
                  onClick={() => setShowTokenSelector('in')}
                  className={`flex items-center gap-2 px-3 py-2 rounded-lg ${isDark ? 'bg-slate-700 text-white' : 'bg-slate-200 text-gray-900'}`}
                >
                  {tokenIn?.symbol || 'Select'}
                  <ArrowDropDown />
                </button>
              </div>
            </div>

            <div className="flex justify-center -my-2 relative z-10">
              <IconButton
                onClick={handleSwapTokens}
                className="bg-orange-500 hover:bg-orange-600 text-white"
              >
                <SwapHoriz />
              </IconButton>
            </div>

            <div className={`rounded-xl p-4 mt-2 ${isDark ? 'bg-slate-800' : 'bg-slate-100'}`}>
              <div className="flex justify-between mb-2">
                <span className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>You Receive</span>
                <span className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Balance: --</span>
              </div>
              <div className="flex gap-2">
                <input
                  type="number"
                  value={amountOut}
                  readOnly
                  placeholder="0.00"
                  className={`flex-1 bg-transparent text-2xl font-semibold outline-none ${isDark ? 'text-white' : 'text-gray-900'}`}
                />
                <button
                  onClick={() => setShowTokenSelector('out')}
                  className={`flex items-center gap-2 px-3 py-2 rounded-lg ${isDark ? 'bg-slate-700 text-white' : 'bg-slate-200 text-gray-900'}`}
                >
                  {tokenOut?.symbol || 'Select'}
                  <ArrowDropDown />
                </button>
              </div>
            </div>

            {quote && (
              <div className={`mt-4 rounded-lg p-3 text-sm ${isDark ? 'bg-slate-800' : 'bg-slate-100'}`}>
                <div className="flex justify-between mb-1">
                  <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Rate</span>
                  <span>1 {tokenIn?.symbol} = {quote.exchangeRate.toFixed(6)} {tokenOut?.symbol}</span>
                </div>
                <div className="flex justify-between mb-1">
                  <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Price Impact</span>
                  <span className={quote.priceImpact > 1 ? 'text-red-500' : 'text-green-500'}>
                    {quote.priceImpact.toFixed(2)}%
                  </span>
                </div>
                <div className="flex justify-between mb-1">
                  <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Min. Received</span>
                  <span>{quote.minimumOut.toFixed(6)} {tokenOut?.symbol}</span>
                </div>
                <div className="flex justify-between">
                  <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Gas Fee</span>
                  <span>~${quote.gasFeeUsd.toFixed(2)}</span>
                </div>
              </div>
            )}

            <Button
              fullWidth
              variant="contained"
              size="large"
              onClick={handleSwap}
              disabled={loadingTx || !amountIn || !quote || !walletAddress}
              className="mt-4 bg-orange-500 hover:bg-orange-600"
              startIcon={loadingTx ? <CircularProgress size={20} /> : null}
            >
              {!walletAddress ? 'Connect Wallet' : !amountIn ? 'Enter Amount' : loadingTx ? 'Swapping...' : 'Swap'}
            </Button>

            {error && <Alert severity="error" className="mt-4" onClose={() => setError(null)}>{error}</Alert>}
            {success && <Alert severity="success" className="mt-4" onClose={() => setSuccess(null)}>{success}</Alert>}

            {txHash && (
              <Button
                href={`${CHAIN_CONFIG[chainId]?.explorer}/tx/${txHash}`}
                target="_blank"
                endIcon={<OpenInNew />}
                className="mt-2 text-green-500"
              >
                View on Explorer
              </Button>
            )}
          </CardContent>
        </Card>

        <div className={`flex justify-center gap-6 text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
          <div className="flex items-center gap-1"><Shield className="text-green-500" /> MEV Protected</div>
          <div className="flex items-center gap-1"><Speed className="text-green-500" /> Best Route</div>
          <div className="flex items-center gap-1"><CompareArrows className="text-green-500" /> 20+ DEXs</div>
        </div>
      </div>

      <Dialog open={!!showTokenSelector} onClose={() => setShowTokenSelector(null)} maxWidth="xs" fullWidth>
        <DialogTitle>Select Token</DialogTitle>
        <DialogContent>
          <List>
            {tokens.map((token) => (
              <ListItemButton
                key={token.address}
                onClick={() => {
                  if (showTokenSelector === 'in') {
                    if (token.symbol !== tokenOut?.symbol) setTokenIn(token);
                  } else {
                    if (token.symbol !== tokenIn?.symbol) setTokenOut(token);
                  }
                  setShowTokenSelector(null);
                }}
              >
                <Avatar src={token.logoUri} className="mr-2">{token.symbol[0]}</Avatar>
                <ListItemText
                  primary={token.symbol}
                  secondary={token.name}
                />
                {token.priceUsd && <Typography className={isDark ? 'text-gray-400' : 'text-gray-500'}>${token.priceUsd}</Typography>}
              </ListItemButton>
            ))}
          </List>
        </DialogContent>
      </Dialog>

      <Dialog open={showSettings} onClose={() => setShowSettings(false)} maxWidth="xs" fullWidth>
        <DialogTitle>Swap Settings</DialogTitle>
        <DialogContent>
          <Typography variant="subtitle2" className="mb-2">Slippage Tolerance</Typography>
          <div className="flex gap-2 mb-2">
            {[0.1, 0.5, 1.0].map((val) => (
              <Button
                key={val}
                variant={slippage === val ? 'contained' : 'outlined'}
                onClick={() => setSlippage(val)}
                size="small"
              >
                {val}%
              </Button>
            ))}
          </div>
          <input
            type="number"
            min={0.01}
            max={50}
            step={0.1}
            value={slippage}
            onChange={(e) => setSlippage(parseFloat(e.target.value))}
            className={`w-full px-3 py-2 rounded-lg border ${isDark ? 'bg-gray-700 text-white border-gray-600' : 'bg-white text-gray-900 border-gray-300'}`}
            placeholder="0.5"
          />
          {(() => {
            const s = Number(slippage);
            if (!Number.isFinite(s) || s <= 0) {
              return <Typography variant="caption" color="error" className="mt-1 block">Slippage must be greater than 0%.</Typography>;
            }
            if (s > 50) {
              return <Typography variant="caption" color="error" className="mt-1 block">Slippage above 50% is unsafe; your swap will likely revert or be front-run.</Typography>;
            }
            if (s > 5) {
              return <Typography variant="caption" color="warning.main" className="mt-1 block">High slippage — your swap may execute at an unfavorable price.</Typography>;
            }
            return null;
          })()}
          <Typography variant="caption" className={`${isDark ? 'text-gray-400' : 'text-gray-500'} mt-2 block`}>
            Your transaction will revert if the price changes unfavorably by more than this percentage.
          </Typography>
        </DialogContent>
      </Dialog>
    </div>
  );
}
