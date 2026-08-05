//
//  StakingService.swift
//  TigerWallet
//
//  Complete Staking Service for iOS
//

import Foundation

class StakingService {
    static let shared = StakingService()
    private let baseURL = "https://api.tigerwallet.com/v1/staking"
    
    private init() {}
    
    func getChains(completion: @escaping (Result<[StakingChain], Error>) -> Void) {
        let url = URL(string: "\(baseURL)/chains")!
        
        URLSession.shared.dataTask(with: url) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data else {
                completion(.failure(NSError(domain: "StakingService", code: -1, userInfo: [NSLocalizedDescriptionKey: "No data"])))
                return
            }
            
            do {
                let chains = try JSONDecoder().decode([StakingChain].self, from: data)
                completion(.success(chains))
            } catch {
                completion(.failure(error))
            }
        }.resume()
    }
    
    func getValidators(chain: String, completion: @escaping (Result<[Validator], Error>) -> Void) {
        let url = URL(string: "\(baseURL)/\(chain)/validators")!
        
        URLSession.shared.dataTask(with: url) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data else {
                completion(.failure(NSError(domain: "StakingService", code: -1, userInfo: [NSLocalizedDescriptionKey: "No data"])))
                return
            }
            
            do {
                let validators = try JSONDecoder().decode([Validator].self, from: data)
                completion(.success(validators))
            } catch {
                completion(.failure(error))
            }
        }.resume()
    }
    
    func stake(chain: String, validatorAddress: String, amount: String, address: String, completion: @escaping (Result<String, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/\(chain)/stake")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "validatorAddress": validatorAddress,
            "amount": amount,
            "address": address
        ]
        
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let txHash = json["txHash"] as? String else {
                completion(.failure(NSError(domain: "StakingService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Stake failed"])))
                return
            }
            
            completion(.success(txHash))
        }.resume()
    }
    
    func unstake(chain: String, validatorAddress: String, amount: String, address: String, completion: @escaping (Result<String, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/\(chain)/unstake")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "validatorAddress": validatorAddress,
            "amount": amount,
            "address": address
        ]
        
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let txHash = json["txHash"] as? String else {
                completion(.failure(NSError(domain: "StakingService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Unstake failed"])))
                return
            }
            
            completion(.success(txHash))
        }.resume()
    }
    
    func claimRewards(chain: String, validatorAddress: String, address: String, completion: @escaping (Result<String, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/\(chain)/claim")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "validatorAddress": validatorAddress,
            "address": address
        ]
        
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let txHash = json["txHash"] as? String else {
                completion(.failure(NSError(domain: "StakingService", code: -1, userInfo: [NSLocalizedDescriptionKey: "Claim failed"])))
                return
            }
            
            completion(.success(txHash))
        }.resume()
    }
    
    func getPositions(address: String, chain: String, completion: @escaping (Result<[StakingPosition], Error>) -> Void) {
        let url = URL(string: "\(baseURL)/\(chain)/positions/\(address)")!
        
        URLSession.shared.dataTask(with: url) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data else {
                completion(.failure(NSError(domain: "StakingService", code: -1, userInfo: [NSLocalizedDescriptionKey: "No data"])))
                return
            }
            
            do {
                let positions = try JSONDecoder().decode([StakingPosition].self, from: data)
                completion(.success(positions))
            } catch {
                completion(.failure(error))
            }
        }.resume()
    }
}

struct StakingChain: Codable {
    let id: String
    let name: String
    let symbol: String
    let stakingAPY: Double
    let minStake: String
    let lockPeriod: Int
}

struct Validator: Codable {
    let address: String
    let name: String
    let commission: Double
    let apy: Double
    let totalStaked: String
    let delegators: Int
    let uptime: Double
}

struct StakingPosition: Codable {
    let validatorAddress: String
    let stakedAmount: String
    let rewardsAccrued: String
    let rewardsClaimed: String
    let chain: String
    let status: String
}
