'use client';
import { useState, useEffect, createContext, useContext, ReactNode } from 'react';

type Theme = 'light' | 'dark' | 'system';
interface ThemeContextType { theme: Theme; setTheme: (t: Theme) => void; isDark: boolean }
const ThemeContext = createContext<ThemeContextType | undefined>(undefined);
export const useTheme = () => { const c = useContext(ThemeContext); if (!c) throw new Error('useTheme error'); return c; };

function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setTheme] = useState<Theme>('dark');
  const [isDark, setIsDark] = useState(true);
  useEffect(() => {
    const s = localStorage.getItem('tw-theme') as Theme;
    if (s) setTheme(s);
  }, []);
  useEffect(() => {
    const root = document.documentElement;
    const dark = theme === 'system' ? window.matchMedia('(prefers-color-scheme: dark)').matches : theme === 'dark';
    setIsDark(dark);
    root.classList.toggle('dark', dark);
    localStorage.setItem('tw-theme', theme);
  }, [theme]);
  return <ThemeContext.Provider value={{ theme, setTheme, isDark }}>{children}</ThemeContext.Provider>;
}

interface User { id: string; email: string; username: string; role: string; whiteLabelId?: string; }
interface UserContextType { user: User | null; login: (e: string, p: string) => Promise<void>; logout: () => void; loading: boolean }
const UserContext = createContext<UserContextType | undefined>(undefined);
export const useUser = () => { const c = useContext(UserContext); if (!c) throw new Error('useUser error'); return c; };

function UserProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  useEffect(() => { const u = localStorage.getItem('tw-user'); if (u) setUser(JSON.parse(u)); setLoading(false); }, []);
  const login = async (email: string, password: string) => {
    setLoading(true);
    try {
      const res = await fetch('/api/auth/login', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({ email, password }) });
      if (!res.ok) throw new Error('Login failed');
      const u = await res.json(); setUser(u); localStorage.setItem('tw-user', JSON.stringify(u));
    } finally { setLoading(false); }
  };
  const logout = () => { setUser(null); localStorage.removeItem('tw-user'); };
  return <UserContext.Provider value={{ user, login, logout, loading }}>{children}</UserContext.Provider>;
}

function Navbar() {
  const { isDark, setTheme } = useTheme();
  const { user, logout } = useUser();
  const [scrolled, setScrolled] = useState(false);
  useEffect(() => { const h = () => setScrolled(window.scrollY > 10); window.addEventListener('scroll', h); return () => window.removeEventListener('scroll', h); }, []);
  return (
    <nav className={`fixed top-0 left-0 right-0 z-50 transition-all ${scrolled ? 'bg-white/90 dark:bg-gray-900/90 backdrop-blur shadow-lg' : 'bg-transparent'}`}>
      <div className="max-w-7xl mx-auto px-4 h-16 flex items-center justify-between">
        <a href="/" className="flex items-center space-x-2">
          <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-orange-500 to-red-600 flex items-center justify-center"><span className="text-white font-bold">T</span></div>
          <span className="text-xl font-bold">TigerWallet</span>
        </a>
        <div className="hidden md:flex items-center space-x-6">
          <a href="/wallet" className="hover:text-orange-500">Wallet</a>
          <a href="/swap" className="hover:text-orange-500">Swap</a>
          <a href="/stake" className="hover:text-orange-500">Stake</a>
          <a href="/bridge" className="hover:text-orange-500">Bridge</a>
          {user?.role === 'admin' && <a href="/admin" className="hover:text-orange-500">Admin</a>}
          {user?.role === 'super_admin' && <a href="/super-admin" className="hover:text-orange-500">Super Admin</a>}
        </div>
        <div className="flex items-center space-x-4">
          <button onClick={() => setTheme(isDark ? 'light' : 'dark')} className="p-2 rounded-lg bg-gray-100 dark:bg-gray-800">
            {isDark ? <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" /></svg> : <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" /></svg>}
          </button>
          {user ? (
            <div className="flex items-center space-x-2">
              <span>{user.username}</span>
              <button onClick={logout} className="px-3 py-1 bg-red-500 text-white rounded hover:bg-red-600">Logout</button>
            </div>
          ) : (
            <div className="flex space-x-2">
              <a href="/login" className="px-4 py-2">Login</a>
              <a href="/register" className="px-4 py-2 bg-orange-500 text-white rounded hover:bg-orange-600">Register</a>
            </div>
          )}
        </div>
      </div>
    </nav>
  );
}

function Footer() {
  return (
    <footer className="bg-gray-50 dark:bg-gray-900 border-t border-gray-200 dark:border-gray-800 py-8">
      <div className="max-w-7xl mx-auto px-4 text-center text-gray-500">
        <p>© {new Date().getFullYear()} TigerWallet. All rights reserved.</p>
        <div className="flex justify-center space-x-4 mt-4">
          <a href="/privacy" className="hover:text-orange-500">Privacy</a>
          <a href="/terms" className="hover:text-orange-500">Terms</a>
          <a href="/security" className="hover:text-orange-500">Security</a>
        </div>
      </div>
    </footer>
  );
}

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className="bg-white dark:bg-gray-950 text-gray-900 dark:text-gray-100 transition-colors">
        <ThemeProvider>
          <UserProvider>
            <div className="min-h-screen flex flex-col">
              <Navbar />
              <main className="flex-1 pt-16">{children}</main>
              <Footer />
            </div>
          </UserProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
