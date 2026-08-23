//
//  AccountAbstractionService.swift
//  TigerWallet
//
//  Complete Account Abstraction Service for iOS
//

import Foundation

class AccountAbstractionService {
    static let shared = AccountAbstractionService()
    private let baseURL = "http://localhost:8443/api/v1/aa"
    
    private init() {}
    
    func createAccount(ownerAddress: String, salt: String? = nil, completion: @escaping (Result<SmartAccount, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/account")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        var body: [String: Any] = ["ownerAddress": ownerAddress]
        if let salt = salt {
            body["salt"] = salt
        }
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data else {
                completion(.failure(NSError(domain: "AAService", code: -1, userInfo: [NSLocalizedDescriptionKey: "No data"])))
                return
            }
            
            do {
                let account = try JSONDecoder().decode(SmartAccount.self, from: data)
                completion(.success(account))
            } catch {
                completion(.failure(error))
            }
        }.resume()
    }
    
    func getAccountAddress(ownerAddress: String, index: Int = 0, completion: @escaping (Result<String, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/address/\(ownerAddress)/\(index)")!
        
        URLSession.shared.dataTask(with: url) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let address = json["address"] as? String else {
                completion(.failure(NSError(domain: "AAService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to get address"])))
                return
            }
            
            completion(.success(address))
        }.resume()
    }
    
    func getNonce(accountAddress: String, completion: @escaping (Result<UInt64, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/nonce/\(accountAddress)")!
        
        URLSession.shared.dataTask(with: url) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let nonce = json["nonce"] as? UInt64 else {
                completion(.failure(NSError(domain: "AAService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to get nonce"])))
                return
            }
            
            completion(.success(nonce))
        }.resume()
    }
    
    func executeUserOp(userOp: UserOperation, completion: @escaping (Result<String, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/execute")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        request.httpBody = try? JSONEncoder().encode(userOp)
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let userOpHash = json["userOpHash"] as? String else {
                completion(.failure(NSError(domain: "AAService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to execute"])))
                return
            }
            
            completion(.success(userOpHash))
        }.resume()
    }
    
    func estimateGas(userOp: UserOperation, completion: @escaping (Result<GasEstimate, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/estimate")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        request.httpBody = try? JSONEncoder().encode(userOp)
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data else {
                completion(.failure(NSError(domain: "AAService", code: -1, userInfo: [NSLocalizedDescriptionKey: "No data"])))
                return
            }
            
            do {
                let estimate = try JSONDecoder().decode(GasEstimate.self, from: data)
                completion(.success(estimate))
            } catch {
                completion(.failure(error))
            }
        }.resume()
    }
}

struct SmartAccount: Codable {
    let address: String
    let owner: String
    let nonce: UInt64
    let factory: String
}

struct UserOperation: Codable {
    let sender: String
    let nonce: String
    let initCode: String
    let callData: String
    let callGasLimit: String
    let verificationGasLimit: String
    let preVerificationGas: String
    let maxFeePerGas: String
    let maxPriorityFeePerGas: String
    let paymasterAndData: String
    let signature: String
}

struct GasEstimate: Codable {
    let callGasLimit: String
    let verificationGasLimit: String
    let preVerificationGas: String
}
