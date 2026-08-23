import Foundation

/**
 * Hardware Wallet Service - iOS Native Implementation
 */
class HardwareWalletService {
    static let API_BASE = "http://localhost:8443/api/v1"
    static let SUPPORTED_DEVICES = ["LEDGER_NANO_X", "LEDGER_NANO_S", "TREZOR_MODEL_T", "TREZOR_ONE", "KEYSTONE", "COLDCAED"]
    var authToken: String?
    
    init(token: String? = nil) { self.authToken = token }
    
    private func createRequest(endpoint: String, method: String = "GET", body: [String: Any]? = nil) -> URLRequest? {
        guard let url = URL(string: "\(Self.API_BASE)\(endpoint)") else { return nil }
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let token = authToken { request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization") }
        if let body = body { request.httpBody = try? JSONSerialization.data(withJSONObject: body) }
        return request
    }
    
    /// Register device
    func registerDevice(deviceType: String, serialNumber: String, firmwareVersion: String, completion: @escaping (Result<HardwareWallet, Error>) -> Void) {
        guard let request = createRequest(endpoint: "/hardware/register", method: "POST", body: [
            "deviceType": deviceType, "serialNumber": serialNumber, "firmwareVersion": firmwareVersion
        ]) else { completion(.failure(HWError.invalidRequest)); return }
        
        URLSession.shared.dataTask(with: request) { data, _, error in
            if let error = error { completion(.failure(error)); return }
            guard let data = data, let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let walletData = json["data"] as? [String: Any] else { completion(.failure(HWError.invalidResponse)); return }
            let wallet = HardwareWallet(id: walletData["id"] as? String ?? "", deviceType: walletData["deviceType"] as? String ?? "",
                                       serialNumber: walletData["serialNumber"] as? String ?? "", firmwareVersion: walletData["firmwareVersion"] as? String ?? "",
                                       status: walletData["status"] as? String ?? "ACTIVE")
            completion(.success(wallet))
        }.resume()
    }
    
    /// Sign transaction
    func signTransaction(walletId: String, txHash: String, completion: @escaping (Result<String, Error>) -> Void) {
        guard let request = createRequest(endpoint: "/hardware/sign", method: "POST", body: ["walletId": walletId, "txHash": txHash]) else {
            completion(.failure(HWError.invalidRequest)); return
        }
        
        URLSession.shared.dataTask(with: request) { data, _, error in
            if let error = error { completion(.failure(error)); return }
            guard let data = data, let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let signature = json["data"] as? [String: Any], let sig = signature["signature"] as? String else {
                completion(.failure(HWError.invalidResponse)); return
            }
            completion(.success(sig))
        }.resume()
    }
    
    /// Get wallets
    func getWallets(completion: @escaping (Result<[HardwareWallet], Error>) -> Void) {
        guard let request = createRequest(endpoint: "/hardware/wallets") else { completion(.failure(HWError.invalidRequest)); return }
        
        URLSession.shared.dataTask(with: request) { data, _, error in
            if let error = error { completion(.failure(error)); return }
            guard let data = data, let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let walletsData = json["data"] as? [[String: Any]] else { completion(.failure(HWError.invalidResponse)); return }
            let wallets = walletsData.compactMap { w -> HardwareWallet? in
                guard let id = w["id"] as? String else { return nil }
                return HardwareWallet(id: id, deviceType: w["deviceType"] as? String ?? "", serialNumber: w["serialNumber"] as? String ?? "",
                                     firmwareVersion: w["firmwareVersion"] as? String ?? "", status: w["status"] as? String ?? "ACTIVE")
            }
            completion(.success(wallets))
        }.resume()
    }
}

struct HardwareWallet {
    let id: String
    let deviceType: String
    let serialNumber: String
    let firmwareVersion: String
    let status: String
}

enum HWError: Error { case invalidRequest, invalidResponse }

/**
 * MPC Wallet Service - iOS Native Implementation
 */
class MPCWalletService {
    static let API_BASE = "http://localhost:8443/api/v1"
    var authToken: String?
    
    init(token: String? = nil) { self.authToken = token }
    
    private func createRequest(endpoint: String, method: String = "GET", body: [String: Any]? = nil) -> URLRequest? {
        guard let url = URL(string: "\(Self.API_BASE)\(endpoint)") else { return nil }
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let token = authToken { request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization") }
        if let body = body { request.httpBody = try? JSONSerialization.data(withJSONObject: body) }
        return request
    }
    
    /// Create share
    func createShare(deviceId: String, publicKey: String, completion: @escaping (Result<MPCShare, Error>) -> Void) {
        guard let request = createRequest(endpoint: "/mpc/shares", method: "POST", body: ["deviceId": deviceId, "publicKey": publicKey]) else {
            completion(.failure(MPCError.invalidRequest)); return
        }
        
        URLSession.shared.dataTask(with: request) { data, _, error in
            if let error = error { completion(.failure(error)); return }
            guard let data = data, let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let shareData = json["data"] as? [String: Any] else { completion(.failure(MPCError.invalidResponse)); return }
            let share = MPCShare(id: shareData["id"] as? String ?? "", deviceId: shareData["deviceId"] as? String ?? "",
                               publicKey: shareData["publicKey"] as? String ?? "", status: shareData["status"] as? String ?? "ACTIVE")
            completion(.success(share))
        }.resume()
    }
    
    /// Sign transaction
    func signTransaction(txHash: String, completion: @escaping (Result<String, Error>) -> Void) {
        guard let request = createRequest(endpoint: "/mpc/sign", method: "POST", body: ["txHash": txHash]) else {
            completion(.failure(MPCError.invalidRequest)); return
        }
        
        URLSession.shared.dataTask(with: request) { data, _, error in
            if let error = error { completion(.failure(error)); return }
            guard let data = data, let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let sigData = json["data"] as? [String: Any], let sig = sigData["signature"] as? String else {
                completion(.failure(MPCError.invalidResponse)); return
            }
            completion(.success(sig))
        }.resume()
    }
    
    /// Get address
    func getWalletAddress(completion: @escaping (Result<String, Error>) -> Void) {
        guard let request = createRequest(endpoint: "/mpc/address") else { completion(.failure(MPCError.invalidRequest)); return }
        
        URLSession.shared.dataTask(with: request) { data, _, error in
            if let error = error { completion(.failure(error)); return }
            guard let data = data, let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let addrData = json["data"] as? [String: Any], let address = addrData["address"] as? String else {
                completion(.failure(MPCError.invalidResponse)); return
            }
            completion(.success(address))
        }.resume()
    }
}

struct MPCShare { let id: String; let deviceId: String; let publicKey: String; let status: String }
enum MPCError: Error { case invalidRequest, invalidResponse }

/**
 * Social Recovery Service - iOS Native Implementation
 */
class SocialRecoveryService {
    static let API_BASE = "http://localhost:8443/api/v1"
    var authToken: String?
    
    init(token: String? = nil) { self.authToken = token }
    
    private func createRequest(endpoint: String, method: String = "GET", body: [String: Any]? = nil) -> URLRequest? {
        guard let url = URL(string: "\(Self.API_BASE)\(endpoint)") else { return nil }
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let token = authToken { request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization") }
        if let body = body { request.httpBody = try? JSONSerialization.data(withJSONObject: body) }
        return request
    }
    
    /// Setup recovery
    func setupRecovery(guardians: [[String: String]], completion: @escaping (Result<Bool, Error>) -> Void) {
        guard let request = createRequest(endpoint: "/recovery/setup", method: "POST", body: ["guardians": guardians]) else {
            completion(.failure(RecoveryError.invalidRequest)); return
        }
        
        URLSession.shared.dataTask(with: request) { _, response, error in
            if let error = error { completion(.failure(error)); return }
            if let httpResponse = response as? HTTPURLResponse {
                completion(.success(httpResponse.statusCode == 201))
            } else {
                completion(.failure(RecoveryError.invalidResponse))
            }
        }.resume()
    }
    
    /// Get guardians
    func getGuardians(completion: @escaping (Result<[Guardian], Error>) -> Void) {
        guard let request = createRequest(endpoint: "/recovery/guardians") else { completion(.failure(RecoveryError.invalidRequest)); return }
        
        URLSession.shared.dataTask(with: request) { data, _, error in
            if let error = error { completion(.failure(error)); return }
            guard let data = data, let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let guardiansData = json["data"] as? [[String: Any]] else { completion(.failure(RecoveryError.invalidResponse)); return }
            let guardians = guardiansData.compactMap { g -> Guardian? in
                guard let address = g["address"] as? String else { return nil }
                return Guardian(address: address, name: g["name"] as? String ?? "", relationship: g["relationship"] as? String ?? "", status: g["status"] as? String ?? "PENDING")
            }
            completion(.success(guardians))
        }.resume()
    }
}

struct Guardian { let address: String; let name: String; let relationship: String; let status: String }
enum RecoveryError: Error { case invalidRequest, invalidResponse }

/**
 * Account Abstraction Service - iOS Native Implementation
 */
class AccountAbstractionService {
    static let API_BASE = "http://localhost:8443/api/v1"
    var authToken: String?
    
    init(token: String? = nil) { self.authToken = token }
    
    private func createRequest(endpoint: String, method: String = "GET", body: [String: Any]? = nil) -> URLRequest? {
        guard let url = URL(string: "\(Self.API_BASE)\(endpoint)") else { return nil }
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let token = authToken { request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization") }
        if let body = body { request.httpBody = try? JSONSerialization.data(withJSONObject: body) }
        return request
    }
    
    /// Create account
    func createAccount(ownerAddress: String, salt: String?, completion: @escaping (Result<SmartAccount, Error>) -> Void) {
        var body: [String: Any] = ["ownerAddress": ownerAddress]
        if let salt = salt { body["salt"] = salt }
        
        guard let request = createRequest(endpoint: "/account/create", method: "POST", body: body) else {
            completion(.failure(AAError.invalidRequest)); return
        }
        
        URLSession.shared.dataTask(with: request) { data, _, error in
            if let error = error { completion(.failure(error)); return }
            guard let data = data, let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let accountData = json["data"] as? [String: Any] else { completion(.failure(AAError.invalidResponse)); return }
            let account = SmartAccount(id: accountData["id"] as? String ?? "", accountAddress: accountData["accountAddress"] as? String ?? "",
                                     ownerAddress: accountData["ownerAddress"] as? String ?? "", nonce: accountData["nonce"] as? Int ?? 0,
                                     threshold: accountData["threshold"] as? Int ?? 1, status: accountData["status"] as? String ?? "ACTIVE",
                                     deployed: accountData["deployed"] as? Bool ?? false)
            completion(.success(account))
        }.resume()
    }
    
    /// Get accounts
    func getAccounts(completion: @escaping (Result<[SmartAccount], Error>) -> Void) {
        guard let request = createRequest(endpoint: "/account/list") else { completion(.failure(AAError.invalidRequest)); return }
        
        URLSession.shared.dataTask(with: request) { data, _, error in
            if let error = error { completion(.failure(error)); return }
            guard let data = data, let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let accountsData = json["data"] as? [[String: Any]] else { completion(.failure(AAError.invalidResponse)); return }
            let accounts = accountsData.compactMap { a -> SmartAccount? in
                guard let id = a["id"] as? String else { return nil }
                return SmartAccount(id: id, accountAddress: a["accountAddress"] as? String ?? "", ownerAddress: a["ownerAddress"] as? String ?? "",
                                   nonce: a["nonce"] as? Int ?? 0, threshold: a["threshold"] as? Int ?? 1, status: a["status"] as? String ?? "ACTIVE",
                                   deployed: a["deployed"] as? Bool ?? false)
            }
            completion(.success(accounts))
        }.resume()
    }
}

struct SmartAccount {
    let id: String
    let accountAddress: String
    let ownerAddress: String
    let nonce: Int
    let threshold: Int
    let status: String
    let deployed: Bool
}

enum AAError: Error { case invalidRequest, invalidResponse }
