import React, { createContext, useContext, useState, useEffect } from 'react';
import { api, setToken } from '../services/api';

const AuthContext = createContext();

export function AuthProvider({ children }) {
  const [token, setTokenState] = useState(() => localStorage.getItem('userwallet-token'));
  const [user, setUser] = useState(null);

  useEffect(() => {
    if (token) {
      setToken(token);
      try {
        const payload = token.split('.')[1];
        const decoded = JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')));
        setUser({ id: decoded.sub || '', email: decoded.email || '', username: decoded.username || decoded.email || '' });
      } catch {
        setUser(null);
      }
    }
  }, [token]);

  const login = async (email, password) => {
    const { token: newToken, user: newUser } = await api.login(email, password);
    localStorage.setItem('userwallet-token', newToken);
    setToken(newToken);
    setTokenState(newToken);
    setUser(newUser);
  };

  const register = async (email, username, password) => {
    const { token: newToken } = await api.register(email, username, password);
    localStorage.setItem('userwallet-token', newToken);
    setToken(newToken);
    setTokenState(newToken);
    setUser({ email, username });
  };

  // Guest auth — provisions an anonymous account so the user can create/import
  // a wallet without registering. Persists the token exactly like login().
  const guestAuth = async (deviceId) => {
    const { token: newToken, user: returnedUser } = await api.guestAuth(deviceId);
    localStorage.setItem('userwallet-token', newToken);
    setToken(newToken);
    setTokenState(newToken);
    setUser(returnedUser || { id: '', email: '', username: 'Guest', guest: true });
  };

  const logout = () => {
    localStorage.removeItem('userwallet-token');
    setToken('');
    setTokenState(null);
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ token, user, login, register, guestAuth, logout, isAuthenticated: !!token }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
