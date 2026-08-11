//
//  HardwareWalletService.swift
//  TigerWallet
//
//  Complete Hardware Wallet Service for iOS
//

import Foundation

class HardwareWalletService {
    static let shared = HardwareWalletService()
    private let baseURL = "http://localhost:8443/api/v1/hardware"
    
    private init() {}
    
    // MARK: - Supported Devices
    
    func getSupportedDevices() -> [HardwareWalletDevice] {
        return [
            HardwareWalletDevice(id: "ledger", name: "Ledger", icon: "ledger_icon", supportedChains: ["ethereum", "bitcoin", "solana", "polygon", "bsc"]),
            HardwareWalletDevice(id: "trezor", name: "Trezor", icon: "trezor_icon", supportedChains: ["ethereum", "bitcoin"]),
            HardwareWalletDevice(id: "keepkey", name: "KeepKey", icon: "keepkey_icon", supportedChains: ["ethereum", "bitcoin"]),
            HardwareWalletDevice(id: "coldcard", name: "ColdCard", icon: "coldcard_icon", supportedChains: ["bitcoin"]),
            HardwareWalletDevice(id: "bitbox02", name: "BitBox02", icon: "bitbox02_icon", supportedChains: ["ethereum", "bitcoin"])
        ]
    }
    
    // MARK: - Connect
    
    func connect(deviceId: String, completion: @escaping (Result<HardwareWalletConnection, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/connect")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = ["deviceId": deviceId]
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data else {
                completion(.failure(NSError(domain: "HardwareWalletService", code: -1, userInfo: [NSLocalizedDescriptionKey: "No data"])))
                return
            }
            
            do {
                let connection = try JSONDecoder().decode(HardwareWalletConnection.self, from: data)
                completion(.success(connection))
            } catch {
                completion(.failure(error))
            }
        }.resume()
    }
    
    // MARK: - Get Address
    
    func getAddress(deviceId: String, chain: String, path: String, completion: @escaping (Result<String, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/address")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = ["deviceId": deviceId, "chain": chain, "path": path]
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let address = json["address"] as? String else {
                completion(.failure(NSError(domain: "HardwareWalletService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to get address"])))
                return
            }
            
            completion(.success(address))
        }.resume()
    }
    
    // MARK: - Sign Transaction
    
    func signTransaction(deviceId: String, txData: String, chain: String, completion: @escaping (Result<String, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/sign")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = ["deviceId": deviceId, "txData": txData, "chain": chain]
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let signature = json["signature"] as? String else {
                completion(.failure(NSError(domain: "HardwareWalletService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to sign"])))
                return
            }
            
            completion(.success(signature))
        }.resume()
    }
    
    // MARK: - Sign Message
    
    func signMessage(deviceId: String, message: String, completion: @escaping (Result<String, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/sign-message")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = ["deviceId": deviceId, "message": message]
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let signature = json["signature"] as? String else {
                completion(.failure(NSError(domain: "HardwareWalletService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to sign"])))
                return
            }
            
            completion(.success(signature))
        }.resume()
    }
    
    // MARK: - Disconnect
    
    func disconnect(deviceId: String, completion: @escaping (Result<Bool, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/disconnect")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = ["deviceId": deviceId]
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        
        URLSession.shared.dataTask(with: request) { _, _, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            completion(.success(true))
        }.resume()
    }
}

struct HardwareWalletDevice {
    let id: String
    let name: String
    let icon: String
    let supportedChains: [String]
}

struct HardwareWalletConnection: Codable {
    let deviceId: String
    let address: String
    let publicKey: String
    let chain: String
    let connectedAt: Date
}
