// Auth Context - Authentication State
import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { api } from '../services/api';

interface User {
  id: string;
  email: string;
  username: string;
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, username: string, password: string) => Promise<void>;
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
    const { token: newToken } = await api.register(email, username, password);
    localStorage.setItem('userwallet-token', newToken);
    setToken(newToken);
    api.setToken(newToken);
    setUser({ id: '', email, username });
  };

  const logout = () => {
    localStorage.removeItem('userwallet-token');
    setToken(null);
    setUser(null);
    api.setToken('');
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
