/**
 * TigerWallet Admin - Auth Service
 */

import Foundation
import KeychainAccess

class AuthService {
    static let shared = AuthService()
    
    private let keychain = Keychain(service: "com.tigerwallet.admin")
    private let defaults = UserDefaults.standard
    
    private init() {}
    
    var isLoggedIn: Bool {
        return token != nil
    }
    
    var token: String? {
        get { try? keychain.get("auth_token") }
        set {
            if let value = newValue {
                try? keychain.set(value, key: "auth_token")
            } else {
                try? keychain.remove("auth_token")
            }
        }
    }
    
    var refreshToken: String? {
        get { try? keychain.get("refresh_token") }
        set {
            if let value = newValue {
                try? keychain.set(value, key: "refresh_token")
            } else {
                try? keychain.remove("refresh_token")
            }
        }
    }
    
    func login(email: String, password: String, completion: @escaping (Result<User, Error>) -> Void) {
        // Simulate API call
        DispatchQueue.main.asyncAfter(deadline: .now() + 1) { [weak self] in
            self?.token = "mock_token_\(UUID().uuidString)"
            self?.refreshToken = "mock_refresh_\(UUID().uuidString)"
            
            let user = User(id: UUID().uuidString, email: email, username: "Admin", role: .admin)
            completion(.success(user))
        }
    }
    
    func logout() {
        token = nil
        refreshToken = nil
    }
}
