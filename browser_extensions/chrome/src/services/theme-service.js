/**
 * TigerWallet Theme Service (browser extension)
 * Provides Light/Dark/System theme switching for the Chrome extension popup.
 *
 * Persists the preference to chrome.storage.local (the same store the popup
 * uses for the auth token) so it survives popup reopen, and falls back to
 * localStorage when the extension API is unavailable (e.g. unit tests).
 */

const THEME_KEY = 'tigerwallet_theme';
const THEME_LIGHT = 'light';
const THEME_DARK = 'dark';
const THEME_SYSTEM = 'system';

class ThemeService {
    static get THEME_KEY() { return THEME_KEY; }
    static get THEME_LIGHT() { return THEME_LIGHT; }
    static get THEME_DARK() { return THEME_DARK; }
    static get THEME_SYSTEM() { return THEME_SYSTEM; }

    constructor() {
        this.currentTheme = THEME_LIGHT;
        this.listeners = [];
        this.init();
    }

    async init() {
        const saved = await this.loadStored();
        if (saved) {
            this.currentTheme = saved;
        } else if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
            this.currentTheme = THEME_DARK;
        }
        this.applyTheme();

        if (window.matchMedia) {
            window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
                if (this.currentTheme === THEME_SYSTEM) {
                    this.applyResolvedTheme(e.matches ? THEME_DARK : THEME_LIGHT);
                }
            });
        }
    }

    async loadStored() {
        if (typeof chrome !== 'undefined' && chrome.storage && chrome.storage.local) {
            const result = await chrome.storage.local.get(THEME_KEY);
            return result[THEME_KEY] || null;
        }
        if (typeof localStorage !== 'undefined') {
            return localStorage.getItem(THEME_KEY);
        }
        return null;
    }

    async persist(theme) {
        if (typeof chrome !== 'undefined' && chrome.storage && chrome.storage.local) {
            await chrome.storage.local.set({ [THEME_KEY]: theme });
        } else if (typeof localStorage !== 'undefined') {
            localStorage.setItem(THEME_KEY, theme);
        }
    }

    async setTheme(theme) {
        this.currentTheme = theme;
        await this.persist(theme);
        this.applyTheme();
        this.notifyListeners();
    }

    getTheme() {
        return this.currentTheme;
    }

    isDark() {
        return this.resolveTheme() === THEME_DARK;
    }

    isLight() {
        return this.resolveTheme() === THEME_LIGHT;
    }

    isSystem() {
        return this.currentTheme === THEME_SYSTEM;
    }

    resolveTheme() {
        if (this.currentTheme !== THEME_SYSTEM) return this.currentTheme;
        if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
            return THEME_DARK;
        }
        return THEME_LIGHT;
    }

    toggle() {
        this.setTheme(this.isDark() ? THEME_LIGHT : THEME_DARK);
    }

    applyTheme() {
        this.applyResolvedTheme(this.resolveTheme());
    }

    applyResolvedTheme(resolved) {
        const root = document.documentElement;
        if (resolved === THEME_DARK) {
            root.setAttribute('data-theme', 'dark');
            root.classList.add('dark-theme');
            root.classList.remove('light-theme');
        } else {
            root.setAttribute('data-theme', 'light');
            root.classList.add('light-theme');
            root.classList.remove('dark-theme');
        }

        const dark = resolved === THEME_DARK;
        root.style.setProperty('--bg-primary', dark ? '#0a0a0a' : '#ffffff');
        root.style.setProperty('--bg-secondary', dark ? '#1a1a1a' : '#f9fafb');
        root.style.setProperty('--bg-tertiary', dark ? '#2a2a2a' : '#f3f4f6');
        root.style.setProperty('--text-primary', dark ? '#ffffff' : '#111827');
        root.style.setProperty('--text-secondary', dark ? '#a0a0a0' : '#6b7280');
        root.style.setProperty('--text-tertiary', dark ? '#707070' : '#9ca3af');
        root.style.setProperty('--border-color', dark ? '#333333' : '#e5e7eb');
        root.style.setProperty('--accent-primary', '#f59e0b');
        root.style.setProperty('--accent-secondary', '#d97706');
        root.style.setProperty('--success', dark ? '#10b981' : '#059669');
        root.style.setProperty('--error', dark ? '#ef4444' : '#dc2626');
        root.style.setProperty('--warning', dark ? '#f59e0b' : '#d97706');
        root.style.setProperty('--info', dark ? '#3b82f6' : '#2563eb');
    }

    addListener(callback) {
        this.listeners.push(callback);
    }

    removeListener(callback) {
        this.listeners = this.listeners.filter((l) => l !== callback);
    }

    notifyListeners() {
        this.listeners.forEach((callback) => callback(this.currentTheme));
    }

    getColors() {
        const dark = this.isDark();
        return {
            background: {
                primary: dark ? '#0a0a0a' : '#ffffff',
                secondary: dark ? '#1a1a1a' : '#f9fafb',
                tertiary: dark ? '#2a2a2a' : '#f3f4f6',
                card: dark ? '#1a1a1a' : '#ffffff',
                modal: dark ? '#1a1a1a' : '#ffffff',
            },
            text: {
                primary: dark ? '#ffffff' : '#111827',
                secondary: dark ? '#a0a0a0' : '#6b7280',
                tertiary: dark ? '#707070' : '#9ca3af',
                link: dark ? '#60a5fa' : '#3b82f6',
            },
            border: {
                default: dark ? '#333333' : '#e5e7eb',
                focus: '#f59e0b',
            },
            accent: {
                primary: '#f59e0b',
                secondary: '#d97706',
                hover: dark ? '#fbbf24' : '#f59e0b',
            },
            status: {
                success: dark ? '#10b981' : '#059669',
                error: dark ? '#ef4444' : '#dc2626',
                warning: dark ? '#f59e0b' : '#d97706',
                info: dark ? '#3b82f6' : '#2563eb',
            },
        };
    }
}

if (typeof window !== 'undefined') {
    window.ThemeService = new ThemeService();
}
if (typeof module !== 'undefined' && module.exports) {
    module.exports = ThemeService;
}
