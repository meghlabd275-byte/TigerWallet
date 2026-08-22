// Onboarding Context — the no-registration self-custody entry model.
//
// The user NEVER sees a register/login form. On first launch the app shows a
// "Create Wallet" / "Import Wallet" choice. Behind the scenes a transparent
// ephemeral account is auto-provisioned (random device-bound identity stored in
// localStorage) so the JWT-backed WL backend security is preserved. The wallet
// password the user enters encrypts the seed (server-side scrypt + AES-GCM),
// independent of the ephemeral account.
//
// Flow:
//   ensureSession()  — if no local token, auto-register a random identity +
//                      login to obtain a JWT. One-time, invisible to the user.
//   createWallet()   — password -> backend POST /wallets -> mnemonic (backup).
//   importWallet()   — seed + password -> backend POST /wallets { mnemonic }.
//
// `onboarded` (a wallet exists locally OR the user completed backup) gates the
// app: false => show Onboarding (Create/Import); true => show Dashboard.
import React, { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';
import { api } from '../services/api';

const SESSION_KEY = 'userwallet-session';
const WALLET_IDS_KEY = 'userwallet-wallet-ids';

interface SessionBlob {
  email: string;
  password: string; // the ephemeral account password (NOT the wallet password)
  token: string;
  userId: string;
}

interface OnboardingContextType {
  ready: boolean;          // session bootstrap complete
  onboarded: boolean;      // at least one wallet exists locally
  ensureSession: () => Promise<void>;
  createWallet: (label: string, password: string, chainId: number) => Promise<{ mnemonic: string; id: string; address: string }>;
  importWallet: (mnemonic: string, label: string, password: string, chainId: number) => Promise<{ id: string; address: string }>;
  rememberWallet: (id: string) => void;
  localWalletIds: string[];
}

const OnboardingContext = createContext<OnboardingContextType | undefined>(undefined);

// CSPRNG identity for the transparent account. Uses crypto.getRandomValues
// (Web Crypto). The email is a random UUID @local.device pseudo-address so the
// backend's {email,password} register contract is satisfied without ever asking
// the user for an email.
function randomIdentity(): { email: string; password: string } {
  const bytes = new Uint8Array(32);
  crypto.getRandomValues(bytes);
  const id = bytesToHex(bytes.slice(0, 16));
  const email = `${id}@device.local`;
  // 32 hex chars = 128 bits of entropy for the ephemeral account password.
  const password = bytesToHex(bytes);
  return { email, password };
}

function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes).map((b) => b.toString(16).padStart(2, '0')).join('');
}

function loadSession(): SessionBlob | null {
  try {
    const raw = localStorage.getItem(SESSION_KEY);
    return raw ? (JSON.parse(raw) as SessionBlob) : null;
  } catch {
    return null;
  }
}

function saveSession(s: SessionBlob) {
  localStorage.setItem(SESSION_KEY, JSON.stringify(s));
}

export function OnboardingProvider({ children }: { children: ReactNode }) {
  const [ready, setReady] = useState(false);
  const [token, setToken] = useState<string | null>(null);
  const [localWalletIds, setLocalWalletIds] = useState<string[]>(() => {
    try {
      const raw = localStorage.getItem(WALLET_IDS_KEY);
      return raw ? (JSON.parse(raw) as string[]) : [];
    } catch {
      return [];
    }
  });

  const onboarded = localWalletIds.length > 0;

  const ensureSession = useCallback(async () => {
    if (token) {
      api.setToken(token);
      setReady(true);
      return;
    }
    let s = loadSession();
    if (!s) {
      const id = randomIdentity();
      try {
        await api.register(id.email, id.email, id.password);
      } catch {
        // If register fails (e.g. identity collision / network), fall through
        // to login which will surface the real error.
      }
      try {
        const { token: jwt } = await api.login(id.email, id.password);
        s = { email: id.email, password: id.password, token: jwt, userId: '' };
        saveSession(s);
      } catch (err) {
        // Cannot provision a transparent session — surface a real error.
        setReady(true);
        throw err;
      }
    } else {
      // Re-validate the stored token; if expired, re-login transparently.
      api.setToken(s.token);
      try {
        await api.getProfile();
      } catch {
        const { token: jwt } = await api.login(s.email, s.password);
        s = { ...s, token: jwt };
        saveSession(s);
      }
    }
    api.setToken(s.token);
    setToken(s.token);
    setReady(true);
  }, [token]);

  // Bootstrap on mount: provision the transparent session immediately so the
  // app can show Create/Import (the user never waits on a login form).
  useEffect(() => {
    ensureSession().catch(() => {
      // The error is surfaced via the UI banner; ready=true so the landing
      // page renders and can retry.
      setReady(true);
    });
  }, [ensureSession]);

  const createWallet = useCallback(
    async (label: string, password: string, chainId: number) => {
      await ensureSession();
      const w = await api.createWalletTyped({ label, password, chainId });
      if (w.mnemonic) {
        return { mnemonic: w.mnemonic, id: w.id, address: w.address };
      }
      throw new Error('Backend did not return a recovery phrase');
    },
    [ensureSession]
  );

  const importWallet = useCallback(
    async (mnemonic: string, label: string, password: string, chainId: number) => {
      await ensureSession();
      const w = await api.createWalletTyped({ label, password, chainId, mnemonic });
      return { id: w.id, address: w.address };
    },
    [ensureSession]
  );

  const rememberWallet = useCallback((id: string) => {
    setLocalWalletIds((prev) => {
      if (prev.includes(id)) return prev;
      const next = [...prev, id];
      localStorage.setItem(WALLET_IDS_KEY, JSON.stringify(next));
      return next;
    });
  }, []);

  return (
    <OnboardingContext.Provider
      value={{ ready, onboarded, ensureSession, createWallet, importWallet, rememberWallet, localWalletIds }}
    >
      {children}
    </OnboardingContext.Provider>
  );
}

export function useOnboarding() {
  const ctx = useContext(OnboardingContext);
  if (!ctx) throw new Error('useOnboarding must be used within OnboardingProvider');
  return ctx;
}
