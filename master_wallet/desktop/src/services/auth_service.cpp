/**
 * AuthService - C++ Implementation
 * See header for contract. Uses the existing libcurl APIClient.
 */

#include "auth_service.hpp"
#include "api_client.hpp"

namespace tiger {
namespace master {

namespace {
// Build the JSON body for auth endpoints from string fields.
std::string buildAuthBody(const std::vector<std::pair<std::string, std::string>>& fields) {
    return api::buildJsonObject(fields);
}
} // namespace

AuthService& AuthService::getInstance() {
    static AuthService instance;
    return instance;
}

AuthResult AuthService::parseAuthResponse(const std::string& resp) {
    AuthResult r;
    r.token = api::jsonStringField(resp, "token").value_or("");
    r.userId = api::jsonStringField(resp, "user_id")
                   .value_or(api::jsonStringField(resp, "userId").value_or(""));
    r.email = api::jsonStringField(resp, "email").value_or("");
    r.role = api::jsonStringField(resp, "role").value_or("");
    if (!r.token.empty() && !r.userId.empty()) {
        r.success = true;
    } else {
        r.success = false;
        r.error = "Backend auth response missing token or user_id";
    }
    return r;
}

AuthResult AuthService::login(const std::string& email, const std::string& password) {
    AuthResult fail;
    if (email.empty() || password.empty()) {
        fail.error = "Email and password are required";
        return fail;
    }

    // Auth endpoints are public: ensure no stale Bearer token is sent.
    api::backend()->clearAuthToken();

    std::string body = buildAuthBody({
        {"email", email},
        {"password", password},
    });

    std::string resp;
    try {
        resp = api::backendPost("/api/v1/auth/login", body);
    } catch (const api::APIException& e) {
        fail.error = std::string("Login failed: ") + e.what();
        return fail;
    }

    AuthResult r = parseAuthResponse(resp);
    if (r.success) {
        api::backend()->setAuthToken(r.token);
    }
    return r;
}

AuthResult AuthService::registerUser(const std::string& email, const std::string& password,
                                     const std::string& name) {
    AuthResult fail;
    if (email.empty() || password.empty()) {
        fail.error = "Email and password are required";
        return fail;
    }

    // Auth endpoints are public: ensure no stale Bearer token is sent.
    api::backend()->clearAuthToken();

    std::string body = buildAuthBody({
        {"email", email},
        {"password", password},
        {"name", name},
    });

    std::string resp;
    try {
        resp = api::backendPost("/api/v1/auth/register", body);
    } catch (const api::APIException& e) {
        fail.error = std::string("Registration failed: ") + e.what();
        return fail;
    }

    AuthResult r = parseAuthResponse(resp);
    if (r.success) {
        api::backend()->setAuthToken(r.token);
    }
    return r;
}

void AuthService::logout() {
    api::backend()->clearAuthToken();
}

std::string AuthService::currentToken() const {
    return api::backend()->authToken();
}

} // namespace master
} // namespace tiger
