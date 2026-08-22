// Onboarding Context — the no-registration self-custody entry model.
//
// The user NEVER sees a register/login form. On first launch the app shows a
// "Create Wallet" / "Import Wallet" choice. Behind the scenes a transparent
// ephemeral account is auto-provisioned (random device-bound identity stored in
// localStorage) so the JWT-backed wallet-api backend security is preserved. The
// wallet password the user enters encrypts the seed (server-side AES-GCM),
// independent of the ephemeral account.
//
// Flow:
//   ensureSession()  — if no local token, auto-register a random identity +
//                      login to obtain a JWT. One-time, invisible to the user.
//   createWallet()   — password -> backend POST /wallets -> mnemonic (backup).
//   importWallet()   — seed + password -> backend POST /wallets { mnemonic }.
//
// `onboarded` (a wallet exists locally OR the user completed backup) gates the
// app: false => show Onboarding (Create/Import); true => show the main routes.
//
// Mirrors web/src/contexts/OnboardingContext.tsx but delegates to the production
// WalletService (createWalletTyped) + AuthService (login/register) instead of
// the web `api` singleton.
import React, { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';
import { WalletService } from '../services/WalletService';
import { AuthService } from '../services/AuthService';

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

  // The production services are stateful singletons; create once per provider.
  const [walletService] = useState(() => new WalletService());
  const [authService] = useState(() => new AuthService());

  const onboarded = localWalletIds.length > 0;

  const ensureSession = useCallback(async () => {
    if (token) {
      setReady(true);
      return;
    }
    let s = loadSession();
    if (!s) {
      const id = randomIdentity();
      try {
        // AuthService.register persists the token to localStorage
        // ('tigerwallet-token') which the WalletService interceptor reads.
        await authService.register({
          email: id.email,
          username: id.email,
          password: id.password,
        });
      } catch {
        // If register fails (e.g. identity collision / network), fall through
        // to login which will surface the real error.
      }
      try {
        const { token: jwt } = await authService.login({
          email: id.email,
          password: id.password,
        });
        s = { email: id.email, password: id.password, token: jwt, userId: '' };
        saveSession(s);
      } catch (err) {
        // Cannot provision a transparent session — surface a real error.
        setReady(true);
        throw err;
      }
    } else {
      // Re-validate the stored token; if expired, re-login transparently.
      // AuthService stores/reads 'tigerwallet-token'; mirror it here.
      localStorage.setItem('tigerwallet-token', s.token);
      try {
        await authService.getCurrentUser();
      } catch {
        const { token: jwt } = await authService.login({
          email: s.email,
          password: s.password,
        });
        s = { ...s, token: jwt };
        saveSession(s);
      }
    }
    localStorage.setItem('tigerwallet-token', s.token);
    setToken(s.token);
    setReady(true);
  }, [token, authService]);

  // Bootstrap on mount: provision the transparent session immediately so the
  // app can show Create/Import (the user never waits on a login form).
  useEffect(() => {
    ensureSession().catch(() => {
      // The error is surfaced via the UI; ready=true so the landing
      // page renders and can retry.
      setReady(true);
    });
  }, [ensureSession]);

  const createWallet = useCallback(
    async (label: string, password: string, chainId: number) => {
      await ensureSession();
      const w = await walletService.createWalletTyped({ label, password, chainId });
      if (w.mnemonic) {
        return { mnemonic: w.mnemonic, id: w.id, address: w.address };
      }
      throw new Error('Backend did not return a recovery phrase');
    },
    [ensureSession, walletService]
  );

  const importWallet = useCallback(
    async (mnemonic: string, label: string, password: string, chainId: number) => {
      await ensureSession();
      const w = await walletService.createWalletTyped({ label, password, chainId, mnemonic });
      return { id: w.id, address: w.address };
    },
    [ensureSession, walletService]
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
