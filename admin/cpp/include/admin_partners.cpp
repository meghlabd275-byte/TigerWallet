/**
 * TigerAdmin C++ Core - Partners Domain Handler
 *
 * Header-only handler for the /partners admin domain backed by the admin/go
 * service on port 9093. Real API calls with loading/error/empty states are
 * delegated to DomainHandler; this file adds the domain-specific actions.
 */
#pragma once

#include "admin_domain.hpp"

namespace tiger {
namespace admin {

class PartnersHandler : public DomainHandler {
public:
    PartnersHandler() : DomainHandler("/partners") {}

    // PUT /api/v1/partners/{id}/status
    void register_status(Router& router) {
        router.put("/api/v1/partners/{id}/status",
            [this](const HttpRequest& r) { return set_status(r); });
    }

    // POST /api/v1/partners/{id}/approve  and  POST /api/v1/partners/{id}/reject
    void register_approve_reject(Router& router) {
        router.post("/api/v1/partners/{id}/approve",
            [this](const HttpRequest& r) { return approve(r); });
        router.post("/api/v1/partners/{id}/reject",
            [this](const HttpRequest& r) { return reject(r); });
    }

    void register_routes(Router& router) override {
        DomainHandler::register_routes(router);
        register_status(router);
        register_approve_reject(router);
    }
};

} // namespace admin
} // namespace tiger