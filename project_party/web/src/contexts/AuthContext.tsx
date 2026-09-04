// Auth Context — WL-ProjectParty. Real register/login through the API service
// (proxied to the WL standalone backend). No hardcoded hosts.
import React, { createContext, useContext, useState, ReactNode } from 'react';
import { api } from '../services/api';

interface AuthContextType {
  token: string | null;
  email: string | null;
  isAdmin: boolean;
  register: (email: string, password: string) => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
}

const ADMIN_ROLES = ['admin', 'super_admin'];
const ADMIN_SCOPES = ['wl_client', 'listing_admin'];

function computeIsAdmin(role: string | null, scopes: string[] | null): boolean {
  if (role && ADMIN_ROLES.includes(role)) return true;
  if (scopes && scopes.some(s => ADMIN_SCOPES.includes(s))) return true;
  return false;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(() => {
    if (typeof window === 'undefined') return null;
    const t = localStorage.getItem('projectparty-token');
    if (t) api.setToken(t);
    return t;
  });
  const [email, setEmail] = useState<string | null>(() =>
    typeof window === 'undefined' ? null : localStorage.getItem('projectparty-email')
  );
  const [isAdmin, setIsAdmin] = useState<boolean>(() => {
    if (typeof window === 'undefined') return false;
    try {
      const raw = localStorage.getItem('projectparty-admin');
      if (!raw) return false;
      const { role, scopes } = JSON.parse(raw);
      return computeIsAdmin(role, scopes);
    } catch { return false; }
  });

  const login = async (email: string, password: string) => {
    const data = await api.login(email, password);
    if (!data?.token) throw new Error('Login failed: no token returned');
    localStorage.setItem('projectparty-token', data.token);
    localStorage.setItem('projectparty-email', data.email || email);
    const role = data.role || null;
    const scopes = data.scopes || null;
    localStorage.setItem('projectparty-admin', JSON.stringify({ role, scopes }));
    api.setToken(data.token);
    setToken(data.token);
    setEmail(data.email || email);
    setIsAdmin(computeIsAdmin(role, scopes));
  };

  const register = async (email: string, password: string) => {
    await api.register(email, password);
    await login(email, password);
  };

  const logout = () => {
    localStorage.removeItem('projectparty-token');
    localStorage.removeItem('projectparty-email');
    localStorage.removeItem('projectparty-admin');
    api.clearToken();
    setToken(null);
    setEmail(null);
    setIsAdmin(false);
  };

  return (
    <AuthContext.Provider value={{ token, email, isAdmin, register, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be used within AuthProvider');
  return context;
}
