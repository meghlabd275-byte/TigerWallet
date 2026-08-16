/**
 * TigerAdmin C++ Core - Domain Route Registrar
 *
 * Wires the 12 admin domain handlers (futures, options, copy-trading, convert,
 * onramp, offramp, p2p-clients, partners, rewards, marketing, roles, and
 * p2p-merchants) into the AdminServer router. Endpoints mirror the admin/go
 * backend on port 9093 and are protected by JWT bearer auth.
 */
#pragma once

#include "admin_server.hpp"
#include "admin_logger.hpp"

// The domain handlers are header-only translation units (admin_<domain>.cpp,
// each #pragma once) included directly here per the existing cpp pattern.
#include "admin_futures.cpp"
#include "admin_options.cpp"
#include "admin_copy_trading.cpp"
#include "admin_convert.cpp"
#include "admin_onramp.cpp"
#include "admin_offramp.cpp"
#include "admin_p2p_clients.cpp"
#include "admin_partners.cpp"
#include "admin_rewards.cpp"
#include "admin_marketing.cpp"
#include "admin_roles.cpp"
#include "admin_p2p_merchants.cpp"

namespace tiger {
namespace admin {

// Registers all 12 domain handler routes on the given router. Handlers are
// owned by the registry for the lifetime of the server.
inline void register_domain_routes(Router& router) {
    static FuturesHandler      futures;
    static OptionsHandler      options;
    static CopyTradingHandler  copy_trading;
    static ConvertHandler      convert;
    static OnrampHandler       onramp;
    static OfframpHandler      offramp;
    static P2PClientsHandler   p2p_clients;
    static PartnersHandler     partners;
    static RewardsHandler      rewards;
    static MarketingHandler    marketing;
    static RolesHandler        roles;
    static P2PMerchantsHandler p2p_merchants;

    futures.register_routes(router);
    options.register_routes(router);
    copy_trading.register_routes(router);
    convert.register_routes(router);
    onramp.register_routes(router);
    offramp.register_routes(router);
    p2p_clients.register_routes(router);
    partners.register_routes(router);
    rewards.register_routes(router);
    marketing.register_routes(router);
    roles.register_routes(router);
    p2p_merchants.register_routes(router);

    LOG_INFO("Registered 12 admin domain route groups on port 9093");
}

} // namespace admin
} // namespace tiger
