import Foundation

class FiatRampService {
    static let API_BASE = "https://api.tigerwallet.com/api/v1"
    var authToken: String?
    
    init(token: String?) { self.authToken = token }
    
    private var headers: [String: String] {
        var h = ["Content-Type": "application/json"]
        if let token = authToken { h["Authorization"] = "Bearer \(token)" }
        return h
    }
    
    func buyCrypto(amount: Double, currency: String, crypto: String, completion: @escaping (String?) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/fiat/buy") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.allHTTPHeaderFields = headers
        request.httpBody = try? JSONSerialization.data(withJSONObject: ["amount": amount, "currency": currency, "crypto": crypto])
        
        URLSession.shared.dataTask(with: request) { data, _, _ in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let orderId = (json["data"] as? [String: Any])?["orderId"] as? String else {
                completion(nil)
                return
            }
            completion(orderId)
        }.resume()
    }
    
    func sellCrypto(amount: Double, crypto: String, currency: String, completion: @escaping (String?) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/fiat/sell") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.allHTTPHeaderFields = headers
        request.httpBody = try? JSONSerialization.data(withJSONObject: ["amount": amount, "crypto": crypto, "currency": currency])
        
        URLSession.shared.dataTask(with: request) { data, _, _ in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let orderId = (json["data"] as? [String: Any])?["orderId"] as? String else {
                completion(nil)
                return
            }
            completion(orderId)
        }.resume()
    }
}

class GiftCardService {
    static let API_BASE = "https://api.tigerwallet.com/api/v1"
    var authToken: String?
    
    init(token: String?) { self.authToken = token }
    
    private var headers: [String: String] {
        var h = ["Content-Type": "application/json"]
        if let token = authToken { h["Authorization"] = "Bearer \(token)" }
        return h
    }
    
    func getBrands(completion: @escaping ([GiftCardBrand]) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/giftcards/brands") else { return }
        var request = URLRequest(url: url)
        request.allHTTPHeaderFields = headers
        
        URLSession.shared.dataTask(with: request) { data, _, _ in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let array = json["data"] as? [[String: Any]] else {
                completion([])
                return
            }
            let brands = array.compactMap { dict -> GiftCardBrand? in
                guard let id = dict["id"] as? String, let name = dict["name"] as? String else { return nil }
                return GiftCardBrand(id: id, name: name, discount: dict["discount"] as? Double ?? 0)
            }
            completion(brands)
        }.resume()
    }
    
    func buyGiftCard(brandId: String, amount: Double, completion: @escaping (String?) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/giftcards/buy") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.allHTTPHeaderFields = headers
        request.httpBody = try? JSONSerialization.data(withJSONObject: ["brandId": brandId, "amount": amount])
        
        URLSession.shared.dataTask(with: request) { data, _, _ in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let code = (json["data"] as? [String: Any])?["code"] as? String else {
                completion(nil)
                return
            }
            completion(code)
        }.resume()
    }
}

struct GiftCardBrand { let id: String; let name: String; let discount: Double }

class DAppBrowserService {
    static let API_BASE = "https://api.tigerwallet.com/api/v1"
    var authToken: String?
    
    init(token: String?) { self.authToken = token }
    
    private var headers: [String: String] {
        var h = ["Content-Type": "application/json"]
        if let token = authToken { h["Authorization"] = "Bearer \(token)" }
        return h
    }
    
    func getFeaturedDApps(completion: @escaping ([DApp]) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/dapp/featured") else { return }
        var request = URLRequest(url: url)
        request.allHTTPHeaderFields = headers
        
        URLSession.shared.dataTask(with: request) { data, _, _ in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let array = json["data"] as? [[String: Any]] else {
                completion([])
                return
            }
            let dapps = array.compactMap { dict -> DApp? in
                guard let id = dict["id"] as? String, let name = dict["name"] as? String, let url = dict["url"] as? String else { return nil }
                return DApp(id: id, name: name, url: url, category: dict["category"] as? String ?? "")
            }
            completion(dapps)
        }.resume()
    }
    
    func connectWallet(dappId: String, address: String, chain: String, completion: @escaping (Bool) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/dapp/connect") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.allHTTPHeaderFields = headers
        request.httpBody = try? JSONSerialization.data(withJSONObject: ["dappId": dappId, "address": address, "chain": chain])
        
        URLSession.shared.dataTask(with: request) { _, response, _ in
            if let http = response as? HTTPURLResponse { completion(http.statusCode == 201) }
            else { completion(false) }
        }.resume()
    }
}

struct DApp { let id: String; let name: String; let url: String; let category: String }
