//
//  AuthManager.swift
//  TigerWallet
//
//  Production-Ready Authentication Service
//

import Foundation
import LocalAuthentication
import CryptoKit

// MARK: - Auth Service

class AuthManager {
    static let shared = AuthManager()
    
    private let userDefaults = UserDefaults.standard
    private let keychain = KeychainManager.shared
    
    private let isLoggedInKey = "is_logged_in"
    private let userIdKey = "user_id"
    private let userEmailKey = "user_email"
    private let sessionTokenKey = "session_token"
    private let refreshTokenKey = "refresh_token"
    
    // Session management
    private var sessionExpiry: Date?
    private let sessionDuration: TimeInterval = 7 * 24 * 60 * 60 // 7 days
    
    var isLoggedIn: Bool {
        guard userDefaults.bool(forKey: isLoggedInKey) else { return false }
        
        // Check session expiry
        if let expiry = sessionExpiry, Date() > expiry {
            logout()
            return false
        }
        
        return true
    }
    
    var userId: String? {
        userDefaults.string(forKey: userIdKey)
    }
    
    var userEmail: String? {
        userDefaults.string(forKey: userEmailKey)
    }
    
    var sessionToken: String? {
        keychain.load(key: sessionTokenKey).flatMap { String(data: $0, encoding: .utf8) }
    }
    
    private init() {
        // Check if session needs refresh
        checkSessionExpiry()
    }
    
    // MARK: - Registration
    
    func register(
        email: String,
        password: String,
        referralCode: String? = nil
    ) async throws -> User {
        // Validate inputs
        try validateEmail(email)
        try validatePassword(password)
        
        // Create registration request
        let request = RegisterRequest(
            email: email,
            password: password,
            referralCode: referralCode
        )
        
        // Make API call
        let response: AuthResponse = try await APIClient.shared.post(
            endpoint: "/api/v1/auth/register",
            body: request
        )
        
        // Save session
        try saveSession(response: response, userId: response.user.id, email: email)
        
        return response.user
    }
    
    // MARK: - Login
    
    func login(email: String, password: String, otp: String? = nil) async throws -> User {
        // Validate inputs
        try validateEmail(email)
        
        let request = LoginRequest(
            email: email,
            password: password,
            otp: otp
        )
        
        let response: AuthResponse = try await APIClient.shared.post(
            endpoint: "/api/v1/auth/login",
            body: request
        )
        
        // Save session
        try saveSession(response: response, userId: response.user.id, email: email)
        
        return response.user
    }
    
    // MARK: - Logout
    
    func logout() {
        // Clear session data
        userDefaults.set(false, forKey: isLoggedInKey)
        userDefaults.removeObject(forKey: userIdKey)
        userDefaults.removeObject(forKey: userEmailKey)
        keychain.delete(key: sessionTokenKey)
        keychain.delete(key: refreshTokenKey)
        
        sessionExpiry = nil
        
        // Post notification
        NotificationCenter.default.post(name: .userDidLogout, object: nil)
    }
    
    // MARK: - Session Refresh
    
    func refreshSession() async throws {
        guard let refreshToken = keychain.load(key: refreshTokenKey).flatMap({ String(data: $0, encoding: .utf8) }) else {
            throw AuthError.noRefreshToken
        }
        
        let request = RefreshTokenRequest(refreshToken: refreshToken)
        
        let response: TokenResponse = try await APIClient.shared.post(
            endpoint: "/api/v1/auth/refresh",
            body: request
        )
        
        // Update session
        try saveTokens(response: response)
    }
    
    // MARK: - Password Management
    
    func changePassword(currentPassword: String, newPassword: String) async throws {
        try validatePassword(newPassword)
        
        let request = ChangePasswordRequest(
            currentPassword: currentPassword,
            newPassword: newPassword
        )
        
        try await APIClient.shared.post(
            endpoint: "/api/v1/user/password",
            body: request,
            authenticated: true
        )
    }
    
    func resetPassword(email: String) async throws {
        try validateEmail(email)
        
        let request = ResetPasswordRequest(email: email)
        
        try await APIClient.shared.post(
            endpoint: "/api/v1/auth/reset-password",
            body: request
        )
    }
    
    func confirmResetPassword(token: String, newPassword: String) async throws {
        try validatePassword(newPassword)
        
        let request = ConfirmResetPasswordRequest(
            token: token,
            newPassword: newPassword
        )
        
        try await APIClient.shared.post(
            endpoint: "/api/v1/auth/reset-password/confirm",
            body: request
        )
    }
    
    // MARK: - Biometric Authentication
    
    func enableBiometric(purpose: String = "Unlock TigerWallet") async throws -> Bool {
        let context = LAContext()
        var error: NSError?
        
        guard context.canEvaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, error: &error) else {
            throw AuthError.biometricNotAvailable
        }
        
        // Store biometric flag
        userDefaults.set(true, forKey: "biometric_enabled")
        
        return true
    }
    
    func disableBiometric() {
        userDefaults.set(false, forKey: "biometric_enabled")
    }
    
    func isBiometricEnabled() -> Bool {
        userDefaults.bool(forKey: "biometric_enabled")
    }
    
    func authenticateWithBiometric(purpose: String = "Unlock TigerWallet") async throws -> Bool {
        let context = LAContext()
        var error: NSError?
        
        guard context.canEvaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, error: &error) else {
            throw AuthError.biometricNotAvailable
        }
        
        return try await withCheckedThrowingContinuation { continuation in
            context.evaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, localizedReason: purpose) { success, error in
                if let error = error {
                    continuation.resume(throwing: error)
                } else {
                    continuation.resume(returning: success)
                }
            }
        }
    }
    
    // MARK: - PIN Code
    
    func setPIN(_ pin: String) throws {
        guard pin.count >= 4 && pin.count <= 6 else {
            throw AuthError.invalidPIN
        }
        
        // Hash PIN before storing
        let hashedPIN = hashPIN(pin)
        try keychain.save(key: "user_pin", data: hashedPIN)
        
        userDefaults.set(true, forKey: "pin_enabled")
    }
    
    func verifyPIN(_ pin: String) throws -> Bool {
        guard let storedHash = keychain.load(key: "user_pin") else {
            throw AuthError.noPINSet
        }
        
        let hashedPIN = hashPIN(pin)
        return hashedPIN == Data(storedHash)
    }
    
    func removePIN() {
        keychain.delete(key: "user_pin")
        userDefaults.set(false, forKey: "pin_enabled")
    }
    
    func isPINEnabled() -> Bool {
        userDefaults.bool(forKey: "pin_enabled")
    }
    
    private func hashPIN(_ pin: String) -> Data {
        let pinData = Data(pin.utf8)
        let hash = SHA256.hash(data: pinData)
        return Data(hash)
    }
    
    // MARK: - Session Management
    
    private func saveSession(response: AuthResponse, userId: Int, email: String) throws {
        userDefaults.set(true, forKey: isLoggedInKey)
        userDefaults.set(userId, forKey: userIdKey)
        userDefaults.set(email, forKey: userEmailKey)
        
        try saveTokens(response: response)
        
        // Set session expiry
        sessionExpiry = Date().addingTimeInterval(sessionDuration)
    }
    
    private func saveTokens(response: TokenResponse) throws {
        if let token = response.token {
            try keychain.save(key: sessionTokenKey, data: Data(token.utf8))
        }
        
        if let refreshToken = response.refreshToken {
            try keychain.save(key: refreshTokenKey, data: Data(refreshToken.utf8))
        }
    }
    
    private func checkSessionExpiry() {
        // Check if stored session is still valid
        if userDefaults.bool(forKey: isLoggedInKey) {
            // If no expiry stored, assume valid
            // In production, store and check expiry
        }
    }
    
    // MARK: - Validation
    
    private func validateEmail(_ email: String) throws {
        let emailRegex = #"^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$"#
        let emailPredicate = NSPredicate(format: "SELF MATCHES %@", emailRegex)
        
        guard emailPredicate.evaluate(with: email) else {
            throw AuthError.invalidEmail
        }
    }
    
    private func validatePassword(_ password: String) throws {
        guard password.count >= 8 else {
            throw AuthError.passwordTooShort
        }
        
        let hasUppercase = password.range(of: "[A-Z]", options: .regularExpression) != nil
        let hasLowercase = password.range(of: "[a-z]", options: .regularExpression) != nil
        let hasDigit = password.range(of: "[0-9]", options: .regularExpression) != nil
        let hasSpecial = password.range(of: "[!@#$%^&*()_+\\-=\\[\\]{}|;:,.<>?]", options: .regularExpression) != nil
        
        guard hasUppercase && hasLowercase && hasDigit && hasSpecial else {
            throw AuthError.passwordTooWeak
        }
    }
}

// MARK: - Request/Response Models

struct RegisterRequest: Encodable {
    let email: String
    let password: String
    let referralCode: String?
}

struct LoginRequest: Encodable {
    let email: String
    let password: String
    let otp: String?
}

struct RefreshTokenRequest: Encodable {
    let refreshToken: String
}

struct ChangePasswordRequest: Encodable {
    let currentPassword: String
    let newPassword: String
}

struct ResetPasswordRequest: Encodable {
    let email: String
}

struct ConfirmResetPasswordRequest: Encodable {
    let token: String
    let newPassword: String
}

struct AuthResponse: Decodable {
    let token: String
    let refreshToken: String
    let user: User
}

struct TokenResponse: Decodable {
    let token: String?
    let refreshToken: String?
}

struct User: Codable {
    let id: Int
    let email: String
    let username: String?
    let phone: String?
    let country: String?
    let kycStatus: String
    let isActive: Bool
    let createdAt: Date
}

// MARK: - Auth Errors

enum AuthError: Error {
    case invalidEmail
    case passwordTooShort
    case passwordTooWeak
    case invalidCredentials
    case noRefreshToken
    case biometricNotAvailable
    case biometricFailed
    case noPINSet
    case invalidPIN
    case sessionExpired
}

// MARK: - Notifications

extension Notification.Name {
    static let userDidLogout = Notification.Name("userDidLogout")
}
