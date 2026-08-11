import Foundation

class OptionsTradingService {
    static let API_BASE = "http://localhost:8443/api/v1"
    var authToken: String?
    
    init(token: String?) { self.authToken = token }
    
    private var headers: [String: String] {
        var h = ["Content-Type": "application/json"]
        if let token = authToken { h["Authorization"] = "Bearer \(token)" }
        return h
    }
    
    func getOptions(completion: @escaping ([OptionContract]) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/options/contracts") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        request.allHTTPHeaderFields = headers
        
        URLSession.shared.dataTask(with: request) { data, _, _ in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let array = json["data"] as? [[String: Any]] else {
                completion([])
                return
            }
            let contracts = array.compactMap { dict -> OptionContract? in
                guard let id = dict["id"] as? String, let symbol = dict["symbol"] as? String else { return nil }
                return OptionContract(id: id, symbol: symbol, strike: dict["strike"] as? Double ?? 0, premium: dict["premium"] as? Double ?? 0)
            }
            completion(contracts)
        }.resume()
    }
    
    func buyOption(contractId: String, quantity: Int, completion: @escaping (Bool) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/options/buy") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.allHTTPHeaderFields = headers
        request.httpBody = try? JSONSerialization.data(withJSONObject: ["contractId": contractId, "quantity": quantity])
        
        URLSession.shared.dataTask(with: request) { _, response, _ in
            if let http = response as? HTTPURLResponse { completion(http.statusCode == 201) }
            else { completion(false) }
        }.resume()
    }
}

struct OptionContract {
    let id: String
    let symbol: String
    let strike: Double
    let premium: Double
}

class CopyTradingService {
    static let API_BASE = "http://localhost:8443/api/v1"
    var authToken: String?
    
    init(token: String?) { self.authToken = token }
    
    private var headers: [String: String] {
        var h = ["Content-Type": "application/json"]
        if let token = authToken { h["Authorization"] = "Bearer \(token)" }
        return h
    }
    
    func getTopTraders(completion: @escaping ([Trader]) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/copy/traders") else { return }
        var request = URLRequest(url: url)
        request.allHTTPHeaderFields = headers
        
        URLSession.shared.dataTask(with: request) { data, _, _ in
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let array = json["data"] as? [[String: Any]] else {
                completion([])
                return
            }
            let traders = array.compactMap { dict -> Trader? in
                guard let id = dict["id"] as? String, let name = dict["name"] as? String else { return nil }
                return Trader(id: id, name: name, pnl: dict["pnl"] as? Double ?? 0, followers: dict["followers"] as? Double ?? 0)
            }
            completion(traders)
        }.resume()
    }
    
    func followTrader(traderId: String, amount: Double, completion: @escaping (Bool) -> Void) {
        guard let url = URL(string: "\(Self.API_BASE)/copy/follow") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.allHTTPHeaderFields = headers
        request.httpBody = try? JSONSerialization.data(withJSONObject: ["traderId": traderId, "amount": amount])
        
        URLSession.shared.dataTask(with: request) { _, response, _ in
            if let http = response as? HTTPURLResponse { completion(http.statusCode == 201) }
            else { completion(false) }
        }.resume()
    }
}

struct Trader {
    let id: String
    let name: String
    let pnl: Double
    let followers: Double
}
