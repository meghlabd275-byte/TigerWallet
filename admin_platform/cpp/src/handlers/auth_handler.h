#pragma once

#include <string>
#include <functional>
#include "../../include/server.h"

namespace tiger::auth_handler {

void register_routes(HTTPServer& server);

HTTPResponse handle_login(const HTTPRequest& req);
HTTPResponse handle_logout(const HTTPRequest& req);
HTTPResponse handle_refresh_token(const HTTPRequest& req);
HTTPResponse handle_get_current_admin(const HTTPRequest& req);
HTTPResponse handle_change_password(const HTTPRequest& req);
HTTPResponse handle_enable_2fa(const HTTPRequest& req);
HTTPResponse handle_verify_2fa(const HTTPRequest& req);
HTTPResponse handle_disable_2fa(const HTTPRequest& req);
HTTPResponse handle_get_sessions(const HTTPRequest& req);
HTTPResponse handle_revoke_session(const HTTPRequest& req);
HTTPResponse handle_revoke_all_sessions(const HTTPRequest& req);

} // namespace tiger::auth_handler
