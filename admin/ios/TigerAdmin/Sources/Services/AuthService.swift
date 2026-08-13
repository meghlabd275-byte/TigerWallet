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
        // Real backend login against the canonical wallet_api /api/v1/auth/login
        // (real JWT, NOT a simulated mock_token). Fails-closed on any error.
        let apiBase = ProcessInfo.processInfo.environment["ADMIN_API_URL"] ?? "http://localhost:8443"
        guard let url = URL(string: apiBase + "/api/v1/auth/login") else {
            completion(.failure(NSError(domain: "AuthService", code: 0,
                userInfo: [NSLocalizedDescriptionKey: "Invalid ADMIN_API_URL"])))
            return
        }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try? JSONSerialization.data(withJSONObject: [
            "email": email, "password": password
        ])
        URLSession.shared.dataTask(with: request) { [weak self] data, response, error in
            guard let self = self,
                  error == nil,
                  let http = response as? HTTPURLResponse,
                  http.statusCode == 200,
                  let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let token = json["token"] as? String else {
                DispatchQueue.main.async {
                    completion(.failure(error ?? NSError(domain: "AuthService", code: 1,
                        userInfo: [NSLocalizedDescriptionKey: "Login failed"])))
                }
                return
            }
            self.token = token
            self.refreshToken = json["refresh_token"] as? String
            let user = User(
                id: (json["user_id"] as? String) ?? UUID().uuidString,
                email: (json["email"] as? String) ?? email,
                username: (json["username"] as? String) ?? "Admin",
                role: .admin
            )
            DispatchQueue.main.async { completion(.success(user)) }
        }.resume()
    }
    
    func logout() {
        token = nil
        refreshToken = nil
    }
}
