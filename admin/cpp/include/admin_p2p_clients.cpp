/**
 * TigerAdmin C++ Core - P2P Clients Domain Handler
 *
 * Header-only handler for the /p2p-clients admin domain backed by the admin/go
 * service on port 9093. Real API calls with loading/error/empty states are
 * delegated to DomainHandler; this file adds the domain-specific actions.
 */
#pragma once

#include "admin_domain.hpp"

namespace tiger {
namespace admin {

class P2PClientsHandler : public DomainHandler {
public:
    P2PClientsHandler() : DomainHandler("/p2p-clients") {}

    // PUT /api/v1/p2p-clients/{id}/status
    void register_status(Router& router) {
        router.put("/api/v1/p2p-clients/{id}/status",
            [this](const HttpRequest& r) { return set_status(r); });
    }

    void register_routes(Router& router) override {
        DomainHandler::register_routes(router);
        register_status(router);
    }
};

} // namespace admin
} // namespace tiger