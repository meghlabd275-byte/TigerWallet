import Foundation

/**
 * Bridge Service - iOS Native Implementation
 * Cross-chain bridging
 */
class BridgeService {
    static let API_BASE = "https://api.tigerwallet.com/api/v1"
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
    
    /// Get supported chains
    func getSupportedChains(completion: @escaping (Result<[Chain], Error>) -> Void) {
        guard let request = createRequest(endpoint: "/bridge/chains") else {
            completion(.failure(BridgeError.invalidRequest))
            return
        }
        
        URLSession.shared.dataTask(with: request) { data, _, error in
            if let error = error { completion(.failure(error)); return }
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let chainsData = json["data"] as? [[String: Any]] else {
                completion(.failure(BridgeError.invalidResponse)); return
            }
            
            let chains = chainsData.compactMap { c -> Chain? in
                guard let id = c["id"] as? String, let name = c["name"] as? String else { return nil }
                return Chain(id: id, name: name, symbol: c["symbol"] as? String ?? "", isActive: c["isActive"] as? Bool ?? false)
            }
            completion(.success(chains))
        }.resume()
    }
    
    /// Get tokens for chain
    func getTokens(chain: String, completion: @escaping (Result<[BridgeToken], Error>) -> Void) {
        guard let request = createRequest(endpoint: "/bridge/tokens?chain=\(chain)") else {
            completion(.failure(BridgeError.invalidRequest)); return
        }
        
        URLSession.shared.dataTask(with: request) { data, _, error in
            if let error = error { completion(.failure(error)); return }
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let tokensData = json["data"] as? [[String: Any]] else {
                completion(.failure(BridgeError.invalidResponse)); return
            }
            
            let tokens = tokensData.compactMap { t -> BridgeToken? in
                guard let token = t["token"] as? String else { return nil }
                return BridgeToken(
                    token: token,
                    name: t["name"] as? String ?? "",
                    symbol: t["symbol"] as? String ?? "",
                    minAmount: t["minAmount"] as? Double ?? 0,
                    maxAmount: t["maxAmount"] as? Double ?? 0,
                    isActive: t["isActive"] as? Bool ?? false
                )
            }
            completion(.success(tokens))
        }.resume()
    }
    
    /// Get bridge estimate
    func getEstimate(fromChain: String, toChain: String, token: String, amount: Double, completion: @escaping (Result<BridgeEstimate, Error>) -> Void) {
        guard let request = createRequest(endpoint: "/bridge/estimate?fromChain=\(fromChain)&toChain=\(toChain)&token=\(token)&amount=\(amount)") else {
            completion(.failure(BridgeError.invalidRequest)); return
        }
        
        URLSession.shared.dataTask(with: request) { data, _, error in
            if let error = error { completion(.failure(error)); return }
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let estimateData = json["data"] as? [String: Any] else {
                completion(.failure(BridgeError.invalidResponse)); return
            }
            
            let estimate = BridgeEstimate(
                receivedAmount: estimateData["receivedAmount"] as? Double ?? 0,
                fee: estimateData["fee"] as? Double ?? 0,
                feePercentage: estimateData["feePercentage"] as? Double ?? 0,
                estimatedTime: estimateData["estimatedTime"] as? String ?? ""
            )
            completion(.success(estimate))
        }.resume()
    }
    
    /// Initiate bridge transaction
    func initiateBridge(fromChain: String, toChain: String, token: String, amount: Double, completion: @escaping (Result<BridgeTransaction, Error>) -> Void) {
        guard let request = createRequest(endpoint: "/bridge/transactions", method: "POST", body: [
            "fromChain": fromChain,
            "toChain": toChain,
            "token": token,
            "amount": amount
        ]) else {
            completion(.failure(BridgeError.invalidRequest)); return
        }
        
        URLSession.shared.dataTask(with: request) { data, _, error in
            if let error = error { completion(.failure(error)); return }
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let txData = json["data"] as? [String: Any] else {
                completion(.failure(BridgeError.invalidResponse)); return
            }
            
            let tx = BridgeTransaction(
                id: txData["id"] as? String ?? "",
                fromChain: txData["fromChain"] as? String ?? "",
                toChain: txData["toChain"] as? String ?? "",
                token: txData["token"] as? String ?? "",
                amount: txData["amount"] as? Double ?? 0,
                fee: txData["fee"] as? Double ?? 0,
                receivedAmount: txData["receivedAmount"] as? Double ?? 0,
                status: txData["status"] as? String ?? "PENDING"
            )
            completion(.success(tx))
        }.resume()
    }
    
    /// Get user transactions
    func getUserTransactions(completion: @escaping (Result<[BridgeTransaction], Error>) -> Void) {
        guard let request = createRequest(endpoint: "/bridge/transactions") else {
            completion(.failure(BridgeError.invalidRequest)); return
        }
        
        URLSession.shared.dataTask(with: request) { data, _, error in
            if let error = error { completion(.failure(error)); return }
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let txsData = json["data"] as? [[String: Any]] else {
                completion(.failure(BridgeError.invalidResponse)); return
            }
            
            let txs = txsData.compactMap { t -> BridgeTransaction? in
                guard let id = t["id"] as? String else { return nil }
                return BridgeTransaction(
                    id: id,
                    fromChain: t["fromChain"] as? String ?? "",
                    toChain: t["toChain"] as? String ?? "",
                    token: t["token"] as? String ?? "",
                    amount: t["amount"] as? Double ?? 0,
                    fee: t["fee"] as? Double ?? 0,
                    receivedAmount: t["receivedAmount"] as? Double ?? 0,
                    status: t["status"] as? String ?? "PENDING"
                )
            }
            completion(.success(txs))
        }.resume()
    }
}

struct Chain {
    let id: String
    let name: String
    let symbol: String
    let isActive: Bool
}

struct BridgeToken {
    let token: String
    let name: String
    let symbol: String
    let minAmount: Double
    let maxAmount: Double
    let isActive: Bool
}

struct BridgeEstimate {
    let receivedAmount: Double
    let fee: Double
    let feePercentage: Double
    let estimatedTime: String
}

struct BridgeTransaction {
    let id: String
    let fromChain: String
    let toChain: String
    let token: String
    let amount: Double
    let fee: Double
    let receivedAmount: Double
    let status: String
}

enum BridgeError: Error {
    case invalidRequest
    case invalidResponse
}
