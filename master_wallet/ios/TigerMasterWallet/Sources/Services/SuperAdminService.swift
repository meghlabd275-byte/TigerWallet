// MasterWallet Super Admin Service (iOS)
//
// FAIL-CLOSED: SuperAdmin is an Admin-app feature, NOT a MasterWallet feature.
// The MasterWallet Go backend (:8450) does not expose any `/api/super-admin/*`
// routes (isolation guarantee). Therefore every method in this service refuses
// to make a network call and throws instead. The file is kept for parity with
// the rest of the client surface; no method is permitted to reach the backend.

import Foundation

/// Error thrown by every `SuperAdminService` method. Carries a fixed message
/// so callers can distinguish "feature not available here" from real backend
/// errors raised by `MasterAPIService`.
struct SuperAdminUnavailableError: Error, LocalizedError {
    var errorDescription: String? {
        "SuperAdmin is an Admin-app feature, not available in MasterWallet"
    }
}

class SuperAdminService {

    private let unavailableMessage = "SuperAdmin is an Admin-app feature, not available in MasterWallet"

    // MARK: - Initialize

    func initialize() throws {
        throw SuperAdminUnavailableError()
    }

    // MARK: - Authentication

    func authenticate(email: String, password: String) async throws {
        throw SuperAdminUnavailableError()
    }

    func logout() throws {
        throw SuperAdminUnavailableError()
    }

    // MARK: - Feature Flags

    func setFeatureFlag(name: String, enabled: Bool) throws {
        throw SuperAdminUnavailableError()
    }

    func getFeatureFlag(name: String) throws -> [String: Any] {
        throw SuperAdminUnavailableError()
    }

    func listFeatureFlags() throws -> [String: [String: Any]] {
        throw SuperAdminUnavailableError()
    }

    func isFeatureEnabled(name: String) throws -> Bool {
        throw SuperAdminUnavailableError()
    }

    // MARK: - Admin Management

    func listAdmins(roleFilter: String? = nil) async throws -> [[String: Any]] {
        throw SuperAdminUnavailableError()
    }

    // MARK: - Audit Logs

    func getAuditLogs(adminId: String? = nil, action: String? = nil, limit: Int = 100) async throws -> [[String: Any]] {
        throw SuperAdminUnavailableError()
    }

    // MARK: - Accessors (fail-closed)

    func isAuthenticated() throws -> Bool {
        throw SuperAdminUnavailableError()
    }

    func getRole() throws -> String {
        throw SuperAdminUnavailableError()
    }

    func isSuperAdmin() throws -> Bool {
        throw SuperAdminUnavailableError()
    }
}
