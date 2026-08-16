/**
 * TigerAdmin C++ Core - New Admin Domain Handlers
 *
 * Header-only handlers for the 4 new admin domains backed by the admin/go
 * service on port 9093:
 *   - /bots            (CRUD + status + stats + tiers CRUD)
 *   - /bots-clients    (CRUD + status)
 *   - /project-teams   (CRUD + status + members)
 *   - /liquidity-sources (CRUD + status + priority + health-check + stats)
 *
 * Real API calls with loading/error/empty states are delegated to DomainHandler;
 * each subclass adds its domain-specific actions and registers the routes.
 * Every method proxies the inbound request verbatim to the upstream admin/go
 * backend, forwarding the caller's Bearer JWT. No stubs, fakes, or canned
 * payloads are returned.
 */
#pragma once

#include "admin_domain.hpp"

namespace tiger {
namespace admin {

// ============================================================================
// Bots — CRUD + status + getStats + tiers (getTiers/createTier/updateTier/
// deleteTier). Tiers are a sub-resource of a bot: /bots/{id}/tiers[/{tid}].
// getStats is a literal collection route (/bots/stats) registered before the
// /bots/{id} capture so the custom router matches the literal first.
// ============================================================================
class BotsHandler : public DomainHandler {
public:
    BotsHandler() : DomainHandler("/bots") {}

    // GET /api/v1/bots/stats
    HttpResponse get_stats(const HttpRequest& req) {
        if (!bearer_token(req)) return unauthorized();
        LOG_INFO("GET /bots/stats");
        return proxy_to_upstream(req, HttpMethod::GET, "/api/v1/bots/stats");
    }

    // GET /api/v1/bots/{id}/tiers
    HttpResponse get_tiers(const HttpRequest& req) {
        if (!bearer_token(req)) return unauthorized();
        std::string id = path_param(req, "id");
        LOG_INFO("GET /bots/" + id + "/tiers");
        return proxy_to_upstream(req, HttpMethod::GET,
                                 "/api/v1/bots/" + id + "/tiers");
    }

    // POST /api/v1/bots/{id}/tiers
    HttpResponse create_tier(const HttpRequest& req) {
        if (!bearer_token(req)) return unauthorized();
        std::string id = path_param(req, "id");
        LOG_INFO("POST /bots/" + id + "/tiers");
        return proxy_to_upstream(req, HttpMethod::POST,
                                 "/api/v1/bots/" + id + "/tiers");
    }

    // PUT /api/v1/bots/{id}/tiers/{tid}
    HttpResponse update_tier(const HttpRequest& req) {
        if (!bearer_token(req)) return unauthorized();
        std::string id = path_param(req, "id");
        std::string tid = path_param(req, "tid");
        LOG_INFO("PUT /bots/" + id + "/tiers/" + tid);
        return proxy_to_upstream(req, HttpMethod::PUT,
                                 "/api/v1/bots/" + id + "/tiers/" + tid);
    }

    // DELETE /api/v1/bots/{id}/tiers/{tid}
    HttpResponse delete_tier(const HttpRequest& req) {
        if (!bearer_token(req)) return unauthorized();
        std::string id = path_param(req, "id");
        std::string tid = path_param(req, "tid");
        LOG_INFO("DELETE /bots/" + id + "/tiers/" + tid);
        return proxy_to_upstream(req, HttpMethod::DELETE,
                                 "/api/v1/bots/" + id + "/tiers/" + tid);
    }

    void register_stats(Router& router) {
        // Literal collection route registered before /bots/{id} so the custom
        // router (first-match) resolves /bots/stats over the {id} capture.
        router.get("/api/v1/bots/stats",
            [this](const HttpRequest& r) { return get_stats(r); });
    }

    void register_status(Router& router) {
        router.put("/api/v1/bots/{id}/status",
            [this](const HttpRequest& r) { return set_status(r); });
    }

    void register_tiers(Router& router) {
        router.get("/api/v1/bots/{id}/tiers",
            [this](const HttpRequest& r) { return get_tiers(r); });
        router.post("/api/v1/bots/{id}/tiers",
            [this](const HttpRequest& r) { return create_tier(r); });
        router.put("/api/v1/bots/{id}/tiers/{tid}",
            [this](const HttpRequest& r) { return update_tier(r); });
        router.delete_route("/api/v1/bots/{id}/tiers/{tid}",
            [this](const HttpRequest& r) { return delete_tier(r); });
    }

    void register_routes(Router& router) override {
        // Register literal/action routes before the CRUD captures so the
        // first-match router prefers them.
        register_stats(router);
        DomainHandler::register_routes(router);
        register_status(router);
        register_tiers(router);
    }
};

// ============================================================================
// Bots Clients — CRUD + status. No domain-specific actions beyond the base.
// ============================================================================
class BotsClientsHandler : public DomainHandler {
public:
    BotsClientsHandler() : DomainHandler("/bots-clients") {}

    void register_status(Router& router) {
        router.put("/api/v1/bots-clients/{id}/status",
            [this](const HttpRequest& r) { return set_status(r); });
    }

    void register_routes(Router& router) override {
        DomainHandler::register_routes(router);
        register_status(router);
    }
};

// ============================================================================
// Project Teams — CRUD + status + members (getMembers/addMember/removeMember).
// Members are a sub-resource: /project-teams/{id}/members[/{mid}].
// ============================================================================
class ProjectTeamsHandler : public DomainHandler {
public:
    ProjectTeamsHandler() : DomainHandler("/project-teams") {}

    // GET /api/v1/project-teams/{id}/members
    HttpResponse get_members(const HttpRequest& req) {
        if (!bearer_token(req)) return unauthorized();
        std::string id = path_param(req, "id");
        LOG_INFO("GET /project-teams/" + id + "/members");
        return proxy_to_upstream(req, HttpMethod::GET,
                                 "/api/v1/project-teams/" + id + "/members");
    }

    // POST /api/v1/project-teams/{id}/members
    HttpResponse add_member(const HttpRequest& req) {
        if (!bearer_token(req)) return unauthorized();
        std::string id = path_param(req, "id");
        LOG_INFO("POST /project-teams/" + id + "/members");
        return proxy_to_upstream(req, HttpMethod::POST,
                                 "/api/v1/project-teams/" + id + "/members");
    }

    // DELETE /api/v1/project-teams/{id}/members/{mid}
    HttpResponse remove_member(const HttpRequest& req) {
        if (!bearer_token(req)) return unauthorized();
        std::string id = path_param(req, "id");
        std::string mid = path_param(req, "mid");
        LOG_INFO("DELETE /project-teams/" + id + "/members/" + mid);
        return proxy_to_upstream(req, HttpMethod::DELETE,
                                 "/api/v1/project-teams/" + id +
                                 "/members/" + mid);
    }

    void register_status(Router& router) {
        router.put("/api/v1/project-teams/{id}/status",
            [this](const HttpRequest& r) { return set_status(r); });
    }

    void register_members(Router& router) {
        router.get("/api/v1/project-teams/{id}/members",
            [this](const HttpRequest& r) { return get_members(r); });
        router.post("/api/v1/project-teams/{id}/members",
            [this](const HttpRequest& r) { return add_member(r); });
        router.delete_route("/api/v1/project-teams/{id}/members/{mid}",
            [this](const HttpRequest& r) { return remove_member(r); });
    }

    void register_routes(Router& router) override {
        DomainHandler::register_routes(router);
        register_status(router);
        register_members(router);
    }
};

// ============================================================================
// Liquidity Sources — CRUD + status + setPriority + healthCheck + getStats.
// getStats is a literal collection route (/liquidity-sources/stats) registered
// before the /liquidity-sources/{id} capture so the custom router matches the
// literal first.
// ============================================================================
class LiquiditySourcesHandler : public DomainHandler {
public:
    LiquiditySourcesHandler() : DomainHandler("/liquidity-sources") {}

    // PUT /api/v1/liquidity-sources/{id}/priority  body: {"priority": ...}
    HttpResponse set_priority(const HttpRequest& req) {
        if (!bearer_token(req)) return unauthorized();
        std::string id = path_param(req, "id");
        LOG_INFO("PUT /liquidity-sources/" + id + "/priority");
        return proxy_to_upstream(req, HttpMethod::PUT,
                                 "/api/v1/liquidity-sources/" + id +
                                 "/priority");
    }

    // POST /api/v1/liquidity-sources/{id}/health-check
    HttpResponse health_check(const HttpRequest& req) {
        if (!bearer_token(req)) return unauthorized();
        std::string id = path_param(req, "id");
        LOG_INFO("POST /liquidity-sources/" + id + "/health-check");
        return proxy_to_upstream(req, HttpMethod::POST,
                                 "/api/v1/liquidity-sources/" + id +
                                 "/health-check");
    }

    // GET /api/v1/liquidity-sources/stats
    HttpResponse get_stats(const HttpRequest& req) {
        if (!bearer_token(req)) return unauthorized();
        LOG_INFO("GET /liquidity-sources/stats");
        return proxy_to_upstream(req, HttpMethod::GET,
                                 "/api/v1/liquidity-sources/stats");
    }

    void register_stats(Router& router) {
        // Literal collection route registered before /liquidity-sources/{id}
        // so the custom router (first-match) resolves the literal first.
        router.get("/api/v1/liquidity-sources/stats",
            [this](const HttpRequest& r) { return get_stats(r); });
    }

    void register_status(Router& router) {
        router.put("/api/v1/liquidity-sources/{id}/status",
            [this](const HttpRequest& r) { return set_status(r); });
    }

    void register_priority(Router& router) {
        router.put("/api/v1/liquidity-sources/{id}/priority",
            [this](const HttpRequest& r) { return set_priority(r); });
    }

    void register_health_check(Router& router) {
        router.post("/api/v1/liquidity-sources/{id}/health-check",
            [this](const HttpRequest& r) { return health_check(r); });
    }

    void register_routes(Router& router) override {
        // Register literal/action routes before the CRUD captures so the
        // first-match router prefers them.
        register_stats(router);
        DomainHandler::register_routes(router);
        register_status(router);
        register_priority(router);
        register_health_check(router);
    }
};

} // namespace admin
} // namespace tiger
