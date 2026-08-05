/**
 * TigerWallet Theme Service
 * Provides Light/Dark theme switching across all platforms
 */

class ThemeService {
    static const String THEME_KEY = 'tigerwallet_theme';
    static const String THEME_LIGHT = 'light';
    static const String THEME_DARK = 'dark';
    static const String THEME_SYSTEM = 'system';
    
    constructor() {
        this.currentTheme = THEME_LIGHT;
        this.listeners = [];
        this.init();
    }
    
    init() {
        // Load saved theme preference
        const saved = localStorage.getItem(THEME_KEY);
        if (saved) {
            this.currentTheme = saved;
        } else {
            // Check system preference
            if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
                this.currentTheme = THEME_DARK;
            }
        }
        
        // Apply theme
        this.applyTheme();
        
        // Listen for system preference changes
        if (window.matchMedia) {
            window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
                if (localStorage.getItem(THEME_KEY) === THEME_SYSTEM) {
                    this.setTheme(e.matches ? THEME_DARK : THEME_LIGHT);
                }
            });
        }
    }
    
    setTheme(theme) {
        this.currentTheme = theme;
        localStorage.setItem(THEME_KEY, theme);
        this.applyTheme();
        this.notifyListeners();
    }
    
    getTheme() {
        return this.currentTheme;
    }
    
    isDark() {
        return this.currentTheme === THEME_DARK;
    }
    
    isLight() {
        return this.currentTheme === THEME_LIGHT;
    }
    
    isSystem() {
        return this.currentTheme === THEME_SYSTEM;
    }
    
    toggle() {
        this.setTheme(this.isDark() ? THEME_LIGHT : THEME_DARK);
    }
    
    applyTheme() {
        const root = document.documentElement;
        
        if (this.currentTheme === THEME_DARK) {
            root.setAttribute('data-theme', 'dark');
            root.classList.add('dark-theme');
            root.classList.remove('light-theme');
        } else {
            root.setAttribute('data-theme', 'light');
            root.classList.add('light-theme');
            root.classList.remove('dark-theme');
        }
        
        // Set CSS variables for theme
        if (this.currentTheme === THEME_DARK) {
            root.style.setProperty('--bg-primary', '#0a0a0a');
            root.style.setProperty('--bg-secondary', '#1a1a1a');
            root.style.setProperty('--bg-tertiary', '#2a2a2a');
            root.style.setProperty('--text-primary', '#ffffff');
            root.style.setProperty('--text-secondary', '#a0a0a0');
            root.style.setProperty('--text-tertiary', '#707070');
            root.style.setProperty('--border-color', '#333333');
            root.style.setProperty('--accent-primary', '#f59e0b');
            root.style.setProperty('--accent-secondary', '#d97706');
            root.style.setProperty('--success', '#10b981');
            root.style.setProperty('--error', '#ef4444');
            root.style.setProperty('--warning', '#f59e0b');
            root.style.setProperty('--info', '#3b82f6');
        } else {
            root.style.setProperty('--bg-primary', '#ffffff');
            root.style.setProperty('--bg-secondary', '#f9fafb');
            root.style.setProperty('--bg-tertiary', '#f3f4f6');
            root.style.setProperty('--text-primary', '#111827');
            root.style.setProperty('--text-secondary', '#6b7280');
            root.style.setProperty('--text-tertiary', '#9ca3af');
            root.style.setProperty('--border-color', '#e5e7eb');
            root.style.setProperty('--accent-primary', '#f59e0b');
            root.style.setProperty('--accent-secondary', '#d97706');
            root.style.setProperty('--success', '#059669');
            root.style.setProperty('--error', '#dc2626');
            root.style.setProperty('--warning', '#d97706');
            root.style.setProperty('--info', '#2563eb');
        }
    }
    
    addListener(callback) {
        this.listeners.push(callback);
    }
    
    removeListener(callback) {
        this.listeners = this.listeners.filter(l => l !== callback);
    }
    
    notifyListeners() {
        this.listeners.forEach(callback => callback(this.currentTheme));
    }
    
    // Get theme colors for specific components
    getColors() {
        return {
            background: {
                primary: this.isDark() ? '#0a0a0a' : '#ffffff',
                secondary: this.isDark() ? '#1a1a1a' : '#f9fafb',
                tertiary: this.isDark() ? '#2a2a2a' : '#f3f4f6',
                card: this.isDark() ? '#1a1a1a' : '#ffffff',
                modal: this.isDark() ? '#1a1a1a' : '#ffffff',
            },
            text: {
                primary: this.isDark() ? '#ffffff' : '#111827',
                secondary: this.isDark() ? '#a0a0a0' : '#6b7280',
                tertiary: this.isDark() ? '#707070' : '#9ca3af',
                link: this.isDark() ? '#60a5fa' : '#3b82f6',
            },
            border: {
                default: this.isDark() ? '#333333' : '#e5e7eb',
                focus: this.isDark() ? '#f59e0b' : '#d97706',
            },
            accent: {
                primary: '#f59e0b',
                secondary: '#d97706',
                hover: this.isDark() ? '#fbbf24' : '#f59e0b',
            },
            status: {
                success: this.isDark() ? '#10b981' : '#059669',
                error: this.isDark() ? '#ef4444' : '#dc2626',
                warning: this.isDark() ? '#f59e0b' : '#d97706',
                info: this.isDark() ? '#3b82f6' : '#2563eb',
            },
            gradient: {
                primary: this.isDark() 
                    ? 'linear-gradient(135deg, #f59e0b 0%, #d97706 100%)'
                    : 'linear-gradient(135deg, #f59e0b 0%, #d97706 100%)',
                dark: this.isDark()
                    ? 'linear-gradient(135deg, #1a1a1a 0%, #0a0a0a 100%)'
                    : 'linear-gradient(135deg, #ffffff 0%, #f9fafb 100%)',
            }
        };
    }
    
    // Get icon set based on theme
    getIcons() {
        return {
            sun: this.isDark() ? '☀️' : '☀️',
            moon: this.isDark() ? '🌙' : '🌙',
            theme: this.isDark() ? '🌙' : '☀️',
        };
    }
}

// Export for different environments
if (typeof window !== 'undefined') {
    window.ThemeService = new ThemeService();
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = ThemeService;
}
