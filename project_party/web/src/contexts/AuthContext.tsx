// Auth Context — WL-ProjectParty. Real register/login through the API service
// (proxied to the WL standalone backend). No hardcoded hosts.
import React, { createContext, useContext, useState, ReactNode } from 'react';
import { api } from '../services/api';

interface AuthContextType {
  token: string | null;
  email: string | null;
  register: (email: string, password: string) => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
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

  const login = async (email: string, password: string) => {
    const data = await api.login(email, password);
    if (!data?.token) throw new Error('Login failed: no token returned');
    localStorage.setItem('projectparty-token', data.token);
    localStorage.setItem('projectparty-email', data.email || email);
    api.setToken(data.token);
    setToken(data.token);
    setEmail(data.email || email);
  };

  const register = async (email: string, password: string) => {
    await api.register(email, password);
    await login(email, password);
  };

  const logout = () => {
    localStorage.removeItem('projectparty-token');
    localStorage.removeItem('projectparty-email');
    api.clearToken();
    setToken(null);
    setEmail(null);
  };

  return (
    <AuthContext.Provider value={{ token, email, register, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be used within AuthProvider');
  return context;
}
