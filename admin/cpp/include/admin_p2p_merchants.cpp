/**
 * TigerAdmin C++ Core - P2P Merchants Domain Handler
 *
 * Header-only handler for the /p2p-merchants admin domain backed by the admin/go
 * service on port 9093. Real API calls with loading/error/empty states are
 * delegated to DomainHandler; this file adds the domain-specific actions.
 */
#pragma once

#include "admin_domain.hpp"

namespace tiger {
namespace admin {

class P2PMerchantsHandler : public DomainHandler {
public:
    P2PMerchantsHandler() : DomainHandler("/p2p-merchants") {}

    // POST /api/v1/p2p-merchants/{id}/approve
    // POST /api/v1/p2p-merchants/{id}/reject
    // GET  /api/v1/p2p-merchants/{id}/transactions
    // The admin/go backend exposes approve/reject (not setStatus) for
    // merchants, plus a transactions sub-resource.
    void register_approve_reject(Router& router) {
        router.post("/api/v1/p2p-merchants/{id}/approve",
            [this](const HttpRequest& r) { return approve(r); });
        router.post("/api/v1/p2p-merchants/{id}/reject",
            [this](const HttpRequest& r) { return reject(r); });
        router.get("/api/v1/p2p-merchants/{id}/transactions",
            [this](const HttpRequest& r) {
                LOG_INFO("GET /api/v1/p2p-merchants/" + path_param(r, "id") + "/transactions");
                return proxy_to_upstream(r, HttpMethod::GET,
                    "/api/v1/p2p-merchants/" + path_param(r, "id") + "/transactions");
            });
    }

    void register_routes(Router& router) override {
        DomainHandler::register_routes(router);
        register_approve_reject(router);
    }
};

} // namespace admin
} // namespace tiger