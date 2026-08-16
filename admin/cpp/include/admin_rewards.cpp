/**
 * TigerAdmin C++ Core - Rewards Domain Handler
 *
 * Header-only handler for the /rewards admin domain backed by the admin/go
 * service on port 9093. Real API calls with loading/error/empty states are
 * delegated to DomainHandler; this file adds the domain-specific actions.
 */
#pragma once

#include "admin_domain.hpp"

namespace tiger {
namespace admin {

class RewardsHandler : public DomainHandler {
public:
    RewardsHandler() : DomainHandler("/rewards") {}

    // PUT /api/v1/rewards/{id}/status
    void register_status(Router& router) {
        router.put("/api/v1/rewards/{id}/status",
            [this](const HttpRequest& r) { return set_status(r); });
    }

    void register_routes(Router& router) override {
        DomainHandler::register_routes(router);
        register_status(router);
    }
};

} // namespace admin
} // namespace tiger