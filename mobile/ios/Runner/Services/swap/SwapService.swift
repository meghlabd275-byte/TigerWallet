import Foundation

class SwapService {
    static let API_BASE = "https://api.tigerwallet.com/api/v1"
    var authToken: String?
    
    init(token: String?) { self.authToken = token }
    
    private var headers: [String: String] {
        var h = ["Content-Type": "application/json"]
        if let token = authToken { h["Authorization"] = "Bearer \(token)" }
        return h
    }
    
    func getTokens(completion: @escaping ([Token]) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/swap/tokens") else { return }
        var request = URLRequest(url: url)
        request.allHTTPHeaderFields = headers
        
        URLSession.shared.dataTask(with: request) { data, _, _ in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let array = json["data"] as? [[String: Any]] else {
                completion([])
                return
            }
            let tokens = array.compactMap { dict -> Token? in
                guard let symbol = dict["symbol"] as? String, let name = dict["name"] as? String else { return nil }
                return Token(symbol: symbol, name: name, price: dict["price"] as? Double ?? 0)
            }
            completion(tokens)
        }.resume()
    }
    
    func getQuote(from: String, to: String, amount: Double, completion: @escaping (SwapQuote?) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/swap/quote?from=\(from)&to=\(to)&amount=\(amount)") else { return }
        var request = URLRequest(url: url)
        request.allHTTPHeaderFields = headers
        
        URLSession.shared.dataTask(with: request) { data, _, _ in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let dataObj = json["data"] as? [String: Any] else {
                completion(nil)
                return
            }
            let quote = SwapQuote(fromAmount: dataObj["fromAmount"] as? Double ?? 0,
                                  toAmount: dataObj["toAmount"] as? Double ?? 0,
                                  priceImpact: dataObj["priceImpact"] as? Double ?? 0,
                                  gasFee: dataObj["gasFee"] as? Double ?? 0)
            completion(quote)
        }.resume()
    }
    
    func executeSwap(from: String, to: String, amount: Double, slippage: Double, completion: @escaping (String?) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/swap/execute") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.allHTTPHeaderFields = headers
        request.httpBody = try? JSONSerialization.data(withJSONObject: ["fromToken": from, "toToken": to, "amount": amount, "slippage": slippage])
        
        URLSession.shared.dataTask(with: request) { data, _, _ in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let txHash = (json["data"] as? [String: Any])?["txHash"] as? String else {
                completion(nil)
                return
            }
            completion(txHash)
        }.resume()
    }
}

struct Token { let symbol: String; let name: String; let price: Double }
struct SwapQuote { let fromAmount: Double; let toAmount: Double; let priceImpact: Double; let gasFee: Double }

class StakingService {
    static let API_BASE = "https://api.tigerwallet.com/api/v1"
    var authToken: String?
    
    init(token: String?) { self.authToken = token }
    
    private var headers: [String: String] {
        var h = ["Content-Type": "application/json"]
        if let token = authToken { h["Authorization"] = "Bearer \(token)" }
        return h
    }
    
    func getPools(completion: @escaping ([StakingPool]) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/staking/pools") else { return }
        var request = URLRequest(url: url)
        request.allHTTPHeaderFields = headers
        
        URLSession.shared.dataTask(with: request) { data, _, _ in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let array = json["data"] as? [[String: Any]] else {
                completion([])
                return
            }
            let pools = array.compactMap { dict -> StakingPool? in
                guard let id = dict["id"] as? String, let token = dict["token"] as? String else { return nil }
                return StakingPool(id: id, token: token, apy: dict["apy"] as? Double ?? 0, totalStaked: dict["totalStaked"] as? Double ?? 0)
            }
            completion(pools)
        }.resume()
    }
    
    func stake(poolId: String, amount: Double, completion: @escaping (Bool) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/staking/stake") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.allHTTPHeaderFields = headers
        request.httpBody = try? JSONSerialization.data(withJSONObject: ["poolId": poolId, "amount": amount])
        
        URLSession.shared.dataTask(with: request) { _, response, _ in
            if let http = response as? HTTPURLResponse { completion(http.statusCode == 201) }
            else { completion(false) }
        }.resume()
    }
}

struct StakingPool { let id: String; let token: String; let apy: Double; let totalStaked: Double }

class NFTService {
    static let API_BASE = "https://api.tigerwallet.com/api/v1"
    var authToken: String?
    
    init(token: String?) { self.authToken = token }
    
    private var headers: [String: String] {
        var h = ["Content-Type": "application/json"]
        if let token = authToken { h["Authorization"] = "Bearer \(token)" }
        return h
    }
    
    func getCollections(completion: @escaping ([NFTCollection]) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/nft/collections") else { return }
        var request = URLRequest(url: url)
        request.allHTTPHeaderFields = headers
        
        URLSession.shared.dataTask(with: request) { data, _, _ in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let array = json["data"] as? [[String: Any]] else {
                completion([])
                return
            }
            let collections = array.compactMap { dict -> NFTCollection? in
                guard let id = dict["id"] as? String, let name = dict["name"] as? String else { return nil }
                return NFTCollection(id: id, name: name, floorPrice: dict["floorPrice"] as? Double ?? 0)
            }
            completion(collections)
        }.resume()
    }
    
    func getUserNFTs(completion: @escaping ([NFT]) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/nft/user/nfts") else { return }
        var request = URLRequest(url: url)
        request.allHTTPHeaderFields = headers
        
        URLSession.shared.dataTask(with: request) { data, _, _ in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let array = json["data"] as? [[String: Any]] else {
                completion([])
                return
            }
            let nfts = array.compactMap { dict -> NFT? in
                guard let id = dict["id"] as? String, let name = dict["name"] as? String else { return nil }
                return NFT(id: id, name: name, price: dict["price"] as? Double ?? 0)
            }
            completion(nfts)
        }.resume()
    }
    
    func buyNFT(collectionId: String, tokenId: String, price: Double, completion: @escaping (Bool) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/nft/buy") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.allHTTPHeaderFields = headers
        request.httpBody = try? JSONSerialization.data(withJSONObject: ["collectionId": collectionId, "tokenId": tokenId, "price": price])
        
        URLSession.shared.dataTask(with: request) { _, response, _ in
            if let http = response as? HTTPURLResponse { completion(http.statusCode == 201) }
            else { completion(false) }
        }.resume()
    }
}

struct NFTCollection { let id: String; let name: String; let floorPrice: Double }
struct NFT { let id: String; let name: String; let price: Double }
