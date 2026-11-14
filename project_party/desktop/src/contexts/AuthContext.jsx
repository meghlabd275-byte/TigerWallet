import React, { createContext, useContext, useState, useCallback } from 'react';
import api, { clearToken, getToken } from '../services/project-party-api';

const AuthContext = createContext();

export function AuthProvider({ children }) {
  const [token, setTokenState] = useState(() => getToken());
  const [role, setRole] = useState(() => localStorage.getItem('projectparty-role') || null);
  const [error, setError] = useState(null);

  const applySession = useCallback((data) => {
    setTokenState(getToken());
    const r = data && (data.role || data.user?.role);
    if (r) {
      localStorage.setItem('projectparty-role', r);
      setRole(r);
    }
    return data;
  }, []);

  const login = useCallback(async (username, password) => {
    setError(null);
    try {
      return applySession(await api.login(username, password));
    } catch (e) {
      setError(e.message || 'Login failed');
      throw e;
    }
  }, [applySession]);

  const register = useCallback(async (username, password) => {
    setError(null);
    try {
      return applySession(await api.register(username, password));
    } catch (e) {
      setError(e.message || 'Registration failed');
      throw e;
    }
  }, [applySession]);

  const logout = useCallback(() => {
    clearToken();
    localStorage.removeItem('projectparty-role');
    setTokenState(null);
    setRole(null);
  }, []);

  const isAdmin = role === 'admin' || role === 'super_admin' || role === 'wl_client';

  return (
    <AuthContext.Provider value={{ token, role, isAdmin, error, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
