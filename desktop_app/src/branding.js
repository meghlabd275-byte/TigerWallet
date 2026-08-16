// TigerWallet Desktop — White-label branding config
//
// Loads WL branding for a rebranded desktop build:
//   1. `process.env.WL_BRANDING_SLUG` (Tauri injects env at build/runtime), OR
//      a sibling `wl-branding.json` config file shipped with the app.
//   2. If a slug is set, fetch `GET {CONTROL_PLANE_URL}/api/v1/branding/{slug}`
//      on app startup. The endpoint is PUBLIC (no auth) so a WL-branded app
//      fetches its branding with no secrets.
//   3. Cache the result in localStorage so a transient failure / cold start
//      still shows the WL brand instead of TigerWallet.
//   4. Fall back to TigerWallet defaults if there's no slug, the fetch fails,
//      or the endpoint returns 404 (no WL client matches the slug).
//
// The in-app displayed title + theme CSS vars are what we override at runtime
// (the OS window title is set from tauri.conf.json at build time; we also
// update document.title and the window title via the Tauri window API when
// available).

(function (global) {
    'use strict';

    const STORAGE_KEY = 'wl_branding_json';
    const STORAGE_SLUG_KEY = 'wl_branding_slug';

    // TigerWallet stock branding — the backward-compatible default.
    const DEFAULTS = Object.freeze({
        slug: '',
        app_name: 'TigerWallet',
        logo_url: '',
        primary_color: '#FF6B35',
        secondary_color: '#F7C94B',
        domain: 'tigerwallet.io',
        support_email: 'support@tigerwallet.io',
        terms_url: 'https://tigerwallet.io/terms',
        privacy_url: 'https://tigerwallet.io/privacy',
    });

    const DEFAULT_CONTROL_PLANE_URL = 'http://localhost:9008';

    function readEnvSlug() {
        // Tauri exposes process.env via the @tauri-apps/api shell env, but the
        // simplest portable path: a global injected by the Tauri build
        // (window.__WL_BRANDING_SLUG__) or process.env in node-based tooling.
        try {
            if (typeof process !== 'undefined' && process.env && process.env.WL_BRANDING_SLUG) {
                return String(process.env.WL_BRANDING_SLUG).trim();
            }
        } catch (_) { /* process not defined in webview */ }
        if (typeof global !== 'undefined' && global.__WL_BRANDING_SLUG__) {
            return String(global.__WL_BRANDING_SLUG__).trim();
        }
        return '';
    }

    function readControlPlaneUrl() {
        try {
            if (typeof process !== 'undefined' && process.env && process.env.WL_CONTROL_PLANE_URL) {
                return String(process.env.WL_CONTROL_PLANE_URL).trim();
            }
        } catch (_) { /* noop */ }
        if (typeof global !== 'undefined' && global.__WL_CONTROL_PLANE_URL__) {
            return String(global.__WL_CONTROL_PLANE_URL__).trim();
        }
        return DEFAULT_CONTROL_PLANE_URL;
    }

    // Merge a fetched/parsed branding object over the TigerWallet defaults.
    // Missing or empty fields fall back to defaults (never null/undefined).
    function mergeWithDefaults(raw) {
        if (!raw || typeof raw !== 'object') return Object.assign({}, DEFAULTS);
        const out = Object.assign({}, DEFAULTS, raw);
        for (const k of Object.keys(DEFAULTS)) {
            if (out[k] == null || out[k] === '') out[k] = DEFAULTS[k];
        }
        if (!out.slug) out.slug = raw.slug || '';
        return out;
    }

    function loadCached(slug) {
        try {
            const raw = localStorage.getItem(STORAGE_KEY);
            if (!raw) return null;
            const parsed = mergeWithDefaults(JSON.parse(raw));
            // Only trust a cache whose slug matches the current build's slug.
            if (slug && parsed.slug && parsed.slug !== slug) return null;
            return parsed;
        } catch (_) { return null; }
    }

    function persist(branding) {
        try {
            localStorage.setItem(STORAGE_KEY, JSON.stringify(branding));
            localStorage.setItem(STORAGE_SLUG_KEY, branding.slug || '');
        } catch (_) { /* storage may be unavailable; branding stays in-memory */ }
    }

    // Apply branding to the live DOM: document title + CSS custom properties.
    function applyToDom(branding) {
        try {
            document.title = branding.app_name + ' Desktop';
            const root = document.documentElement;
            if (branding.primary_color) root.style.setProperty('--accent-primary', branding.primary_color);
            if (branding.secondary_color) root.style.setProperty('--accent-secondary', branding.secondary_color);
            // Also expose as --wl-primary / --wl-secondary for WL-specific CSS.
            root.style.setProperty('--wl-primary', branding.primary_color);
            root.style.setProperty('--wl-secondary', branding.secondary_color);
            root.setAttribute('data-wl-slug', branding.slug || 'tigerwallet');
        } catch (_) { /* DOM not ready */ }
    }

    // Update the native window title when running under Tauri (best-effort).
    function applyTauriWindowTitle(branding) {
        try {
            const w = global.window;
            if (!w || !w.__TAURI__) return;
            const win = w.__TAURI__.window;
            const label = branding.app_name + ' - Secure Multi-Chain Wallet';
            if (win && win.getCurrent && win.getCurrent) {
                const current = win.getCurrent();
                if (current && typeof current.setTitle === 'function') current.setTitle(label);
            } else if (win && typeof win.setWindowTitle === 'function') {
                win.setWindowTitle(label);
            }
        } catch (_) { /* Tauri API shape varies across versions; ignore */ }
    }

    function BrandingConfig() {
        this._slug = readEnvSlug();
        this._controlPlaneUrl = readControlPlaneUrl();
        // Load synchronously from cache (or defaults) so the first paint is
        // WL-branded when a cache is present.
        const cached = loadCached(this._slug);
        this._branding = cached || mergeWithDefaults({ slug: this._slug });
        this._listeners = [];
    }

    BrandingConfig.prototype = {
        get slug() { return this._slug; },
        get branding() { return this._branding; },
        // Convenience accessors mirroring the mobile modules.
        get appName() { return this._branding.app_name; },
        get logoUrl() { return this._branding.logo_url; },
        get primaryColor() { return this._branding.primary_color; },
        get secondaryColor() { return this._branding.secondary_color; },
        get domain() { return this._branding.domain; },
        get supportEmail() { return this._branding.support_email; },

        onChange(fn) {
            if (typeof fn === 'function') this._listeners.push(fn);
        },

        // Apply current branding to the DOM + Tauri window title.
        apply() {
            applyToDom(this._branding);
            applyTauriWindowTitle(this._branding);
        },

        // Load from cache, apply, then async-refresh from the control plane.
        // Call once on app startup. Returns a Promise that resolves with the
        // final branding (cached if fetch fails).
        async bootstrap() {
            this.apply();
            if (this._slug) await this.refresh();
            return this._branding;
        },

        // Fetch `GET {CONTROL_PLANE_URL}/api/v1/branding/{slug}` and apply on
        // success. Failures (network, non-2xx, 404) are silent — the cached /
        // default branding remains (backward compatible).
        async refresh() {
            const slug = encodeURIComponent(this._slug);
            const url = `${this._controlPlaneUrl}/api/v1/branding/${slug}`;
            try {
                const res = await fetch(url, { method: 'GET', headers: { 'Accept': 'application/json' } });
                if (!res.ok) return this._branding; // 404 => no WL client matches slug
                const json = await res.json();
                const merged = mergeWithDefaults(Object.assign({}, json, { slug: this._slug }));
                this._branding = merged;
                persist(merged);
                this.apply();
                for (const fn of this._listeners) {
                    try { fn(merged); } catch (_) { /* listener error */ }
                }
                return merged;
            } catch (_) {
                return this._branding;
            }
        },
    };

    const instance = new BrandingConfig();
    global.BrandingConfig = instance;
    global.DEFAULT_BRANDING = DEFAULTS;
})(typeof window !== 'undefined' ? window : globalThis);
