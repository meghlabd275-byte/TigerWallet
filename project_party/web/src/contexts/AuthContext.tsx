// Auth Context - ProjectParty
import React, { createContext, useContext, useState, ReactNode } from 'react';
import { api } from '../services/api';

const API_URL = 'http://localhost:8106/api/v1';

interface AuthContextType {
  token: string | null;
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

  const login = async (email: string, password: string) => {
    const res = await fetch(`${API_URL}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: email, password })
    });
    if (!res.ok) throw new Error('Login failed');
    const data = await res.json();
    localStorage.setItem('projectparty-token', data.token);
    api.setToken(data.token);
    setToken(data.token);
  };

  const logout = () => {
    localStorage.removeItem('projectparty-token');
    api.clearToken();
    setToken(null);
  };

  return (
    <AuthContext.Provider value={{ token, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be used within AuthProvider');
  return context;
}
