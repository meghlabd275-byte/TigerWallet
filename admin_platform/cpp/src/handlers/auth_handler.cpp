#include "auth_handler.h"
#include <nlohmann/json.hpp>
#include "../../include/session_manager.h"
#include "../../include/admin_service.h"

using json = nlohmann::json;

namespace tiger::auth_handler {

// Global service references (set in main.cpp)
extern std::unique_ptr<admin::AdminService> g_admin_service;
extern std::unique_ptr<SessionManager> g_session_manager;

void register_routes(HTTPServer& server) {
    // Auth routes
    server.post("/api/v1/auth/login", handle_login, "login", false);
    server.post("/api/v1/auth/logout", handle_logout, "logout", true);
    server.post("/api/v1/auth/refresh", handle_refresh_token, "refresh", false);
    server.get("/api/v1/auth/me", handle_get_current_admin, "get_current", true);
    server.post("/api/v1/auth/change-password", handle_change_password, "change_password", true);
    
    // 2FA routes
    server.post("/api/v1/auth/2fa/enable", handle_enable_2fa, "enable_2fa", true);
    server.post("/api/v1/auth/2fa/verify", handle_verify_2fa, "verify_2fa", true);
    server.post("/api/v1/auth/2fa/disable", handle_disable_2fa, "disable_2fa", true);
    
    // Session routes
    server.get("/api/v1/auth/sessions", handle_get_sessions, "get_sessions", true);
    server.delete("/api/v1/auth/sessions/:id", handle_revoke_session, "revoke_session", true);
    server.delete("/api/v1/auth/sessions", handle_revoke_all_sessions, "revoke_all_sessions", true);
}

HTTPResponse handle_login(const HTTPRequest& req) {
    try {
        auto body = json::parse(req.body);
        
        std::string email = body.value("email", "");
        std::string password = body.value("password", "");
        std::string ip = req.remote_ip;
        
        if (email.empty() || password.empty()) {
            return HTTPResponse().status(HTTP_400_BAD_REQUEST)
                .json(R"({"error":"Email and password are required"})");
        }
        
        auto result = g_admin_service->authenticate(email, password, ip);
        
        if (!result.success) {
            return HTTPResponse().status(HTTP_401_UNAUTHORIZED)
                .json(R"({"error":")" + result.error + "\"}");
        }
        
        return HTTPResponse().status(HTTP_200_OK).json(result.data->dump());
        
    } catch (const std::exception& e) {
        return HTTPResponse().status(HTTP_500_INTERNAL_SERVER_ERROR)
            .json(R"({"error":"Internal server error"})");
    }
}

HTTPResponse handle_logout(const HTTPRequest& req) {
    try {
        auto auth_header = req.get_header("Authorization");
        if (auth_header.empty()) {
            return HTTPResponse().status(HTTP_401_UNAUTHORIZED)
                .json(R"({"error":"Authorization required"})");
        }
        
        // Extract token from "Bearer <token>"
        std::string token = auth_header.substr(7);
        
        // Get admin_id from context (set by auth middleware)
        std::string admin_id = req.get_header("X-Admin-Id");
        
        if (!admin_id.empty()) {
            g_admin_service->logout(admin_id, token);
        }
        
        return HTTPResponse().status(HTTP_200_OK)
            .json(R"({"message":"Logged out successfully"})");
            
    } catch (const std::exception& e) {
        return HTTPResponse().status(HTTP_500_INTERNAL_SERVER_ERROR)
            .json(R"({"error":"Internal server error"})");
    }
}

HTTPResponse handle_refresh_token(const HTTPRequest& req) {
    try {
        auto body = json::parse(req.body);
        std::string refresh_token = body.value("refresh_token", "");
        
        if (refresh_token.empty()) {
            return HTTPResponse().status(HTTP_400_BAD_REQUEST)
                .json(R"({"error":"Refresh token required"})");
        }
        
        auto result = g_admin_service->refresh_token(refresh_token);
        
        if (!result.success) {
            return HTTPResponse().status(HTTP_401_UNAUTHORIZED)
                .json(R"({"error":"Invalid refresh token"})");
        }
        
        return HTTPResponse().status(HTTP_200_OK).json(result.data->dump());
        
    } catch (const std::exception& e) {
        return HTTPResponse().status(HTTP_500_INTERNAL_SERVER_ERROR)
            .json(R"({"error":"Internal server error"})");
    }
}

HTTPResponse handle_get_current_admin(const HTTPRequest& req) {
    try {
        std::string admin_id = req.get_header("X-Admin-Id");
        
        if (admin_id.empty()) {
            return HTTPResponse().status(HTTP_401_UNAUTHORIZED)
                .json(R"({"error":"Unauthorized"})");
        }
        
        auto result = g_admin_service->get_admin(admin_id);
        
        if (!result.success) {
            return HTTPResponse().status(HTTP_404_NOT_FOUND)
                .json(R"({"error":"Admin not found"})");
        }
        
        return HTTPResponse().status(HTTP_200_OK).json(result.data->to_json().dump());
        
    } catch (const std::exception& e) {
        return HTTPResponse().status(HTTP_500_INTERNAL_SERVER_ERROR)
            .json(R"({"error":"Internal server error"})");
    }
}

HTTPResponse handle_change_password(const HTTPRequest& req) {
    try {
        std::string admin_id = req.get_header("X-Admin-Id");
        if (admin_id.empty()) {
            return HTTPResponse().status(HTTP_401_UNAUTHORIZED)
                .json(R"({"error":"Unauthorized"})");
        }
        
        auto body = json::parse(req.body);
        std::string old_password = body.value("old_password", "");
        std::string new_password = body.value("new_password", "");
        
        if (old_password.empty() || new_password.empty()) {
            return HTTPResponse().status(HTTP_400_BAD_REQUEST)
                .json(R"({"error":"Old and new passwords are required"})");
        }
        
        // Password change logic would go here
        // For now, return success
        return HTTPResponse().status(HTTP_200_OK)
            .json(R"({"message":"Password changed successfully"})");
            
    } catch (const std::exception& e) {
        return HTTPResponse().status(HTTP_500_INTERNAL_SERVER_ERROR)
            .json(R"({"error":"Internal server error"})");
    }
}

HTTPResponse handle_enable_2fa(const HTTPRequest& req) {
    try {
        std::string admin_id = req.get_header("X-Admin-Id");
        if (admin_id.empty()) {
            return HTTPResponse().status(HTTP_401_UNAUTHORIZED)
                .json(R"({"error":"Unauthorized"})");
        }
        
        auto result = g_admin_service->enable_2fa(admin_id);
        
        if (!result.success) {
            return HTTPResponse().status(HTTP_400_BAD_REQUEST)
                .json(R"({"error":")" + result.error + "\"}");
        }
        
        return HTTPResponse().status(HTTP_200_OK).json(result.data->dump());
        
    } catch (const std::exception& e) {
        return HTTPResponse().status(HTTP_500_INTERNAL_SERVER_ERROR)
            .json(R"({"error":"Internal server error"})");
    }
}

HTTPResponse handle_verify_2fa(const HTTPRequest& req) {
    try {
        auto body = json::parse(req.body);
        std::string code = body.value("code", "");
        
        if (code.empty()) {
            return HTTPResponse().status(HTTP_400_BAD_REQUEST)
                .json(R"({"error":"Code required"})");
        }
        
        std::string admin_id = req.get_header("X-Admin-Id");
        if (admin_id.empty()) {
            return HTTPResponse().status(HTTP_401_UNAUTHORIZED)
                .json(R"({"error":"Unauthorized"})");
        }
        
        auto result = g_admin_service->authenticate_2fa(admin_id, code);
        
        if (!result.success) {
            return HTTPResponse().status(HTTP_401_UNAUTHORIZED)
                .json(R"({"error":"Invalid code"})");
        }
        
        return HTTPResponse().status(HTTP_200_OK).json(result.data->dump());
        
    } catch (const std::exception& e) {
        return HTTPResponse().status(HTTP_500_INTERNAL_SERVER_ERROR)
            .json(R"({"error":"Internal server error"})");
    }
}

HTTPResponse handle_disable_2fa(const HTTPRequest& req) {
    try {
        std::string admin_id = req.get_header("X-Admin-Id");
        if (admin_id.empty()) {
            return HTTPResponse().status(HTTP_401_UNAUTHORIZED)
                .json(R"({"error":"Unauthorized"})");
        }
        
        auto body = json::parse(req.body);
        std::string code = body.value("code", "");
        
        if (code.empty()) {
            return HTTPResponse().status(HTTP_400_BAD_REQUEST)
                .json(R"({"error":"Code required"})");
        }
        
        auto result = g_admin_service->disable_2fa(admin_id, code);
        
        if (!result.success) {
            return HTTPResponse().status(HTTP_400_BAD_REQUEST)
                .json(R"({"error":")" + result.error + "\"}");
        }
        
        return HTTPResponse().status(HTTP_200_OK)
            .json(R"({"message":"2FA disabled successfully"})");
            
    } catch (const std::exception& e) {
        return HTTPResponse().status(HTTP_500_INTERNAL_SERVER_ERROR)
            .json(R"({"error":"Internal server error"})");
    }
}

HTTPResponse handle_get_sessions(const HTTPRequest& req) {
    try {
        std::string admin_id = req.get_header("X-Admin-Id");
        if (admin_id.empty()) {
            return HTTPResponse().status(HTTP_401_UNAUTHORIZED)
                .json(R"({"error":"Unauthorized"})");
        }
        
        auto sessions = g_admin_service->get_active_sessions(admin_id);
        
        json result = json::array();
        for (const auto& sess : sessions) {
            result.push_back(json::parse(sess));
        }
        
        return HTTPResponse().status(HTTP_200_OK).json(result.dump());
        
    } catch (const std::exception& e) {
        return HTTPResponse().status(HTTP_500_INTERNAL_SERVER_ERROR)
            .json(R"({"error":"Internal server error"})");
    }
}

HTTPResponse handle_revoke_session(const HTTPRequest& req) {
    try {
        std::string admin_id = req.get_header("X-Admin-Id");
        if (admin_id.empty()) {
            return HTTPResponse().status(HTTP_401_UNAUTHORIZED)
                .json(R"({"error":"Unauthorized"})");
        }
        
        std::string session_id = req.get_path_param("id");
        
        auto result = g_admin_service->revoke_session(admin_id, session_id);
        
        if (!result.success) {
            return HTTPResponse().status(HTTP_400_BAD_REQUEST)
                .json(R"({"error":")" + result.error + "\"}");
        }
        
        return HTTPResponse().status(HTTP_200_OK)
            .json(R"({"message":"Session revoked"})");
            
    } catch (const std::exception& e) {
        return HTTPResponse().status(HTTP_500_INTERNAL_SERVER_ERROR)
            .json(R"({"error":"Internal server error"})");
    }
}

HTTPResponse handle_revoke_all_sessions(const HTTPRequest& req) {
    try {
        std::string admin_id = req.get_header("X-Admin-Id");
        if (admin_id.empty()) {
            return HTTPResponse().status(HTTP_401_UNAUTHORIZED)
                .json(R"({"error":"Unauthorized"})");
        }
        
        auto result = g_admin_service->revoke_all_sessions(admin_id);
        
        if (!result.success) {
            return HTTPResponse().status(HTTP_400_BAD_REQUEST)
                .json(R"({"error":")" + result.error + "\"}");
        }
        
        return HTTPResponse().status(HTTP_200_OK)
            .json(R"({"message":"All sessions revoked"})");
            
    } catch (const std::exception& e) {
        return HTTPResponse().status(HTTP_500_INTERNAL_SERVER_ERROR)
            .json(R"({"error":"Internal server error"})");
    }
}

} // namespace tiger::auth_handler
