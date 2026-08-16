// Auth Context - Authentication State for Bots
import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { api } from '../services/api';

interface User {
  id: string;
  email: string;
  role?: string;
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, role?: string) => Promise<void>;
  logout: () => void;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(() => {
    const savedEmail = localStorage.getItem('bots-email');
    const savedRole = localStorage.getItem('bots-role');
    return savedEmail ? { id: '', email: savedEmail, role: savedRole || undefined } : null;
  });
  const [token, setToken] = useState<string | null>(() => localStorage.getItem('bots-token'));

  useEffect(() => {
    if (token) {
      api.setToken(token);
    } else {
      api.clearToken();
    }
  }, [token]);

  const login = async (email: string, password: string) => {
    const data = await api.login(email, password);
    localStorage.setItem('bots-token', data.token);
    localStorage.setItem('bots-email', data.email);
    if (data.role) localStorage.setItem('bots-role', data.role);
    api.setToken(data.token);
    setToken(data.token);
    setUser({ id: data.user_id, email: data.email, role: data.role });
  };

  const register = async (email: string, password: string, role?: string) => {
    await api.register(email, password, role);
    // Registration creates the account; sign in immediately to obtain a JWT.
    await login(email, password);
  };

  const logout = () => {
    localStorage.removeItem('bots-token');
    localStorage.removeItem('bots-email');
    localStorage.removeItem('bots-role');
    api.clearToken();
    setToken(null);
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ user, token, login, register, logout, isAuthenticated: !!token }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be used within AuthProvider');
  return context;
}
