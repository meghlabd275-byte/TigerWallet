// Auth Context - Authentication State
//
// UserWallet requires NO registration. On first open the app shows a
// CreateWallet / ImportWallet choice. Selecting either provisions an anonymous
// guest account via POST /auth/guest (no email/password) so a wallet can be
// created or imported immediately. The guest token is persisted exactly like a
// login token, so subsequent app opens unlock straight into the wallet.
//
// A returning user with a stored token unlocks the app directly (no re-entry
// of anything). Passcode / fingerprint unlock is handled at the UI layer on
// top of the stored token (see UnlockPage). Email/password login + register
// remain available as an OPTIONAL account-recovery path.
import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { api } from '../services/api';

interface User {
  id: string;
  email: string;
  username: string;
  guest?: boolean;
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, username: string, password: string) => Promise<void>;
  guestAuth: (deviceId: string) => Promise<void>;
  logout: () => void;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(() => localStorage.getItem('userwallet-token'));

  useEffect(() => {
    if (token) {
      api.setToken(token);
      api.getProfile().then(setUser).catch(() => logout());
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const login = async (email: string, password: string) => {
    const { token: newToken, user: newUser } = await api.login(email, password);
    localStorage.setItem('userwallet-token', newToken);
    setToken(newToken);
    setUser(newUser);
    api.setToken(newToken);
  };

  const register = async (email: string, username: string, password: string) => {
    await api.register(email, username, password);
    const { token: newToken, user: newUser } = await api.login(email, password);
    localStorage.setItem('userwallet-token', newToken);
    setToken(newToken);
    setUser(newUser);
    api.setToken(newToken);
  };

  // Provision an anonymous guest account so the user can Create/Import a
  // wallet WITHOUT registering. The token is persisted like a login token.
  const guestAuth = async (deviceId: string) => {
    const { token: newToken } = await api.guestAuth(deviceId);
    localStorage.setItem('userwallet-token', newToken);
    setToken(newToken);
    api.setToken(newToken);
    // The guest profile is decoded from the JWT by getProfile on the next tick.
    try {
      const profile = await api.getProfile();
      setUser({ ...profile, guest: true });
    } catch {
      setUser({ id: '', email: '', username: 'Guest', guest: true });
    }
  };

  const logout = () => {
    localStorage.removeItem('userwallet-token');
    setToken(null);
    setUser(null);
    api.setToken('');
  };

  return (
    <AuthContext.Provider value={{ user, token, login, register, guestAuth, logout, isAuthenticated: !!token }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be used within AuthProvider');
  return context;
}
