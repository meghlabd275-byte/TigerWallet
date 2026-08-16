/**
 * TigerAdmin C++ Core - Roles Domain Handler
 *
 * Header-only handler for the /roles admin domain backed by the admin/go
 * service on port 9093. Real API calls with loading/error/empty states are
 * delegated to DomainHandler; this file adds the domain-specific actions.
 */
#pragma once

#include "admin_domain.hpp"

namespace tiger {
namespace admin {

class RolesHandler : public DomainHandler {
public:
    RolesHandler() : DomainHandler("/roles") {}

    // PUT /api/v1/roles/{id}/status
    void register_status(Router& router) {
        router.put("/api/v1/roles/{id}/status",
            [this](const HttpRequest& r) { return set_status(r); });
    }

    // RBAC. The admin/go backend mounts RBAC at the canonical paths:
    //   /api/v1/permissions            (permissions CRUD)
    //   /api/v1/admins/{id}/roles      (GET list / POST assign / DELETE {roleId} revoke)
    //   /api/v1/admins/{id}/permissions (GET effective)
    // These routes are exposed at the same canonical paths and forwarded
    // verbatim to the upstream service so native clients hit the real RBAC API.
    void register_rbac(Router& router) {
        // permissions CRUD
        router.get("/api/v1/permissions",
            [this](const HttpRequest& r) {
                LOG_INFO("GET /api/v1/permissions");
                return proxy_to_upstream(r, HttpMethod::GET, "/api/v1/permissions");
            });
        router.post("/api/v1/permissions",
            [this](const HttpRequest& r) {
                LOG_INFO("POST /api/v1/permissions");
                return proxy_to_upstream(r, HttpMethod::POST, "/api/v1/permissions");
            });
        router.put("/api/v1/permissions/{pid}",
            [this](const HttpRequest& r) {
                LOG_INFO("PUT /api/v1/permissions/" + path_param(r, "pid"));
                return proxy_to_upstream(r, HttpMethod::PUT,
                    "/api/v1/permissions/" + path_param(r, "pid"));
            });
        router.delete_route("/api/v1/permissions/{pid}",
            [this](const HttpRequest& r) {
                LOG_INFO("DELETE /api/v1/permissions/" + path_param(r, "pid"));
                return proxy_to_upstream(r, HttpMethod::DELETE,
                    "/api/v1/permissions/" + path_param(r, "pid"));
            });
        // list an admin's roles
        router.get("/api/v1/admins/{aid}/roles",
            [this](const HttpRequest& r) {
                LOG_INFO("GET /api/v1/admins/" + path_param(r, "aid") + "/roles");
                return proxy_to_upstream(r, HttpMethod::GET,
                    "/api/v1/admins/" + path_param(r, "aid") + "/roles");
            });
        // assign a role to an admin
        router.post("/api/v1/admins/{aid}/roles",
            [this](const HttpRequest& r) {
                LOG_INFO("POST /api/v1/admins/" + path_param(r, "aid") + "/roles");
                return proxy_to_upstream(r, HttpMethod::POST,
                    "/api/v1/admins/" + path_param(r, "aid") + "/roles");
            });
        // revoke a role from an admin
        router.delete_route("/api/v1/admins/{aid}/roles/{rid}",
            [this](const HttpRequest& r) {
                LOG_INFO("DELETE /api/v1/admins/" + path_param(r, "aid") +
                         "/roles/" + path_param(r, "rid"));
                return proxy_to_upstream(r, HttpMethod::DELETE,
                    "/api/v1/admins/" + path_param(r, "aid") +
                    "/roles/" + path_param(r, "rid"));
            });
        // list an admin's effective permissions
        router.get("/api/v1/admins/{aid}/permissions",
            [this](const HttpRequest& r) {
                LOG_INFO("GET /api/v1/admins/" + path_param(r, "aid") + "/permissions");
                return proxy_to_upstream(r, HttpMethod::GET,
                    "/api/v1/admins/" + path_param(r, "aid") + "/permissions");
            });
    }

    void register_routes(Router& router) override {
        DomainHandler::register_routes(router);
        register_status(router);
        register_rbac(router);
    }
};

} // namespace admin
} // namespace tiger