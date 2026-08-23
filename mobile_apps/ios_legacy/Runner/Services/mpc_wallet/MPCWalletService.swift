//
//  MPCWalletService.swift
//  TigerWallet
//
//  Complete MPC Wallet Service for iOS
//

import Foundation

class MPCWalletService {
    static let shared = MPCWalletService()
    private let baseURL = "http://localhost:8443/api/v1/mpc"
    
    private init() {}
    
    func createWallet(userId: String, completion: @escaping (Result<MPCWallet, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/wallet")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = ["userId": userId]
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data else {
                completion(.failure(NSError(domain: "MPCWalletService", code: -1, userInfo: [NSLocalizedDescriptionKey: "No data"])))
                return
            }
            
            do {
                let wallet = try JSONDecoder().decode(MPCWallet.self, from: data)
                completion(.success(wallet))
            } catch {
                completion(.failure(error))
            }
        }.resume()
    }
    
    func getPublicKey(address: String, completion: @escaping (Result<String, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/publickey/\(address)")!
        
        URLSession.shared.dataTask(with: url) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let publicKey = json["publicKey"] as? String else {
                completion(.failure(NSError(domain: "MPCWalletService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to get public key"])))
                return
            }
            
            completion(.success(publicKey))
        }.resume()
    }
    
    func signTransaction(txData: String, address: String, completion: @escaping (Result<String, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/sign")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = ["txData": txData, "address": address]
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let signature = json["signature"] as? String else {
                completion(.failure(NSError(domain: "MPCWalletService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to sign"])))
                return
            }
            
            completion(.success(signature))
        }.resume()
    }
    
    func signMessage(message: String, address: String, completion: @escaping (Result<String, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/sign-message")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = ["message": message, "address": address]
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let signature = json["signature"] as? String else {
                completion(.failure(NSError(domain: "MPCWalletService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to sign"])))
                return
            }
            
            completion(.success(signature))
        }.resume()
    }
    
    func getSessionKey(address: String, completion: @escaping (Result<SessionKey, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/session/\(address)")!
        
        URLSession.shared.dataTask(with: url) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data else {
                completion(.failure(NSError(domain: "MPCWalletService", code: -1, userInfo: [NSLocalizedDescriptionKey: "No data"])))
                return
            }
            
            do {
                let sessionKey = try JSONDecoder().decode(SessionKey.self, from: data)
                completion(.success(sessionKey))
            } catch {
                completion(.failure(error))
            }
        }.resume()
    }
}

struct MPCWallet: Codable {
    let address: String
    let publicKey: String
    let createdAt: Date
}

struct SessionKey: Codable {
    let key: String
    let expiresAt: Date
}
