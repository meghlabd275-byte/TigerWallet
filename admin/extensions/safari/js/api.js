/**
 * TigerWallet Admin Extension - API Service
 * Domain API methods for the admin/go backend on port 9093.
 * Base URL: http://localhost:9093/api/v1/<domain>
 *
 * Exposes window.AdminAPI with a namespace per domain. Each domain provides
 * getAll, getOne, create, update, delete, setStatus. Bots additionally
 * exposes getStats, getTiers, createTier, updateTier, deleteTier.
 * ProjectTeams additionally exposes getMembers, addMember, removeMember.
 * LiquiditySources additionally exposes setPriority, healthCheck, getStats.
 */
(function (root) {
    'use strict';

    const API_BASE_URL = (function () {
        try {
            return localStorage.getItem('api_base_url') || 'http://localhost:9093/api/v1';
        } catch (e) {
            return 'http://localhost:9093/api/v1';
        }
    })();

    function getToken() {
        try {
            return localStorage.getItem('admin_token');
        } catch (e) {
            return null;
        }
    }

    async function request(endpoint, options) {
        options = options || {};
        const token = getToken();
        const response = await fetch(`${API_BASE_URL}${endpoint}`, {
            method: options.method || 'GET',
            body: options.body ? JSON.stringify(options.body) : undefined,
            headers: Object.assign(
                {
                    'Content-Type': 'application/json'
                },
                token ? { Authorization: `Bearer ${token}` } : {},
                options.headers || {}
            )
        });
        if (response.status === 401) {
            throw new Error('Session expired. Please log in again.');
        }
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        if (response.status === 204) return null;
        return response.json();
    }

    // ---- generic CRUD helpers shared by every domain ----
    function crud(domain) {
        return {
            getAll: function () {
                return request(`/${domain}`);
            },
            getOne: function (id) {
                return request(`/${domain}/${encodeURIComponent(id)}`);
            },
            create: function (body) {
                return request(`/${domain}`, { method: 'POST', body: body });
            },
            update: function (id, body) {
                return request(`/${domain}/${encodeURIComponent(id)}`, { method: 'PUT', body: body });
            },
            delete: function (id) {
                return request(`/${domain}/${encodeURIComponent(id)}`, { method: 'DELETE' });
            },
            setStatus: function (id, status) {
                return request(`/${domain}/${encodeURIComponent(id)}/status`, {
                    method: 'PUT',
                    body: { status: status }
                });
            }
        };
    }

    const AdminAPI = {
        // ---- bots (CRUD + stats + tiers) ----
        bots: Object.assign(crud('bots'), {
            getStats: function () {
                return request('/bots/stats');
            },
            getTiers: function (botId) {
                return request(`/bots/${encodeURIComponent(botId)}/tiers`);
            },
            createTier: function (botId, body) {
                return request(`/bots/${encodeURIComponent(botId)}/tiers`, { method: 'POST', body: body });
            },
            updateTier: function (botId, tierId, body) {
                return request(`/bots/${encodeURIComponent(botId)}/tiers/${encodeURIComponent(tierId)}`, {
                    method: 'PUT',
                    body: body
                });
            },
            deleteTier: function (botId, tierId) {
                return request(`/bots/${encodeURIComponent(botId)}/tiers/${encodeURIComponent(tierId)}`, {
                    method: 'DELETE'
                });
            }
        }),

        // ---- bots-clients (CRUD) ----
        'bots-clients': crud('bots-clients'),

        // ---- project-teams (CRUD + members) ----
        'project-teams': Object.assign(crud('project-teams'), {
            getMembers: function (teamId) {
                return request(`/project-teams/${encodeURIComponent(teamId)}/members`);
            },
            addMember: function (teamId, body) {
                return request(`/project-teams/${encodeURIComponent(teamId)}/members`, {
                    method: 'POST',
                    body: body
                });
            },
            removeMember: function (teamId, memberId) {
                return request(
                    `/project-teams/${encodeURIComponent(teamId)}/members/${encodeURIComponent(memberId)}`,
                    { method: 'DELETE' }
                );
            }
        }),

        // ---- liquidity-sources (CRUD + priority + health + stats) ----
        'liquidity-sources': Object.assign(crud('liquidity-sources'), {
            setPriority: function (id, priority) {
                return request(`/liquidity-sources/${encodeURIComponent(id)}/priority`, {
                    method: 'PUT',
                    body: { priority: priority }
                });
            },
            healthCheck: function (id) {
                return request(`/liquidity-sources/${encodeURIComponent(id)}/health`);
            },
            getStats: function () {
                return request('/liquidity-sources/stats');
            }
        }),

        // low-level helper for ad-hoc calls / background script reuse
        request: request,
        baseURL: API_BASE_URL
    };

    root.AdminAPI = AdminAPI;
})(typeof window !== 'undefined' ? window : this);
