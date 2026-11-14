/**
 * Shared storage for the TigerWallet Admin extension.
 *
 * A MV3 service worker has no `localStorage` and no DOM, so the previous code
 * crashed on load and the popup and worker could never share the auth token.
 * `chrome.storage.local` is the one store both contexts can reach.
 *
 * Contexts that do have a DOM (the popup) read through a cache that is
 * hydrated once at startup, so existing synchronous call sites keep working.
 */
(function (root) {
    'use strict';

    var api = (typeof browser !== 'undefined' && browser.storage) ? browser : chrome;

    var cache = Object.create(null);
    var hydrated = false;

    function get(keys) {
        return api.storage.local.get(keys);
    }

    function set(items) {
        return api.storage.local.set(items);
    }

    function remove(keys) {
        return api.storage.local.remove(keys);
    }

    root.TWStore = {
        /** Async API — use this in the service worker. */
        getAsync: function (key, fallback) {
            return get(key).then(function (res) {
                var v = res ? res[key] : undefined;
                return v === undefined ? fallback : v;
            });
        },
        setAsync: function (key, value) {
            var items = {};
            items[key] = value;
            return set(items);
        },
        removeAsync: function (key) {
            return remove(key);
        },

        /** Cache-backed sync API — call hydrate() first. */
        hydrate: function () {
            if (hydrated) return Promise.resolve(cache);
            return get(null).then(function (all) {
                Object.assign(cache, all || {});
                hydrated = true;
                return cache;
            });
        },
        isHydrated: function () {
            return hydrated;
        },
        getItem: function (key) {
            return cache[key] === undefined ? null : cache[key];
        },
        setItem: function (key, value) {
            cache[key] = value;
            var items = {};
            items[key] = value;
            return set(items);
        },
        removeItem: function (key) {
            delete cache[key];
            return remove(key);
        }
    };
})(typeof self !== 'undefined' ? self : this);
