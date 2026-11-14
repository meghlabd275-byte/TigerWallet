import React, { createContext, useContext, useState, useCallback } from 'react';
import api, { clearToken, getToken } from '../services/bot-api';

const AuthContext = createContext();

export function AuthProvider({ children }) {
  const [token, setTokenState] = useState(() => getToken());
  const [user, setUser] = useState(null);
  const [error, setError] = useState(null);

  const login = useCallback(async (username, password) => {
    setError(null);
    try {
      await api.login(username, password);
      setTokenState(getToken());
      const me = await api.currentBotUser();
      setUser(me);
      return me;
    } catch (e) {
      setError(e.message || 'Login failed');
      throw e;
    }
  }, []);

  const register = useCallback(async (username, password, email) => {
    setError(null);
    try {
      await api.register(username, password, email);
      setTokenState(getToken());
      return await api.currentBotUser();
    } catch (e) {
      setError(e.message || 'Registration failed');
      throw e;
    }
  }, []);

  const logout = useCallback(async () => {
    try {
      await api.logout();
    } catch (_) {
      // the local session is dropped even if the backend call fails
    }
    clearToken();
    setTokenState(null);
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider value={{ token, user, error, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
