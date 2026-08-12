'use client'

// TigerWallet - Supported Chains Page
// Displays ALL preinstalled mainnet chains (120 EVM + 66 non-EVM) fetched live
// from the canonical go/wallet_api registry (GET /api/v1/chains). No hardcoded
// data. Light/dark theme via useTheme().

import React, { useState, useEffect } from 'react'
import { useTheme } from '../components/ThemeProvider'
import { walletService, ChainInfo } from '../api/service'

type FilterType = 'all' | 'evm' | 'nonevm'

export default function ChainsPage() {
  const { isDark } = useTheme()
  const [chains, setChains] = useState<ChainInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [filter, setFilter] = useState<FilterType>('all')
  const [selectedChain, setSelectedChain] = useState<ChainInfo | null>(null)

  useEffect(() => {
    let mounted = true
    ;(async () => {
      try {
        setLoading(true)
        const { chains: data } = await walletService.getSupportedChains()
        if (mounted) {
          setChains(data || [])
          setError('')
        }
      } catch (e: unknown) {
        if (mounted) setError(e instanceof Error ? e.message : 'Failed to load chains')
      } finally {
        if (mounted) setLoading(false)
      }
    })()
    return () => { mounted = false }
  }, [])

  const isEVM = (c: ChainInfo) => c.chain_type === 'evm'
  const evmCount = chains.filter(isEVM).length
  const nonEvmCount = chains.length - evmCount

  const evmList = chains.filter(c => filter !== 'nonevm' && isEVM(c))
  const nonEvmList = chains.filter(c => filter !== 'evm' && !isEVM(c))

  const chainColors: Record<string, string> = {
    ethereum: '#627EEA', bsc: '#F3BA2F', polygon: '#8247E5', arbitrum: '#28A0F0',
    optimism: '#FF0420', avalanche: '#E84142', base: '#0052FF', fantom: '#13B5EC',
    solana: '#9945FF', tron: '#EF0027', sui: '#6FBCEF', aptos: '#3D2847',
    bitcoin: '#F7931A', cosmos: '#2E3148', polkadot: '#E6007A', near: '#00EC97',
    cardano: '#0033AD', pi: '#FFC107',
  }
  const colorFor = (c: ChainInfo) =>
    chainColors[c.name.toLowerCase().split(' ')[0]] || (isEVM(c) ? '#f97316' : '#8b5cf6')

  const bgPrimary = isDark ? '#0f172a' : '#f8fafc'
  const bgSecondary = isDark ? 'rgba(30,41,59,0.8)' : '#ffffff'
  const bgModal = isDark ? '#1e293b' : '#ffffff'
  const textColor = isDark ? '#ffffff' : '#0f172a'
  const textSecondary = isDark ? '#94a3b8' : '#64748b'
  const borderColor = isDark ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.1)'
  const btnIdleBg = isDark ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.05)'

  const renderCard = (chain: ChainInfo) => {
    const col = colorFor(chain)
    return (
      <div key={chain.id} onClick={() => setSelectedChain(chain)} style={{
        background: bgSecondary, borderRadius: 12, padding: 20, cursor: 'pointer',
        border: `1px solid ${col}`,
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 12 }}>
          <div style={{ width: 48, height: 48, borderRadius: '50%', background: col, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 20, fontWeight: 'bold', color: '#fff' }}>{chain.name[0]}</div>
          <div>
            <h3 style={{ margin: 0 }}>{chain.name}</h3>
            <p style={{ margin: 0, color: textSecondary, fontSize: 14 }}>Chain ID: {chain.id}</p>
          </div>
        </div>
        <p style={{ color: textSecondary, fontSize: 14, marginBottom: 12, wordBreak: 'break-all' }}>
          {chain.explorer_url || 'Explorer not configured'}
        </p>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <span style={{ background: isEVM(chain) ? 'rgba(16,185,129,0.2)' : 'rgba(139,92,246,0.2)', padding: '4px 12px', borderRadius: 8, fontSize: 12, color: isEVM(chain) ? '#10b981' : '#8b5cf6' }}>{chain.symbol}</span>
          <span style={{ padding: '4px 12px', borderRadius: 8, fontSize: 12, background: chain.rpc_endpoint ? 'rgba(16,185,129,0.2)' : 'rgba(239,68,68,0.2)', color: chain.rpc_endpoint ? '#10b981' : '#ef4444' }}>{chain.rpc_endpoint ? '✓ RPC' : 'No public RPC'}</span>
        </div>
      </div>
    )
  }

  return (
    <div style={{ background: bgPrimary, minHeight: '100vh', color: textColor }}>
      <div style={{ padding: 24, borderBottom: `1px solid ${borderColor}` }}>
        <div style={{ maxWidth: 1200, margin: '0 auto' }}>
          <h1 style={{ fontSize: 28, margin: 0, display: 'flex', alignItems: 'center', gap: 12 }}>
            <span>🔗</span> Supported Chains
          </h1>
          <p style={{ color: textSecondary, margin: '8px 0 0' }}>{chains.length} mainnet chains — preinstalled, admin-extensible</p>
        </div>
      </div>

      <div style={{ maxWidth: 1200, margin: '0 auto', padding: 24 }}>
        {error && (
          <div style={{ background: isDark ? 'rgba(239,68,68,0.15)' : 'rgba(239,68,68,0.1)', color: '#ef4444', padding: 12, borderRadius: 8, marginBottom: 16 }}>
            {error}
          </div>
        )}
        {loading ? (
          <p style={{ color: textSecondary }}>Loading chains…</p>
        ) : (
          <>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 16, marginBottom: 32 }}>
              <div style={{ background: bgSecondary, padding: 20, borderRadius: 12 }}>
                <p style={{ color: textSecondary, margin: 0 }}>Total Chains</p>
                <h2 style={{ margin: '8px 0', fontSize: 32 }}>{chains.length}</h2>
              </div>
              <div style={{ background: bgSecondary, padding: 20, borderRadius: 12 }}>
                <p style={{ color: textSecondary, margin: 0 }}>EVM / Non-EVM</p>
                <h2 style={{ margin: '8px 0', fontSize: 32 }}>{evmCount} / {nonEvmCount}</h2>
              </div>
              <div style={{ background: bgSecondary, padding: 20, borderRadius: 12 }}>
                <p style={{ color: textSecondary, margin: 0 }}>RPC Configured</p>
                <h2 style={{ margin: '8px 0', fontSize: 32, color: '#10b981' }}>{chains.filter(c => c.rpc_endpoint).length}</h2>
              </div>
            </div>

            <div style={{ display: 'flex', gap: 8, marginBottom: 24 }}>
              {(['all', 'evm', 'nonevm'] as FilterType[]).map(f => (
                <button key={f} onClick={() => setFilter(f)} style={{
                  padding: '8px 16px', borderRadius: 8, border: 'none', cursor: 'pointer',
                  background: filter === f ? '#f97316' : btnIdleBg, color: filter === f ? '#fff' : textColor,
                  fontWeight: filter === f ? 'bold' : 'normal',
                }}>
                  {f === 'all' ? 'All' : f === 'evm' ? 'EVM' : 'Non-EVM'}
                </button>
              ))}
            </div>

            {filter !== 'nonevm' && (
              <div style={{ marginBottom: 40 }}>
                <h2 style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16 }}>
                  <span>⚛️</span> EVM Chains <span style={{ background: '#f97316', padding: '2px 12px', borderRadius: 12, fontSize: 12 }}>{evmList.length}</span>
                </h2>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 16 }}>
                  {evmList.map(renderCard)}
                </div>
              </div>
            )}

            {filter !== 'evm' && (
              <div>
                <h2 style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16 }}>
                  <span>🌐</span> Non-EVM Chains <span style={{ background: '#8b5cf6', padding: '2px 12px', borderRadius: 12, fontSize: 12 }}>{nonEvmList.length}</span>
                </h2>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 16 }}>
                  {nonEvmList.map(renderCard)}
                </div>
              </div>
            )}
          </>
        )}
      </div>

      {selectedChain && (
        <div style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, background: 'rgba(0,0,0,0.8)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000 }} onClick={() => setSelectedChain(null)}>
          <div style={{ background: bgModal, borderRadius: 16, padding: 32, maxWidth: 500, width: '90%', border: `2px solid ${colorFor(selectedChain)}` }} onClick={e => e.stopPropagation()}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 24 }}>
              <div style={{ width: 64, height: 64, borderRadius: '50%', background: colorFor(selectedChain), display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 28, fontWeight: 'bold', color: '#fff' }}>{selectedChain.name[0]}</div>
              <div>
                <h2 style={{ margin: 0 }}>{selectedChain.name}</h2>
                <p style={{ margin: 0, color: textSecondary }}>{selectedChain.chain_type.toUpperCase()} • {selectedChain.symbol}</p>
              </div>
            </div>
            <p style={{ color: textSecondary, margin: '0 0 4px' }}>Chain ID</p>
            <p style={{ margin: '0 0 16px', fontSize: 18 }}>{selectedChain.id}</p>
            <p style={{ color: textSecondary, margin: '0 0 4px' }}>RPC Endpoint</p>
            <p style={{ margin: '0 0 16px', fontSize: 14, wordBreak: 'break-all' }}>{selectedChain.rpc_endpoint || 'Not configured (admin can set)'}</p>
            <p style={{ color: textSecondary, margin: '0 0 4px' }}>Explorer</p>
            <p style={{ margin: '0 0 16px', fontSize: 14, wordBreak: 'break-all' }}>{selectedChain.explorer_url || '—'}</p>
            <p style={{ color: textSecondary, margin: '0 0 4px' }}>Derivation Path</p>
            <p style={{ margin: '0 0 16px', fontSize: 14 }}>{selectedChain.derivation_path}</p>
            <p style={{ color: textSecondary, margin: '0 0 4px' }}>Decimals / Coin Type</p>
            <p style={{ margin: '0 0 24px', fontSize: 14 }}>{selectedChain.decimals} / {selectedChain.coin_type}</p>
            <div style={{ display: 'flex', gap: 12 }}>
              <button style={{ flex: 1, padding: 12, borderRadius: 8, border: 'none', background: isDark ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.05)', color: textColor, cursor: 'pointer' }} onClick={() => setSelectedChain(null)}>Close</button>
              <button style={{ flex: 1, padding: 12, borderRadius: 8, border: 'none', background: '#f97316', color: 'white', cursor: 'pointer', fontWeight: 'bold' }}>Connect</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
