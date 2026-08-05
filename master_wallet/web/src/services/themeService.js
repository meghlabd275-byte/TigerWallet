// TigerWallet MasterWallet - Theme Service
// Light/Dark theme switching for all apps
// Production-ready

class ThemeService {
  constructor() {
    this.currentTheme = 'light';
    this.listeners = [];
    this.isInitialized = false;
  }

  async initialize() {
    if (this.isInitialized) return true;
    
    try {
      // Load saved theme preference
      await this.loadThemePreference();
      
      // Apply theme immediately
      this.applyTheme();
      
      // Listen for system theme changes
      this.watchSystemTheme();
      
      this.isInitialized = true;
      return true;
    } catch (error) {
      console.error('ThemeService initialization failed:', error);
      return false;
    }
  }

  async loadThemePreference() {
    try {
      // Try to get from storage (extension)
      if (typeof chrome !== 'undefined' && chrome.storage) {
        const result = await chrome.storage.local.get('themePreference');
        if (result.themePreference) {
          this.currentTheme = result.themePreference;
          return;
        }
      }
      
      // Try localStorage (web)
      const savedTheme = localStorage.getItem('themePreference');
      if (savedTheme) {
        this.currentTheme = savedTheme;
        return;
      }
      
      // Check system preference
      if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
        this.currentTheme = 'dark';
      }
    } catch (error) {
      console.error('Failed to load theme preference:', error);
    }
  }

  async saveThemePreference() {
    try {
      // Save to storage (extension)
      if (typeof chrome !== 'undefined' && chrome.storage) {
        await chrome.storage.local.set({
          themePreference: this.currentTheme,
        });
      }
      
      // Save to localStorage (web)
      localStorage.setItem('themePreference', this.currentTheme);
    } catch (error) {
      console.error('Failed to save theme preference:', error);
    }
  }

  applyTheme() {
    // Apply theme to document
    document.documentElement.setAttribute('data-theme', this.currentTheme);
    
    // Add/remove dark class
    if (this.currentTheme === 'dark') {
      document.documentElement.classList.add('dark');
      document.documentElement.classList.remove('light');
    } else {
      document.documentElement.classList.add('light');
      document.documentElement.classList.remove('dark');
    }
    
    // Update meta theme-color
    const metaTheme = document.querySelector('meta[name="theme-color"]');
    if (metaTheme) {
      metaTheme.setAttribute('content', this.currentTheme === 'dark' ? '#1a1a1a' : '#ffffff');
    }
    
    // Update background
    this.applyBackgroundColors();
    
    // Update text colors
    this.applyTextColors();
    
    // Update component colors
    this.applyComponentColors();
    
    // Notify listeners
    this.notifyListeners();
  }

  applyBackgroundColors() {
    const colors = this.getThemeColors();
    
    // Body background
    document.body.style.backgroundColor = colors.background;
    document.body.style.color = colors.text;
    
    // Apply to common elements
    const bgElements = [
      'header', 'nav', 'main', 'footer', 'aside',
      '.card', '.modal', '.dropdown', '.sidebar',
      '.toolbar', '.status-bar', '.panel'
    ];
    
    bgElements.forEach(selector => {
      try {
        const elements = document.querySelectorAll(selector);
        elements.forEach(el => {
          el.style.backgroundColor = colors.surface;
          el.style.borderColor = colors.border;
        });
      } catch (e) {
        // Ignore invalid selectors
      }
    });
  }

  applyTextColors() {
    const colors = this.getThemeColors();
    
    // Apply text colors
    document.body.style.color = colors.text;
    
    // Headings
    const headings = document.querySelectorAll('h1, h2, h3, h4, h5, h6');
    headings.forEach(h => {
      h.style.color = colors.heading;
    });
    
    // Links
    const links = document.querySelectorAll('a');
    links.forEach(a => {
      a.style.color = colors.link;
    });
  }

  applyComponentColors() {
    const colors = this.getThemeColors();
    
    // Buttons
    const buttons = document.querySelectorAll('button, .btn');
    buttons.forEach(btn => {
      if (!btn.classList.contains('btn-outline') && !btn.classList.contains('btn-ghost')) {
        btn.style.backgroundColor = colors.primary;
        btn.style.color = colors.onPrimary;
        btn.style.borderColor = colors.primary;
      }
    });
    
    // Inputs
    const inputs = document.querySelectorAll('input, textarea, select');
    inputs.forEach(input => {
      input.style.backgroundColor = colors.input;
      input.style.borderColor = colors.border;
      input.style.color = colors.text;
    });
    
    // Tables
    const tables = document.querySelectorAll('table');
    tables.forEach(table => {
      table.style.borderColor = colors.border;
      
      const cells = table.querySelectorAll('th, td');
      cells.forEach(cell => {
        cell.style.borderColor = colors.border;
      });
      
      const headers = table.querySelectorAll('th');
      headers.forEach(th => {
        th.style.backgroundColor = colors.surface;
        th.style.color = colors.heading;
      });
    });
  }

  getThemeColors() {
    if (this.currentTheme === 'dark') {
      return {
        // Dark theme colors
        background: '#0a0a0a',
        surface: '#1a1a1a',
        surfaceElevated: '#242424',
        primary: '#3b82f6',
        primaryHover: '#2563eb',
        secondary: '#6366f1',
        accent: '#8b5cf6',
        text: '#e5e5e5',
        textSecondary: '#a3a3a3',
        textMuted: '#737373',
        heading: '#f5f5f5',
        link: '#60a5fa',
        border: '#333333',
        borderLight: '#404040',
        input: '#262626',
        success: '#22c55e',
        warning: '#f59e0b',
        error: '#ef4444',
        info: '#3b82f6',
        onPrimary: '#ffffff',
        onSurface: '#e5e5e5',
        overlay: 'rgba(0, 0, 0, 0.5)',
        shadow: 'rgba(0, 0, 0, 0.3)',
      };
    } else {
      return {
        // Light theme colors
        background: '#ffffff',
        surface: '#f9fafb',
        surfaceElevated: '#ffffff',
        primary: '#3b82f6',
        primaryHover: '#2563eb',
        secondary: '#6366f1',
        accent: '#8b5cf6',
        text: '#171717',
        textSecondary: '#525252',
        textMuted: '#a3a3a3',
        heading: '#0a0a0a',
        link: '#2563eb',
        border: '#e5e5e5',
        borderLight: '#f5f5f5',
        input: '#ffffff',
        success: '#16a34a',
        warning: '#d97706',
        error: '#dc2626',
        info: '#2563eb',
        onPrimary: '#ffffff',
        onSurface: '#171717',
        overlay: 'rgba(0, 0, 0, 0.1)',
        shadow: 'rgba(0, 0, 0, 0.1)',
      };
    }
  }

  watchSystemTheme() {
    if (window.matchMedia) {
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
      
      const handleChange = (e) => {
        // Only auto-switch if user hasn't set a preference
        const savedTheme = localStorage.getItem('themePreference');
        if (!savedTheme) {
          this.setTheme(e.matches ? 'dark' : 'light');
        }
      };
      
      mediaQuery.addEventListener('change', handleChange);
    }
  }

  setTheme(theme) {
    if (theme !== 'light' && theme !== 'dark') {
      console.error('Invalid theme:', theme);
      return;
    }
    
    this.currentTheme = theme;
    this.applyTheme();
    this.saveThemePreference();
  }

  toggleTheme() {
    this.setTheme(this.currentTheme === 'light' ? 'dark' : 'light');
  }

  getTheme() {
    return this.currentTheme;
  }

  isDark() {
    return this.currentTheme === 'dark';
  }

  addListener(callback) {
    this.listeners.push(callback);
  }

  removeListener(callback) {
    const index = this.listeners.indexOf(callback);
    if (index >= 0) {
      this.listeners.splice(index, 1);
    }
  }

  notifyListeners() {
    this.listeners.forEach(callback => {
      try {
        callback(this.currentTheme);
      } catch (error) {
        console.error('Theme listener error:', error);
      }
    });
  }

  // Apply theme to specific element
  applyToElement(element, type = 'auto') {
    const colors = this.getThemeColors();
    
    if (type === 'card') {
      element.style.backgroundColor = colors.surface;
      element.style.borderColor = colors.border;
      element.style.boxShadow = `0 1px 3px ${colors.shadow}`;
    } else if (type === 'button') {
      element.style.backgroundColor = colors.primary;
      element.style.color = colors.onPrimary;
      element.style.border = 'none';
      element.style.borderRadius = '8px';
      element.style.padding = '10px 20px';
      element.style.cursor = 'pointer';
      element.style.transition = 'all 0.2s';
    } else if (type === 'input') {
      element.style.backgroundColor = colors.input;
      element.style.borderColor = colors.border;
      element.style.color = colors.text;
      element.style.borderRadius = '8px';
      element.style.padding = '10px';
    } else if (type === 'modal') {
      element.style.backgroundColor = colors.surface;
      element.style.color = colors.text;
      element.style.borderRadius = '12px';
      element.style.boxShadow = `0 25px 50px ${colors.shadow}`;
    }
    
    return element;
  }

  // Create themed CSS variables
  createCSSVariables() {
    const colors = this.getThemeColors();
    
    const cssVariables = `
      :root {
        --color-background: ${colors.background};
        --color-surface: ${colors.surface};
        --color-surface-elevated: ${colors.surfaceElevated};
        --color-primary: ${colors.primary};
        --color-primary-hover: ${colors.primaryHover};
        --color-secondary: ${colors.secondary};
        --color-accent: ${colors.accent};
        --color-text: ${colors.text};
        --color-text-secondary: ${colors.textSecondary};
        --color-text-muted: ${colors.textMuted};
        --color-heading: ${colors.heading};
        --color-link: ${colors.link};
        --color-border: ${colors.border};
        --color-border-light: ${colors.borderLight};
        --color-input: ${colors.input};
        --color-success: ${colors.success};
        --color-warning: ${colors.warning};
        --color-error: ${colors.error};
        --color-info: ${colors.info};
        --color-on-primary: ${colors.onPrimary};
        --color-on-surface: ${colors.onSurface};
        --color-overlay: ${colors.overlay};
        --color-shadow: ${colors.shadow};
      }
    `;
    
    return cssVariables;
  }

  injectCSSVariables() {
    // Remove existing style element
    const existing = document.getElementById('theme-variables');
    if (existing) {
      existing.remove();
    }
    
    // Create new style element
    const style = document.createElement('style');
    style.id = 'theme-variables';
    style.textContent = this.createCSSVariables();
    
    // Inject into head
    document.head.appendChild(style);
  }
}

// Export
if (typeof module !== 'undefined' && module.exports) {
  module.exports = ThemeService;
}

// Auto-initialize when DOM is ready
if (typeof document !== 'undefined') {
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
      const themeService = new ThemeService();
      themeService.initialize();
      window.themeService = themeService;
    });
  } else {
    const themeService = new ThemeService();
    themeService.initialize();
    window.themeService = themeService;
  }
}
