/**
 * Auth Context - User Authentication
 */

import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { AuthService, User, LoginCredentials, RegisterData } from '../services/AuthService';
import { WalletService } from '../services/WalletService';

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  login: (credentials: LoginCredentials) => Promise<void>;
  register: (data: RegisterData) => Promise<void>;
  guestAuth: (deviceId: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [authService] = useState(() => new AuthService());
  const [walletService] = useState(() => new WalletService());

  useEffect(() => {
    checkAuth();
  }, []);

  const checkAuth = async () => {
    const token = localStorage.getItem('tigerwallet-token');
    if (token) {
      try {
        const userData = await authService.getCurrentUser();
        setUser(userData);
      } catch (err) {
        localStorage.removeItem('tigerwallet-token');
      }
    }
    setIsLoading(false);
  };

  const login = async (credentials: LoginCredentials) => {
    setIsLoading(true);
    setError(null);
    
    try {
      const { user: userData, token } = await authService.login(credentials);
      localStorage.setItem('tigerwallet-token', token);
      setUser(userData);
    } catch (err: any) {
      setError(err.message || 'Login failed');
      throw err;
    } finally {
      setIsLoading(false);
    }
  };

  const register = async (data: RegisterData) => {
    setIsLoading(true);
    setError(null);
    
    try {
      const { user: userData, token } = await authService.register(data);
      localStorage.setItem('tigerwallet-token', token);
      setUser(userData);
    } catch (err: any) {
      setError(err.message || 'Registration failed');
      throw err;
    } finally {
      setIsLoading(false);
    }
  };

  const guestAuth = async (deviceId: string) => {
    setIsLoading(true);
    setError(null);

    try {
      const { token } = await walletService.guestAuth(deviceId);
      if (token) {
        localStorage.setItem('tigerwallet-token', token);
      }
      setUser({ id: '', email: '', username: 'Guest', guest: true } as unknown as User);
    } catch (err: any) {
      setError(err.message || 'Guest authentication failed');
      throw err;
    } finally {
      setIsLoading(false);
    }
  };

  const logout = () => {
    localStorage.removeItem('tigerwallet-token');
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{
      user,
      isAuthenticated: !!user,
      isLoading,
      error,
      login,
      register,
      guestAuth,
      logout,
    }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}

export default AuthContext;
