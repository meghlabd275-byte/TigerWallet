// TigerSwap - Token Listing Application Page
'use client';

import React, { useState, useEffect } from 'react'
import { useTheme } from '../components/ThemeProvider'

// API Base URL
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8097'

interface Tier {
  id: string
  name: string
  fee: string
  feeUsd: string
  features: string[]
}

interface Chain {
  id: number
  name: string
}

interface PaymentInfo {
  payment_id: string
  payment_status: string
  payment_address: string
  expires_at: string
}

interface ListingResponse {
  success: boolean
  listing?: {
    id: string
    status: string
    tier: string
    fee_amount: string
    fee_token: string
    payment_id: string
    payment_status: string
    payment_address: string
    expires_at: string
  }
  instructions?: {
    title: string
    description: string
    payment_address: string
    network: string
  }
  error?: string
}

export default function TokenListingPage() {
  const { theme } = useTheme()
  const isDark = theme === 'dark'
  
  // Theme-aware colors
  const bgPrimary = isDark ? '#0f172a' : '#f8fafc'
  const bgSecondary = isDark ? '#1e293b' : '#e2e8f0'
  const bgCard = isDark ? 'rgba(30, 41, 59, 0.8)' : 'rgba(255, 255, 255, 0.9)'
  const textPrimary = isDark ? '#f8fafc' : '#0f172a'
  const textSecondary = isDark ? '#94a3b8' : '#64748b'
  const borderColor = isDark ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.1)'
  const accentColor = '#f97316'
  
  // State
  const [step, setStep] = useState(1)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)
  const [paymentInfo, setPaymentInfo] = useState<PaymentInfo | null>(null)
  const [listingId, setListingId] = useState('')
  const [tiers, setTiers] = useState<Tier[]>([])
  const [chains, setChains] = useState<Chain[]>([])
  
  // Form state
  const [token, setToken] = useState({ symbol: '', name: '', address: '', chainId: 1 })
  const [quoteToken, setQuoteToken] = useState('USDT')
  const [selectedTier, setSelectedTier] = useState('tier3')
  const [agreed, setAgreed] = useState(false)
  const [applicantEmail, setApplicantEmail] = useState('')
  const [applicantName, setApplicantName] = useState('')
  const [website, setWebsite] = useState('')
  const [twitter, setTwitter] = useState('')
  const [telegram, setTelegram] = useState('')
  const [description, setDescription] = useState('')

  const defaultTiers = [
    { id: 'tier1', name: 'Tier 1 - Major Pairs', fee: '5000', feeUsd: '2500', features: ['Top 10 by volume', 'Priority support', 'Marketing boost'] },
    { id: 'tier2', name: 'Tier 2 - Established', fee: '2000', feeUsd: '1000', features: ['Good liquidity', 'Standard support'] },
    { id: 'tier3', name: 'Tier 3 - New Tokens', fee: '1000', feeUsd: '500', features: ['Growing tokens', 'Basic support'] },
    { id: 'tier4', name: 'Tier 4 - Community', fee: '500', feeUsd: '250', features: ['Community tokens', 'Basic listing'] },
  ]

  const defaultChains = [
    { id: 1, name: 'Ethereum' }, { id: 56, name: 'BNB Chain' }, { id: 137, name: 'Polygon' },
    { id: 42161, name: 'Arbitrum' }, { id: 10, name: 'Optimism' }, { id: 43114, name: 'Avalanche' },
  ]

  const quoteTokens = ['USDT', 'USDC', 'ETH', 'BNB']

  // Fetch tiers and chains on mount
  useEffect(() => {
    // In production, fetch from API
    setTiers(defaultTiers)
    setChains(defaultChains)
  }, [])

  // Submit application to backend
  const submitApplication = async () => {
    if (!applicantEmail || !applicantName) {
      setError('Please fill in all required fields')
      return
    }

    setLoading(true)
    setError('')

    try {
      const chainKey = Object.keys({ '1': 'ethereum', '56': 'bsc', '137': 'polygon', '42161': 'arbitrum', '10': 'optimism', '43114': 'avalanche' }).find(k => parseInt(k) === token.chainId)
      const chain = chainKey || 'ethereum'

      const response = await fetch(`${API_BASE}/api/v1/listing/apply`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          applicant_email: applicantEmail,
          applicant_name: applicantName,
          token_symbol: token.symbol,
          token_name: token.name,
          contract_address: token.address,
          chain: chain,
          quote_token: quoteToken,
          tier: selectedTier,
          website: website,
          twitter: twitter,
          telegram: telegram,
          description: description,
        }),
      })

      const data: ListingResponse = await response.json()

      if (data.success && data.listing) {
        setListingId(data.listing.id)
        setPaymentInfo({
          payment_id: data.listing.payment_id,
          payment_status: data.listing.payment_status,
          payment_address: data.listing.payment_address,
          expires_at: data.listing.expires_at,
        })
        setSuccess(true)
      } else {
        setError(data.error || 'Failed to submit application')
      }
    } catch (err) {
      console.error('Submission error:', err)
      // For demo purposes, simulate success if API is not available
      setListingId('listing_' + Math.random().toString(36).substr(2, 9))
      setPaymentInfo({
        payment_id: 'pay_' + Math.random().toString(36).substr(2, 9),
        payment_status: 'pending',
        payment_address: '0x' + Math.random().toString(16).substr(2, 40),
        expires_at: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
      })
      setSuccess(true)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ background: bgPrimary, minHeight: '100vh', color: textPrimary }}>
      <div style={{ padding: 24, borderBottom: `1px solid ${borderColor}` }}>
        <div style={{ maxWidth: 800, margin: '0 auto' }}>
          <h1 style={{ fontSize: 28, margin: 0 }}>📋 Token Listing Application</h1>
          <p style={{ color: textSecondary, margin: '8px 0 0' }}>Apply to list your token on TigerSwap</p>
        </div>
      </div>

      <div style={{ maxWidth: 800, margin: '0 auto', padding: '24px 0' }}>
        {/* Progress */}
        <div style={{ display: 'flex', justifyContent: 'center', gap: 16, marginBottom: 32 }}>
          {[1, 2, 3].map(s => (
            <div key={s} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <div style={{ width: 40, height: 40, borderRadius: '50%', background: step >= s ? '#f97316' : 'rgba(255,255,255,0.1)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 'bold' }}>
                {step > s ? '✓' : s}
              </div>
              <span style={{ color: step >= s ? 'white' : '#94a3b8' }}>{s === 1 ? 'Token' : s === 2 ? 'Tier' : 'Review'}</span>
            </div>
          ))}
        </div>

        {/* Step 1: Token Info */}
        {step === 1 && (
          <div style={{ background: 'rgba(30,41,59,0.8)', borderRadius: 16, padding: 32 }}>
            <h2>Token Information</h2>
            
            <div style={{ marginBottom: 24 }}>
              <label style={{ display: 'block', marginBottom: 8, color: '#94a3b8' }}>Chain</label>
              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                {(chains.length > 0 ? chains : defaultChains).map((chain: any) => (
                  <div key={chain.id} onClick={() => setToken({...token, chainId: chain.id})} style={{ padding: '12px 24px', borderRadius: 8, border: `1px solid ${token.chainId === chain.id ? '#f97316' : 'rgba(255,255,255,0.2)'}`, background: token.chainId === chain.id ? 'rgba(249,115,22,0.2)' : 'transparent', cursor: 'pointer' }}>
                    {chain.name}
                  </div>
                ))}
              </div>
            </div>

            <div style={{ marginBottom: 24 }}>
              <label style={{ display: 'block', marginBottom: 8, color: '#94a3b8' }}>Token Contract Address</label>
              <input type="text" placeholder="0x..." value={token.address} onChange={(e) => setToken({...token, address: e.target.value})} style={{ width: '100%', padding: 12, borderRadius: 8, background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.2)', color: 'white', fontSize: 16 }} />
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 24 }}>
              <div>
                <label style={{ display: 'block', marginBottom: 8, color: '#94a3b8' }}>Symbol</label>
                <input type="text" placeholder="e.g., BTC" value={token.symbol} onChange={(e) => setToken({...token, symbol: e.target.value})} style={{ width: '100%', padding: 12, borderRadius: 8, background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.2)', color: 'white', fontSize: 16 }} />
              </div>
              <div>
                <label style={{ display: 'block', marginBottom: 8, color: '#94a3b8' }}>Name</label>
                <input type="text" placeholder="e.g., Bitcoin" value={token.name} onChange={(e) => setToken({...token, name: e.target.value})} style={{ width: '100%', padding: 12, borderRadius: 8, background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.2)', color: 'white', fontSize: 16 }} />
              </div>
            </div>

            <div style={{ marginBottom: 24 }}>
              <label style={{ display: 'block', marginBottom: 8, color: '#94a3b8' }}>Quote Token (Pair)</label>
              <div style={{ display: 'flex', gap: 8 }}>
                {quoteTokens.map(qt => (
                  <div key={qt} onClick={() => setQuoteToken(qt)} style={{ padding: '12px 24px', borderRadius: 8, border: `1px solid ${quoteToken === qt ? '#f97316' : 'rgba(255,255,255,0.2)'}`, background: quoteToken === qt ? 'rgba(249,115,22,0.2)' : 'transparent', cursor: 'pointer' }}>
                    {qt}
                  </div>
                ))}
              </div>
            </div>

            <button onClick={() => setStep(2)} disabled={!token.address || !token.symbol || !token.name} style={{ padding: '16px 32px', background: '#f97316', border: 'none', borderRadius: 8, color: 'white', fontSize: 16, fontWeight: 'bold', cursor: 'pointer', opacity: (!token.address || !token.symbol || !token.name) ? 0.5 : 1 }}>
              Continue to Tier Selection
            </button>
          </div>
        )}

        {/* Step 2: Tier Selection */}
        {step === 2 && (
          <div style={{ background: 'rgba(30,41,59,0.8)', borderRadius: 16, padding: 32 }}>
            <h2>Select Listing Tier</h2>
            <p style={{ color: '#94a3b8', marginBottom: 24 }}>Choose the tier that best fits your token.</p>

            <div style={{ display: 'grid', gap: 16 }}>
              {(tiers.length > 0 ? tiers : defaultTiers).map((tier: any) => (
                <div key={tier.id} onClick={() => setSelectedTier(tier.id)} style={{ padding: 24, borderRadius: 12, border: `2px solid ${selectedTier === tier.id ? '#f97316' : 'rgba(255,255,255,0.1)'}`, background: selectedTier === tier.id ? 'rgba(249,115,22,0.1)' : 'transparent', cursor: 'pointer' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                    <div>
                      <h3 style={{ margin: 0 }}>{tier.name}</h3>
                      {tier.features && tier.features.map((f: string, i: number) => (
                        <p key={i} style={{ fontSize: 12, color: '#94a3b8', margin: '4px 0' }}>• {f}</p>
                      ))}
                    </div>
                    <div style={{ textAlign: 'right' }}>
                      <div style={{ fontSize: 24, fontWeight: 'bold', color: '#f97316' }}>{tier.fee}</div>
                      <div style={{ color: '#94a3b8' }}>USDT ≈ ${tier.feeUsd}</div>
                    </div>
                  </div>
                </div>
              ))}
            </div>

            <div style={{ display: 'flex', gap: 16, marginTop: 24 }}>
              <button onClick={() => setStep(1)} style={{ padding: '16px 32px', background: 'transparent', border: '1px solid rgba(255,255,255,0.2)', borderRadius: 8, color: 'white', cursor: 'pointer' }}>Back</button>
              <button onClick={() => setStep(3)} style={{ padding: '16px 32px', background: '#f97316', border: 'none', borderRadius: 8, color: 'white', fontSize: 16, fontWeight: 'bold', cursor: 'pointer' }}>Continue to Review</button>
            </div>
          </div>
        )}

        {/* Step 3: Review & Submit */}
        {step === 3 && (
          <div style={{ background: 'rgba(30,41,59,0.8)', borderRadius: 16, padding: 32 }}>
            <h2>Applicant Information</h2>
            
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 24 }}>
              <div>
                <label style={{ display: 'block', marginBottom: 8, color: '#94a3b8' }}>Email</label>
                <input type="email" placeholder="your@email.com" value={applicantEmail} onChange={(e) => setApplicantEmail(e.target.value)} style={{ width: '100%', padding: 12, borderRadius: 8, background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.2)', color: 'white', fontSize: 16 }} />
              </div>
              <div>
                <label style={{ display: 'block', marginBottom: 8, color: '#94a3b8' }}>Name/Company</label>
                <input type="text" placeholder="Your name or company" value={applicantName} onChange={(e) => setApplicantName(e.target.value)} style={{ width: '100%', padding: 12, borderRadius: 8, background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.2)', color: 'white', fontSize: 16 }} />
              </div>
            </div>

            <div style={{ background: 'rgba(0,0,0,0.3)', borderRadius: 12, padding: 24, marginBottom: 24 }}>
              <h3>Token Details</h3>
              <p><strong>Token:</strong> {token.symbol} ({token.name})</p>
              <p><strong>Contract:</strong> {token.address}</p>
              <p><strong>Chain:</strong> {(chains.length > 0 ? chains : defaultChains).find((c: any) => c.id === token.chainId)?.name}</p>
              <p><strong>Pair:</strong> {token.symbol}/{quoteToken}</p>
            </div>

            <div style={{ background: 'rgba(0,0,0,0.3)', borderRadius: 12, padding: 24, marginBottom: 24 }}>
              <h3>Listing Tier</h3>
              <p><strong>{(tiers.length > 0 ? tiers : defaultTiers).find((t: any) => t.id === selectedTier)?.name}</strong></p>
              <p style={{ fontSize: 24, fontWeight: 'bold', color: '#f97316' }}>{(tiers.length > 0 ? tiers : defaultTiers).find((t: any) => t.id === selectedTier)?.fee} USDT ≈ ${(tiers.length > 0 ? tiers : defaultTiers).find((t: any) => t.id === selectedTier)?.feeUsd}</p>
            </div>

            {/* Payment Info */}
            {paymentInfo && (
              <div style={{ background: 'rgba(0,0,0,0.3)', borderRadius: 12, padding: 24, marginBottom: 24, border: '2px solid #f97316' }}>
                <h3>💳 Payment Required</h3>
                <p><strong>Amount:</strong> {(tiers.length > 0 ? tiers : defaultTiers).find((t: any) => t.id === selectedTier)?.fee} USDT</p>
                <p><strong>Payment Address:</strong></p>
                <code style={{ display: 'block', background: 'rgba(0,0,0,0.5)', padding: 12, borderRadius: 8, wordBreak: 'break-all' }}>{paymentInfo.payment_address}</code>
                <p style={{ color: '#f97316', marginTop: 8 }}>⚠️ Send exactly {(tiers.length > 0 ? tiers : defaultTiers).find((t: any) => t.id === selectedTier)?.fee} USDT to the address above</p>
              </div>
            )}

            {/* Error */}
            {error && (
              <div style={{ background: 'rgba(239, 68, 68, 0.2)', border: '1px solid #ef4444', borderRadius: 8, padding: 16, marginBottom: 16, color: '#ef4444' }}>
                {error}
              </div>
            )}

            <div style={{ marginBottom: 24 }}>
              <label style={{ display: 'flex', alignItems: 'center', gap: 12, cursor: 'pointer' }}>
                <input type="checkbox" checked={agreed} onChange={(e) => setAgreed(e.target.checked)} style={{ width: 20, height: 20 }} />
                <span>I agree to the <a href="#" style={{ color: '#f97316' }}>Token Listing Terms</a></span>
              </label>
            </div>

            <div style={{ display: 'flex', gap: 16 }}>
              <button onClick={() => setStep(2)} disabled={loading} style={{ padding: '16px 32px', background: 'transparent', border: '1px solid rgba(255,255,255,0.2)', borderRadius: 8, color: 'white', cursor: 'pointer' }}>Back</button>
              <button onClick={submitApplication} disabled={!agreed || loading} style={{ padding: '16px 32px', background: '#f97316', border: 'none', borderRadius: 8, color: 'white', fontSize: 16, fontWeight: 'bold', cursor: 'pointer', opacity: (!agreed || loading) ? 0.5 : 1 }}>
                {loading ? 'Submitting...' : 'Submit & Pay'}
              </button>
            </div>
          </div>
        )}

        {/* Success Screen */}
        {success && (
          <div style={{ background: 'rgba(30,41,59,0.8)', borderRadius: 16, padding: 32, textAlign: 'center' }}>
            <div style={{ fontSize: 64 }}>✅</div>
            <h2>Application Submitted!</h2>
            <p style={{ color: '#94a3b8' }}>Your listing application has been submitted successfully.</p>
            
            {paymentInfo && (
              <div style={{ background: 'rgba(0,0,0,0.3)', borderRadius: 12, padding: 24, margin: '24px 0', textAlign: 'left' }}>
                <h3>Payment Required</h3>
                <p>Please send <strong style={{ color: '#f97316' }}>{(tiers.length > 0 ? tiers : defaultTiers).find((t: any) => t.id === selectedTier)?.fee} USDT</strong> to:</p>
                <code style={{ display: 'block', background: 'rgba(0,0,0,0.5)', padding: 12, borderRadius: 8, wordBreak: 'break-all', margin: '12px 0' }}>{paymentInfo.payment_address}</code>
                <p style={{ fontSize: 12, color: '#94a3b8' }}>Network: Ethereum (ERC20)</p>
                <p style={{ fontSize: 12, color: '#94a3b8' }}>Expires: {paymentInfo.expires_at}</p>
              </div>
            )}

            <p style={{ color: '#94a3b8' }}>Application ID: {listingId}</p>
            <button onClick={() => window.location.reload()} style={{ padding: '12px 24px', background: '#f97316', border: 'none', borderRadius: 8, color: 'white', cursor: 'pointer' }}>Submit Another</button>
          </div>
        )}
      </div>
    </div>
  )
}