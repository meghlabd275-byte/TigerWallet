/**
 * AuthService - C++ Implementation
 * Authenticates against the canonical MasterWallet backend on :8450.
 * Implements POST /api/v1/auth/login and POST /api/v1/auth/register.
 * These endpoints are public (no Bearer token); on success the returned
 * JWT is stored on the shared APIClient so all subsequent protected
 * requests carry `Authorization: Bearer <JWT>`. No fabricated tokens.
 */

#ifndef AUTH_SERVICE_HPP
#define AUTH_SERVICE_HPP

#include <optional>
#include <string>

namespace tiger {
namespace master {

struct AuthResult {
    std::string token;
    std::string userId;
    std::string email;
    std::string role;
    bool success = false;
    std::string error;
};

class AuthService {
public:
    static AuthService& getInstance();

    // POST /api/v1/auth/login — {email, password} → {token, user_id, email, role}.
    // On success the JWT is applied to the shared APIClient (Bearer auth).
    AuthResult login(const std::string& email, const std::string& password);

    // POST /api/v1/auth/register — {email, password, name} → {token, user_id, email, role}.
    // On success the JWT is applied to the shared APIClient (Bearer auth).
    AuthResult registerUser(const std::string& email, const std::string& password,
                            const std::string& name);

    // Clears the JWT from the shared APIClient (client-side logout only;
    // the backend does not issue a session to revoke).
    void logout();

    // The currently cached JWT (empty if logged out / never logged in).
    std::string currentToken() const;

private:
    AuthService() = default;
    ~AuthService() = default;
    AuthService(const AuthService&) = delete;
    AuthService& operator=(const AuthService&) = delete;

    AuthResult parseAuthResponse(const std::string& resp);
};

} // namespace master
} // namespace tiger

#endif // AUTH_SERVICE_HPP
