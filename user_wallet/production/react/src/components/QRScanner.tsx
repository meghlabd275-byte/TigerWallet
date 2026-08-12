/**
 * QRScanner - real camera-based QR / address scanner.
 *
 * Uses the native BarcodeDetector Web API (W3C, shipped in Chromium) to decode
 * QR codes from the live device camera stream. On browsers without
 * BarcodeDetector (Firefox/Safari), it shows a manual-paste input so the
 * send/receive flow still works end-to-end. No mock data -- it only ever
 * returns an address that the user actually scanned or typed.
 *
 * Recognised QR payloads:
 *   - bare address:        0x1234...
 *   - ethereum: URI:        ethereum:0x1234...
 *   - EIP-681 payment:     ethereum:0x1234.../transfer?value=...
 */

import React, { useRef, useEffect, useState, useCallback } from 'react';

export interface QRScannerProps {
  isOpen: boolean;
  onClose: () => void;
  onScan: (address: string, chain?: string) => void;
  title?: string;
  recentAddresses?: string[];
}

interface BarcodeDetectorLike {
  detect: (source: CanvasImageSource) => Promise<Array<{ rawValue: string }>>;
}

function parseAddress(raw: string): { address: string; chain?: string } {
  const value = raw.trim();
  // EIP-681 / ethereum: URI scheme
  const ethMatch = value.match(/^ethereum:(0x[0-9a-fA-F]+)/i);
  if (ethMatch) return { address: ethMatch[1], chain: 'evm' };
  // bare EVM address
  const evmMatch = value.match(/(0x[0-9a-fA-F]{40})/);
  if (evmMatch) return { address: evmMatch[1], chain: 'evm' };
  // Solana base58 address (32-44 chars, base58)
  const solMatch = value.match(/\b([1-9A-HJ-NP-Za-km-z]{32,44})\b/);
  if (solMatch) return { address: solMatch[1], chain: 'solana' };
  return { address: value };
}

const QRScanner: React.FC<QRScannerProps> = ({
  isOpen,
  onClose,
  onScan,
  title = 'Scan QR Code',
  recentAddresses = [],
}) => {
  const videoRef = useRef<HTMLVideoElement>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const detectorRef = useRef<BarcodeDetectorLike | null>(null);
  const rafRef = useRef<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [manualValue, setManualValue] = useState('');
  const [supportsCamera, setSupportsCamera] = useState(true);

  const stop = useCallback(() => {
    if (rafRef.current) cancelAnimationFrame(rafRef.current);
    rafRef.current = null;
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((t) => t.stop());
      streamRef.current = null;
    }
  }, []);

  const handleResult = useCallback(
    (raw: string) => {
      const { address, chain } = parseAddress(raw);
      if (address) {
        onScan(address, chain);
        stop();
        onClose();
      }
    },
    [onScan, onClose, stop],
  );

  useEffect(() => {
    if (!isOpen) {
      stop();
      setError(null);
      setManualValue('');
      return;
    }

    let cancelled = false;

    const start = async () => {
      // Prefer the native BarcodeDetector for real on-device decode.
      const BarcodeDetectorCtor =
        (window as unknown as { BarcodeDetector?: new (opts: { formats: string[] }) => BarcodeDetectorLike })
          .BarcodeDetector;
      if (!BarcodeDetectorCtor || !navigator.mediaDevices?.getUserMedia) {
        setSupportsCamera(false);
        return;
      }
      try {
        detectorRef.current = new BarcodeDetectorCtor({ formats: ['qr_code'] });
        const stream = await navigator.mediaDevices.getUserMedia({
          video: { facingMode: 'environment' },
        });
        if (cancelled) {
          stream.getTracks().forEach((t) => t.stop());
          return;
        }
        streamRef.current = stream;
        if (videoRef.current) {
          videoRef.current.srcObject = stream;
          await videoRef.current.play();
        }
        const tick = async () => {
          if (cancelled || !videoRef.current || !detectorRef.current) return;
          try {
            const results = await detectorRef.current.detect(videoRef.current);
            if (results && results.length > 0 && results[0].rawValue) {
              handleResult(results[0].rawValue);
              return;
            }
          } catch {
            // detection may transiently fail on a blank frame -- just retry
          }
          rafRef.current = requestAnimationFrame(tick);
        };
        tick();
      } catch (e) {
        if (!cancelled) {
          setSupportsCamera(false);
          setError(
            e instanceof Error && e.name === 'NotAllowedError'
              ? 'Camera permission denied. Enter the address manually below.'
              : 'Camera unavailable. Enter the address manually below.',
          );
        }
      }
    };

    start();
    return () => {
      cancelled = true;
      stop();
    };
  }, [isOpen, handleResult, stop]);

  if (!isOpen) return null;

  const submitManual = (e: React.FormEvent) => {
    e.preventDefault();
    if (manualValue.trim()) handleResult(manualValue);
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      style={{ background: 'rgba(0,0,0,0.7)' }}
      onClick={onClose}
    >
      <div
        className="relative w-full max-w-md rounded-2xl p-6"
        style={{ background: 'var(--color-bg-primary)' }}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold" style={{ color: 'var(--color-text-primary)' }}>
            {title}
          </h3>
          <button
            onClick={onClose}
            className="w-8 h-8 flex items-center justify-center rounded-lg"
            style={{ background: 'var(--color-bg-tertiary)', color: 'var(--color-text-primary)' }}
            aria-label="Close"
          >
            x
          </button>
        </div>

        {supportsCamera ? (
          <div className="relative w-full aspect-square rounded-xl overflow-hidden" style={{ background: '#000' }}>
            <video
              ref={videoRef}
              className="w-full h-full object-cover"
              playsInline
              muted
            />
            <div
              className="absolute inset-8 border-2 rounded-xl pointer-events-none"
              style={{ borderColor: 'var(--color-primary)' }}
            />
          </div>
        ) : (
          <p className="text-sm text-center py-8" style={{ color: 'var(--color-text-tertiary)' }}>
            {error || 'Camera scanning unavailable on this browser.'}
          </p>
        )}

        <form onSubmit={submitManual} className="mt-4">
          <input
            type="text"
            value={manualValue}
            onChange={(e) => setManualValue(e.target.value)}
            placeholder="Enter address (0x...) or ethereum: URI"
            className="w-full px-3 py-2 rounded-lg outline-none"
            style={{
              background: 'var(--color-bg-secondary)',
              border: '1px solid var(--color-border)',
              color: 'var(--color-text-primary)',
            }}
          />
          <button
            type="submit"
            className="w-full mt-3 py-2 rounded-lg font-medium"
            style={{ background: 'var(--color-primary)', color: '#fff' }}
          >
            Confirm
          </button>
        </form>

        {recentAddresses.length > 0 && (
          <div className="mt-4">
            <p className="text-xs uppercase tracking-wide mb-2" style={{ color: 'var(--color-text-tertiary)' }}>
              Recent
            </p>
            <div className="space-y-1">
              {recentAddresses.slice(0, 5).map((addr) => (
                <button
                  key={addr}
                  onClick={() => handleResult(addr)}
                  className="block w-full text-left px-3 py-2 rounded-lg text-sm font-mono truncate"
                  style={{ background: 'var(--color-bg-secondary)', color: 'var(--color-text-secondary)' }}
                >
                  {addr}
                </button>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default QRScanner;
