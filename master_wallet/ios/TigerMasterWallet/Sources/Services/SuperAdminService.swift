// MasterWallet Super Admin Service (iOS)
// Admin controls for MasterWallet management
// Production-ready

import Foundation

class SuperAdminService {
    
    private let baseURL = "https://api.tigerwallet.com"
    private var adminId: String?
    private var role: String?
    private var isAuthenticated: Bool = false
    private var featureFlags: [String: [String: Any]] = [:]
    
    // MARK: - Initialize
    
    func initialize() -> Bool {
        loadSession()
        loadFeatureFlags()
        return true
    }
    
    // MARK: - Authentication
    
    func authenticate(email: String, password: String, completion: @escaping (Bool) -> Void) {
        let endpoint = "\(baseURL)/api/super-admin/authenticate"
        
        guard let url = URL(string: endpoint) else {
            completion(false)
            return
        }
        
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body = ["email": email, "password": password]
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        
        URLSession.shared.dataTask(with: request) { [weak self] data, response, error in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let adminId = json["adminId"] as? String else {
                completion(false)
                return
            }
            
            self?.adminId = adminId
            self?.role = json["role"] as? String
            self?.isAuthenticated = true
            self?.saveSession()
            
            completion(true)
        }.resume()
    }
    
    func logout() {
        adminId = nil
        role = nil
        isAuthenticated = false
        clearSession()
    }
    
    // MARK: - Feature Flags
    
    func setFeatureFlag(name: String, enabled: Bool) {
        featureFlags[name] = ["enabled": enabled, "updatedAt": Date().timeIntervalSince1970]
        saveFeatureFlags()
    }
    
    func getFeatureFlag(name: String) -> [String: Any]? {
        return featureFlags[name]
    }
    
    func listFeatureFlags() -> [String: [String: Any]] {
        return featureFlags
    }
    
    func isFeatureEnabled(name: String) -> Bool {
        return (featureFlags[name]?["enabled"] as? Bool) ?? false
    }
    
    // MARK: - Admin Management
    
    func listAdmins(roleFilter: String? = nil, completion: @escaping ([[String: Any]]) -> Void) {
        guard isAuthenticated else {
            completion([])
            return
        }
        
        var endpoint = "\(baseURL)/api/super-admin/admins"
        if let roleFilter = roleFilter {
            endpoint += "?role=\(roleFilter)"
        }
        
        guard let url = URL(string: endpoint) else {
            completion([])
            return
        }
        
        URLRequest(url: url).addValue("X-Admin-ID", forHTTPHeaderField: adminId ?? "")
        
        URLSession.shared.dataTask(with: URLRequest(url: url)) { data, _, error in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let admins = json["admins"] as? [[String: Any]] else {
                completion([])
                return
            }
            
            completion(admins)
        }.resume()
    }
    
    // MARK: - Audit Logs
    
    func getAuditLogs(adminId: String? = nil, action: String? = nil, limit: Int = 100, completion: @escaping ([[String: Any]]) -> Void) {
        guard isAuthenticated else {
            completion([])
            return
        }
        
        var endpoint = "\(baseURL)/api/super-admin/audit?limit=\(limit)"
        if let adminId = adminId {
            endpoint += "&adminId=\(adminId)"
        }
        if let action = action {
            endpoint += "&action=\(action)"
        }
        
        guard let url = URL(string: endpoint) else {
            completion([])
            return
        }
        
        URLSession.shared.dataTask(with: URLRequest(url: url)) { data, _, error in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let logs = json["logs"] as? [[String: Any]] else {
                completion([])
                return
            }
            
            completion(logs)
        }.resume()
    }
    
    // MARK: - Helpers
    
    func isAuthenticated() -> Bool {
        return isAuthenticated
    }
    
    func getRole() -> String? {
        return role
    }
    
    func isSuperAdmin() -> Bool {
        return role == "SUPER_ADMIN"
    }
    
    // MARK: - Private Methods
    
    private func loadSession() {
        adminId = UserDefaults.standard.string(forKey: "adminId")
        role = UserDefaults.standard.string(forKey: "adminRole")
        isAuthenticated = UserDefaults.standard.bool(forKey: "isAuthenticated")
    }
    
    private func saveSession() {
        UserDefaults.standard.set(adminId, forKey: "adminId")
        UserDefaults.standard.set(role, forKey: "adminRole")
        UserDefaults.standard.set(isAuthenticated, forKey: "isAuthenticated")
    }
    
    private func clearSession() {
        UserDefaults.standard.removeObject(forKey: "adminId")
        UserDefaults.standard.removeObject(forKey: "adminRole")
        UserDefaults.standard.set(false, forKey: "isAuthenticated")
    }
    
    private func loadFeatureFlags() {
        // Default feature flags
        let defaults: [String: Bool] = [
            "master_wallet_creation": true,
            "multi_blockchain": true,
            "token_management": true,
            "user_wallet_ownership": true,
            "hd_wallet": true,
            "biometric_auth": true,
            "pin_code_auth": true,
            "nft_support": true,
            "defi_integration": true,
            "staking": true,
            "bridge_support": true,
            "mev_protection": true,
            "swap_trading": true,
            "hardware_wallet": true,
            "admin_controls": true,
            "network_management": true,
            "gas_optimization": true,
            "multi_sig": true,
            "transaction_history": true,
            "price_alerts": true,
            "privacy_zk": true,
            "coinjoin": true,
            "account_abstraction": true,
            "session_keys": true,
            "paymaster": true,
            "passkeys": true,
            "tax_integration": true,
            "analytics": true,
            "cross_chain_intent": true,
            "dapp_browser": true
        ]
        
        if let data = UserDefaults.standard.data(forKey: "featureFlags"),
           let decoded = try? JSONSerialization.jsonObject(with: data) as? [String: [String: Any]] {
            featureFlags = decoded
        } else {
            featureFlags = defaults.mapValues { ["enabled": $0.value, "updatedAt": 0] }
        }
    }
    
    private func saveFeatureFlags() {
        if let encoded = try? JSONSerialization.data(withJSONObject: featureFlags) {
            UserDefaults.standard.set(encoded, forKey: "featureFlags")
        }
    }
}
