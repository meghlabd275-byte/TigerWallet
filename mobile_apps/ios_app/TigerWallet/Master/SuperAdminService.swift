//
//  SuperAdminService.swift
//  TigerWallet
//
//  COMPLETE SUPER ADMIN SYSTEM
//  - Only Super Admin can authorize Master Wallet Admin accounts
//  - Master Wallet Admin login requires Super Admin authorization
//  - Master Wallet Admin can change password and set 2FA after login
//  - Super Admin has FULL control over all features
//  - White Label Admin has full control in their custom branding
//

import Foundation
import CommonCrypto

// MARK: - Enums

enum UserRole: String { case SUPER_ADMIN, MASTER_ADMIN, WHITE_LABEL_ADMIN, USER }
enum AdminStatus: String { case ACTIVE, INACTIVE, PENDING, SUSPENDED }
enum AuthorizationStatus: String { case AUTHORIZED, PENDING, REVOKED, REJECTED }

// MARK: - Data Models

struct SuperAdmin {
    let id: String
    let email: String
    let passwordHash: String
    let secretKey: String
    var twoFactorEnabled: Bool
    var twoFactorSecret: String
    let phone: String
    let createdAt: TimeInterval
    var lastLogin: TimeInterval
    let isActive: Bool
    let permissions: [String]
}

struct MasterAdmin {
    let id: String
    let email: String
    var passwordHash: String
    var authorizedBy: String
    var authorizationStatus: AuthorizationStatus
    var twoFactorEnabled: Bool
    var twoFactorSecret: String
    let phone: String
    var canCreateWhiteLabel: Bool
    var canManageUsers: Bool
    var canManageWallets: Bool
    var canAccessFinance: Bool
    var canModifyFeatures: Bool
    var canManageTokens: Bool
    var canManageNetworks: Bool
    var canViewAnalytics: Bool
    var canManageAdmins: Bool
    var maxWhiteLabels: Int
    var whiteLabelCount: Int
    var status: AdminStatus
    let createdAt: TimeInterval
    var lastLogin: TimeInterval
    var passwordChangedAt: TimeInterval
    var failedAttempts: Int
    var lockedUntil: TimeInterval
}

struct WhiteLabelAdmin {
    let id: String
    let email: String
    var passwordHash: String
    let masterAdminId: String
    var brandName: String
    var brandLogo: String
    var brandColor: String
    var customDomain: String
    var authorizationStatus: AuthorizationStatus
    var twoFactorEnabled: Bool
    var twoFactorSecret: String
    var canCustomizeUi: Bool
    var canCustomizeFees: Bool
    var canManageUsers: Bool
    var canManageWallets: Bool
    var canAccessAnalytics: Bool
    var canManageTokens: Bool
    var feePercentage: Double
    var status: AdminStatus
    let createdAt: TimeInterval
    var lastLogin: TimeInterval
}

struct FeatureControl {
    var featureName: String
    var enabled: Bool
    var globalEnabled: Bool
    var masterAdminId: String
    var whiteLabelId: String
    var updatedBy: String
    var updatedAt: TimeInterval
}

struct AuditLog {
    let id: String
    let adminId: String
    let adminRole: UserRole
    let action: String
    let details: String
    let ipAddress: String
    let userAgent: String
    let timestamp: TimeInterval
}

// MARK: - Super Admin Service

class SuperAdminService {
    static let shared = SuperAdminService()
    
    private var superAdmins: [String: Any] = [:]
    private var masterAdmins: [String: Any] = [:]
    private var whiteLabelAdmins: [String: Any] = [:]
    private var featureControls: [String: FeatureControl] = [:]
    private var auditLogs: [AuditLog] = []
    
    private init() {
        createDefaultSuperAdmin()
        initializeFeatureControls()
    }
    
    // MARK: - Default Super Admin
    
    private func createDefaultSuperAdmin() {
        let superAdmin = SuperAdmin(
            id: "super_admin_001",
            email: "superadmin@tigerwallet.com",
            passwordHash: sha256("SuperAdmin@2024!"),
            secretKey: generateSecretKey(),
            twoFactorEnabled: false,
            twoFactorSecret: "",
            phone: "",
            createdAt: Date().timeIntervalSince1970,
            lastLogin: 0,
            isActive: true,
            permissions: ["*"]
        )
        superAdmins[superAdmin.id] = superAdmin
        superAdmins[superAdmin.email] = superAdmin
    }
    
    private func initializeFeatureControls() {
        let features = ["master_wallet_creation", "multi_blockchain", "token_management", "user_wallet_ownership", "hd_wallet", "biometric_auth", "pin_code_auth", "nft_support", "defi_integration", "staking", "bridge_support", "mev_protection", "swap_trading", "hardware_wallet", "admin_controls", "network_management", "gas_optimization", "multi_sig", "transaction_history", "price_alerts", "privacy_zk", "coinjoin", "account_abstraction", "session_keys", "paymaster", "passkeys", "tax_integration", "analytics", "cross_chain_intent", "dapp_browser"]
        
        for feature in features {
            featureControls[feature] = FeatureControl(
                featureName: feature,
                enabled: true,
                globalEnabled: true,
                masterAdminId: "",
                whiteLabelId: "",
                updatedBy: "",
                updatedAt: Date().timeIntervalSince1970
            )
        }
    }
    
    // MARK: - Super Admin Login
    
    func superAdminLogin(email: String, password: String, twoFactorCode: String = "") -> SuperAdmin? {
        guard let superAdmin = superAdmins[email] as? SuperAdmin else { return nil }
        guard superAdmin.isActive else { return nil }
        
        if sha256(password) != superAdmin.passwordHash {
            logAudit(superAdmin.id, role: .SUPER_ADMIN, action: "LOGIN_FAILED", details: "Invalid password")
            return nil
        }
        
        if superAdmin.twoFactorEnabled && !verifyTwoFactor(superAdmin.twoFactorSecret, code: twoFactorCode) {
            return nil
        }
        
        logAudit(superAdmin.id, role: .SUPER_ADMIN, action: "LOGIN_SUCCESS", details: "Super admin logged in")
        return superAdmin
    }
    
    // MARK: - Master Admin Operations
    
    func createMasterAdminRequest(email: String, requestedBy: String) -> MasterAdmin {
        let masterAdmin = MasterAdmin(
            id: generateId(),
            email: email,
            passwordHash: sha256(generateTempPassword()),
            authorizedBy: "",
            authorizationStatus: .PENDING,
            twoFactorEnabled: false,
            twoFactorSecret: "",
            phone: "",
            canCreateWhiteLabel: false,
            canManageUsers: false,
            canManageWallets: false,
            canAccessFinance: false,
            canModifyFeatures: false,
            canManageTokens: false,
            canManageNetworks: false,
            canViewAnalytics: false,
            canManageAdmins: false,
            maxWhiteLabels: 0,
            whiteLabelCount: 0,
            status: .PENDING,
            createdAt: Date().timeIntervalSince1970,
            lastLogin: 0,
            passwordChangedAt: 0,
            failedAttempts: 0,
            lockedUntil: 0
        )
        
        masterAdmins[masterAdmin.id] = masterAdmin
        masterAdmins[email] = masterAdmin
        logAudit("SYSTEM", role: .SUPER_ADMIN, action: "MASTER_ADMIN_REQUEST", details: "New request: \(email)")
        
        return masterAdmin
    }
    
    func authorizeMasterAdmin(superAdminId: String, masterAdminId: String, authorized: Bool, notes: String = "") throws {
        guard superAdmins[superAdminId] != nil else {
            throw NSError(domain: "SuperAdmin", code: 1, userInfo: [NSLocalizedDescriptionKey: "Only super admin can authorize"])
        }
        
        guard var masterAdmin = masterAdmins[masterAdminId] as? MasterAdmin else { return }
        
        masterAdmin.authorizationStatus = authorized ? .AUTHORIZED : .REJECTED
        masterAdmin.status = authorized ? .ACTIVE : .INACTIVE
        masterAdmin.authorizedBy = superAdminId
        
        masterAdmins[masterAdminId] = masterAdmin
        masterAdmins[masterAdmin.email] = masterAdmin
        
        let action = authorized ? "AUTHORIZED" : "REJECTED"
        logAudit(superAdminId, role: .SUPER_ADMIN, action: "MASTER_ADMIN_\(action)", details: "\(action) \(masterAdmin.email)")
    }
    
    func masterAdminLogin(email: String, password: String, twoFactorCode: String = "") -> MasterAdmin? {
        guard var masterAdmin = masterAdmins[email] as? MasterAdmin else { return nil }
        
        guard masterAdmin.authorizationStatus == .AUTHORIZED else { return nil }
        guard masterAdmin.status == .ACTIVE else { return nil }
        
        if masterAdmin.lockedUntil > Date().timeIntervalSince1970 { return nil }
        
        if sha256(password) != masterAdmin.passwordHash {
            masterAdmin.failedAttempts += 1
            if masterAdmin.failedAttempts >= 5 {
                masterAdmin.lockedUntil = Date().addingTimeInterval(900).timeIntervalSince1970
                masterAdmin.status = .SUSPENDED
            }
            masterAdmins[masterAdmin.id] = masterAdmin
            logAudit(masterAdmin.id, role: .MASTER_ADMIN, action: "LOGIN_FAILED", details: "Invalid password")
            return nil
        }
        
        if masterAdmin.twoFactorEnabled && !verifyTwoFactor(masterAdmin.twoFactorSecret, code: twoFactorCode) {
            return nil
        }
        
        masterAdmin.lastLogin = Date().timeIntervalSince1970
        masterAdmin.failedAttempts = 0
        masterAdmins[masterAdmin.id] = masterAdmin
        masterAdmins[masterAdmin.email] = masterAdmin
        
        logAudit(masterAdmin.id, role: .MASTER_ADMIN, action: "LOGIN_SUCCESS", details: "Master admin logged in")
        
        return masterAdmin
    }
    
    func changeMasterAdminPassword(adminId: String, oldPassword: String, newPassword: String) -> Bool {
        guard var masterAdmin = (masterAdmins[adminId] as? MasterAdmin) ?? (masterAdmins.values.first { ($0 as? MasterAdmin)?.email == adminId } as? MasterAdmin) else {
            return false
        }
        
        guard sha256(oldPassword) == masterAdmin.passwordHash else { return false }
        guard newPassword.count >= 8 else { return false }
        
        masterAdmin.passwordHash = sha256(newPassword)
        masterAdmin.passwordChangedAt = Date().timeIntervalSince1970
        
        masterAdmins[masterAdmin.id] = masterAdmin
        masterAdmins[masterAdmin.email] = masterAdmin
        
        logAudit(adminId, role: .MASTER_ADMIN, action: "PASSWORD_CHANGED", details: "Password changed")
        return true
    }
    
    func enableMasterAdmin2FA(adminId: String, secret: String) -> Bool {
        guard var masterAdmin = masterAdmins[adminId] as? MasterAdmin else { return false }
        
        masterAdmin.twoFactorEnabled = true
        masterAdmin.twoFactorSecret = secret
        
        masterAdmins[masterAdmin.id] = masterAdmin
        masterAdmins[masterAdmin.email] = masterAdmin
        
        logAudit(adminId, role: .MASTER_ADMIN, action: "2FA_ENABLED", details: "2FA enabled")
        return true
    }
    
    // MARK: - White Label Admin
    
    func createWhiteLabelAdmin(masterAdminId: String, email: String, brandName: String) -> WhiteLabelAdmin? {
        guard let masterAdmin = masterAdmins[masterAdminId] as? MasterAdmin else { return nil }
        guard masterAdmin.canCreateWhiteLabel else { return nil }
        guard masterAdmin.whiteLabelCount < masterAdmin.maxWhiteLabels else { return nil }
        
        let whiteLabel = WhiteLabelAdmin(
            id: generateId(),
            email: email,
            passwordHash: sha256(generateTempPassword()),
            masterAdminId: masterAdminId,
            brandName: brandName,
            brandLogo: "",
            brandColor: "#000000",
            customDomain: "",
            authorizationStatus: .AUTHORIZED,
            twoFactorEnabled: false,
            twoFactorSecret: "",
            canCustomizeUi: true,
            canCustomizeFees: true,
            canManageUsers: true,
            canManageWallets: true,
            canAccessAnalytics: true,
            canManageTokens: true,
            feePercentage: 0.0,
            status: .ACTIVE,
            createdAt: Date().timeIntervalSince1970,
            lastLogin: 0
        )
        
        whiteLabelAdmins[whiteLabel.id] = whiteLabel
        whiteLabelAdmins[email] = whiteLabel
        
        logAudit(masterAdminId, role: .MASTER_ADMIN, action: "WHITE_LABEL_CREATED", details: "Created: \(email) - \(brandName)")
        
        return whiteLabel
    }
    
    // MARK: - Feature Control
    
    func setGlobalFeature(superAdminId: String, featureName: String, enabled: Bool) throws {
        guard superAdmins[superAdminId] != nil else {
            throw NSError(domain: "SuperAdmin", code: 1, userInfo: [NSLocalizedDescriptionKey: "Only super admin can modify features"])
        }
        
        guard var feature = featureControls[featureName] else { return }
        
        feature.enabled = enabled
        feature.globalEnabled = enabled
        feature.updatedBy = superAdminId
        feature.updatedAt = Date().timeIntervalSince1970
        
        featureControls[featureName] = feature
        logAudit(superAdminId, role: .SUPER_ADMIN, action: "FEATURE_TOGGLE", details: "Set \(featureName) = \(enabled)")
    }
    
    func getAllFeatures() -> [FeatureControl] {
        return Array(featureControls.values)
    }
    
    func isFeatureEnabled(featureName: String, adminId: String, role: UserRole) -> Bool {
        guard let feature = featureControls[featureName] else { return false }
        guard feature.globalEnabled else { return false }
        
        switch role {
        case .SUPER_ADMIN: return true
        case .MASTER_ADMIN:
            if !feature.masterAdminId.isEmpty && feature.masterAdminId != adminId { return false }
            return feature.enabled
        case .WHITE_LABEL_ADMIN:
            if !feature.whiteLabelId.isEmpty && feature.whiteLabelId != adminId { return false }
            return feature.enabled
        default: return false
        }
    }
    
    // MARK: - Audit
    
    private func logAudit(adminId: String, role: UserRole, action: String, details: String, ipAddress: String = "", userAgent: String = "") {
        let log = AuditLog(
            id: generateId(),
            adminId: adminId,
            adminRole: role,
            action: action,
            details: details,
            ipAddress: ipAddress,
            userAgent: userAgent,
            timestamp: Date().timeIntervalSince1970
        )
        auditLogs.append(log)
        print("[AUDIT] \(role.rawValue) | \(action) | \(details)")
    }
    
    func getAuditLogs(adminId: String = "", limit: Int = 100) -> [AuditLog] {
        if adminId.isEmpty { return Array(auditLogs.suffix(limit)) }
        return auditLogs.filter { $0.adminId == adminId }.suffix(limit).map { $0 }
    }
    
    // MARK: - Helpers
    
    private func generateId() -> String { return "id_\(Int(Date().timeIntervalSince1970))_\(Int.random(in: 0...999999))" }
    private func generateSecretKey() -> String { return (0..<32).map { String(format: "%02x", Int.random(in: 0...255)) }.joined() }
    private func generateTempPassword() -> String { return (0..<16).map { "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"[Int.random(in: 0...61)] }.joined() }
    
    private func sha256(_ input: String) -> String {
        let data = Data(input.utf8)
        var hash = [UInt8](repeating: 0, count: Int(CC_SHA256_DIGEST_LENGTH))
        data.withUnsafeBytes { _ = CC_SHA256($0.baseAddress, CC_LONG(data.count), &hash) }
        return hash.map { String(format: "%02x", $0) }.joined()
    }
    
    /// Real TOTP (RFC 6238) verification using HMAC-SHA1 over the shared
    /// secret and the current 30s time step. Rejects any code that does not
    /// match the HOTP value (±1 step window for clock skew). Never accepts an
    /// arbitrary 6-digit code.
    private func verifyTwoFactor(_ secret: String, code: String) -> Bool {
        guard let key = base32Decode(secret) else { return false }
        let steps = Int(Date().timeIntervalSince1970 / 30)
        // ±1 time-step window to tolerate minor clock skew.
        for skew in -1...1 {
            let counter = UInt64(steps + skew).bigEndian
            let counterData = withUnsafeBytes(of: counter) { Data($0) }
            var hmac = [UInt8](repeating: 0, count: Int(CC_SHA1_DIGEST_LENGTH))
            key.withUnsafeBytes { keyPtr in
                counterData.withUnsafeBytes { ctrPtr in
                    CCHmac(
                        CCHmacAlgorithm(kCCHmacAlgSHA1),
                        keyPtr.baseAddress, key.count,
                        ctrPtr.baseAddress, counterData.count,
                        &hmac
                    )
                }
            }
            let o = Int(hmac[hmac.count - 1] & 0x0f)
            let binary: UInt32 =
                (UInt32(hmac[o] & 0x7f) << 24) |
                (UInt32(hmac[o + 1]) << 16) |
                (UInt32(hmac[o + 2]) << 8) |
                UInt32(hmac[o + 3])
            let otp = String(format: "%06d", binary % 1_000_000)
            if otp == code { return true }
        }
        return false
    }

    /// RFC 4648 Base32 decode (uppercase, no padding required).
    private func base32Decode(_ s: String) -> Data? {
        let alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
        var bits = 0
        var value = 0
        var out = Data()
        for ch in s.uppercased() where ch != "=" {
            guard let idx = alphabet.firstIndex(of: ch) else { return nil }
            value = (value << 5) | alphabet.distance(from: alphabet.startIndex, to: idx)
            bits += 5
            if bits >= 8 {
                bits -= 8
                out.append(UInt8((value >> bits) & 0xff))
            }
        }
        return out.isEmpty ? nil : out
    }
}
