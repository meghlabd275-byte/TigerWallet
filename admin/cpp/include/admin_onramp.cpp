/**
 * TigerAdmin C++ Core - Onramp Domain Handler
 *
 * Header-only handler for the /onramp admin domain backed by the admin/go
 * service on port 9093. Real API calls with loading/error/empty states are
 * delegated to DomainHandler; this file adds the domain-specific actions.
 */
#pragma once

#include "admin_domain.hpp"

namespace tiger {
namespace admin {

class OnrampHandler : public DomainHandler {
public:
    OnrampHandler() : DomainHandler("/onramp") {}

    // POST /api/v1/onramp/{id}/approve  and  POST /api/v1/onramp/{id}/reject
    void register_approve_reject(Router& router) {
        router.post("/api/v1/onramp/{id}/approve",
            [this](const HttpRequest& r) { return approve(r); });
        router.post("/api/v1/onramp/{id}/reject",
            [this](const HttpRequest& r) { return reject(r); });
    }

    void register_routes(Router& router) override {
        DomainHandler::register_routes(router);
        register_approve_reject(router);
    }
};

} // namespace admin
} // namespace tiger