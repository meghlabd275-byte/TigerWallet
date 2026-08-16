/**
 * TigerAdmin C++ Core - Options Domain Handler
 *
 * Header-only handler for the /options admin domain backed by the admin/go
 * service on port 9093. Real API calls with loading/error/empty states are
 * delegated to DomainHandler; this file adds the domain-specific actions.
 */
#pragma once

#include "admin_domain.hpp"

namespace tiger {
namespace admin {

class OptionsHandler : public DomainHandler {
public:
    OptionsHandler() : DomainHandler("/options") {}

    // PUT /api/v1/options/{id}/status
    void register_status(Router& router) {
        router.put("/api/v1/options/{id}/status",
            [this](const HttpRequest& r) { return set_status(r); });
    }

    void register_routes(Router& router) override {
        DomainHandler::register_routes(router);
        register_status(router);
    }
};

} // namespace admin
} // namespace tiger