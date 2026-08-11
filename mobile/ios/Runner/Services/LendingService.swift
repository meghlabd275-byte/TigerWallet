import Foundation

/**
 * Lending Service - iOS Native Implementation
 * Real backend connection to Go lending service
 */
class LendingService {
    static let API_BASE = "http://localhost:8443/api/v1"
    var authToken: String?
    
    init(token: String? = nil) {
        self.authToken = token
    }
    
    private func createRequest(endpoint: String, method: String = "GET", body: [String: Any]? = nil) -> URLRequest? {
        guard let url = URL(string: "\(Self.API_BASE)\(endpoint)") else { return nil }
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let token = authToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        if let body = body {
            request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        }
        return request
    }
    
    /// Get all lending pools
    func getPools(completion: @escaping (Result<[LendingPool], Error>) -> Void) {
        guard let request = createRequest(endpoint: "/lending/pools") else {
            completion(.failure(LendingError.invalidRequest))
            return
        }
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let poolsData = json["data"] as? [[String: Any]] else {
                completion(.failure(LendingError.invalidResponse))
                return
            }
            
            let pools = poolsData.compactMap { poolJson -> LendingPool? in
                guard let token = poolJson["token"] as? String,
                      let name = poolJson["name"] as? String else { return nil }
                return LendingPool(
                    token: token,
                    name: name,
                    totalSupplied: poolJson["totalSupplied"] as? Double ?? 0,
                    totalBorrowed: poolJson["totalBorrowed"] as? Double ?? 0,
                    supplyAPY: poolJson["supplyAPY"] as? Double ?? 0,
                    borrowAPY: poolJson["borrowAPY"] as? Double ?? 0,
                    liquidity: poolJson["liquidity"] as? Double ?? 0
                )
            }
            completion(.success(pools))
        }.resume()
    }
    
    /// Supply assets to pool
    func supply(token: String, amount: Double, completion: @escaping (Result<LendingPosition, Error>) -> Void) {
        guard let request = createRequest(endpoint: "/lending/supply", method: "POST", body: [
            "token": token,
            "amount": amount
        ]) else {
            completion(.failure(LendingError.invalidRequest))
            return
        }
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let positionData = json["data"] as? [String: Any] else {
                completion(.failure(LendingError.invalidResponse))
                return
            }
            
            let position = LendingPosition(
                id: positionData["id"] as? String ?? "",
                token: positionData["token"] as? String ?? "",
                supplied: positionData["supplied"] as? Double ?? 0,
                borrowed: positionData["borrowed"] as? Double ?? 0,
                apy: positionData["apy"] as? Double ?? 0,
                status: positionData["status"] as? String ?? ""
            )
            completion(.success(position))
        }.resume()
    }
    
    /// Borrow from pool
    func borrow(token: String, amount: Double, completion: @escaping (Result<LendingPosition, Error>) -> Void) {
        guard let request = createRequest(endpoint: "/lending/borrow", method: "POST", body: [
            "token": token,
            "amount": amount
        ]) else {
            completion(.failure(LendingError.invalidRequest))
            return
        }
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let positionData = json["data"] as? [String: Any] else {
                completion(.failure(LendingError.invalidResponse))
                return
            }
            
            let position = LendingPosition(
                id: positionData["id"] as? String ?? "",
                token: positionData["token"] as? String ?? "",
                supplied: positionData["supplied"] as? Double ?? 0,
                borrowed: positionData["borrowed"] as? Double ?? 0,
                apy: positionData["apy"] as? Double ?? 0,
                status: positionData["status"] as? String ?? ""
            )
            completion(.success(position))
        }.resume()
    }
    
    /// Repay borrowed amount
    func repay(token: String, amount: Double, completion: @escaping (Result<Bool, Error>) -> Void) {
        guard let request = createRequest(endpoint: "/lending/repay", method: "POST", body: [
            "token": token,
            "amount": amount
        ]) else {
            completion(.failure(LendingError.invalidRequest))
            return
        }
        
        URLSession.shared.dataTask(with: request) { _, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            if let httpResponse = response as? HTTPURLResponse {
                completion(.success(httpResponse.statusCode == 200))
            } else {
                completion(.failure(LendingError.invalidResponse))
            }
        }.resume()
    }
    
    /// Withdraw supplied assets
    func withdraw(token: String, amount: Double, completion: @escaping (Result<Bool, Error>) -> Void) {
        guard let request = createRequest(endpoint: "/lending/withdraw", method: "POST", body: [
            "token": token,
            "amount": amount
        ]) else {
            completion(.failure(LendingError.invalidRequest))
            return
        }
        
        URLSession.shared.dataTask(with: request) { _, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            if let httpResponse = response as? HTTPURLResponse {
                completion(.success(httpResponse.statusCode == 200))
            } else {
                completion(.failure(LendingError.invalidResponse))
            }
        }.resume()
    }
    
    /// Get user's positions
    func getUserPositions(completion: @escaping (Result<[LendingPosition], Error>) -> Void) {
        guard let request = createRequest(endpoint: "/lending/positions") else {
            completion(.failure(LendingError.invalidRequest))
            return
        }
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let positionsData = json["data"] as? [[String: Any]] else {
                completion(.failure(LendingError.invalidResponse))
                return
            }
            
            let positions = positionsData.compactMap { posJson -> LendingPosition? in
                guard let id = posJson["id"] as? String,
                      let token = posJson["token"] as? String else { return nil }
                return LendingPosition(
                    id: id,
                    token: token,
                    supplied: posJson["supplied"] as? Double ?? 0,
                    borrowed: posJson["borrowed"] as? Double ?? 0,
                    apy: posJson["apy"] as? Double ?? 0,
                    status: posJson["status"] as? String ?? ""
                )
            }
            completion(.success(positions))
        }.resume()
    }
    
    /// Get health factor
    func getHealthFactor(completion: @escaping (Result<Double, Error>) -> Void) {
        guard let request = createRequest(endpoint: "/lending/health") else {
            completion(.failure(LendingError.invalidRequest))
            return
        }
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let healthFactor = json["data"] as? Double else {
                completion(.failure(LendingError.invalidResponse))
                return
            }
            completion(.success(healthFactor))
        }.resume()
    }
}

// MARK: - Data Models
struct LendingPool {
    let token: String
    let name: String
    let totalSupplied: Double
    let totalBorrowed: Double
    let supplyAPY: Double
    let borrowAPY: Double
    let liquidity: Double
}

struct LendingPosition {
    let id: String
    let token: String
    let supplied: Double
    let borrowed: Double
    let apy: Double
    let status: String
}

enum LendingError: Error {
    case invalidRequest
    case invalidResponse
}
